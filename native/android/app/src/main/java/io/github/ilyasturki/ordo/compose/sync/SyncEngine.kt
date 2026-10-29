package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.core.CoreApi
import io.github.ilyasturki.ordo.compose.core.perform
import io.github.ilyasturki.ordo.compose.core.replay
import io.github.ilyasturki.ordo.compose.data.Prefs
import io.github.ilyasturki.ordo.compose.data.Queued
import io.github.ilyasturki.ordo.compose.data.Session
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.data.StoredProject
import io.github.ilyasturki.ordo.compose.model.Answer
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Project
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Deferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import okhttp3.sse.EventSource
import java.text.Collator

data class Entry(
    val id: String,
    val name: String,
    val stamp: String,
    val raw: String,
    val project: Project,
    val error: String? = null,
)

// What keeps edits from reaching the server, once it is worth telling: never a single blip.
sealed interface Trouble {
    data class Failing(val since: Long, val message: String, val offline: Boolean) : Trouble
    data class Refused(val message: String) : Trouble
    data class Outdated(val message: String) : Trouble
    data object Expired : Trouble
}

data class SyncState(
    val booted: Boolean = false,
    val session: Session? = null,
    val projects: List<Entry> = emptyList(),
    val prefs: Prefs = Prefs(),
    val pending: Int = 0,
    val syncing: Boolean = false,
    val lastSync: Long = 0,
    val trouble: Trouble? = null,
) {
    val signedIn: Boolean get() = session != null && !session.expired

    fun project(id: String): Entry? = projects.firstOrNull { it.id == id }
}

class Snack(val message: String, val action: String? = null, val run: (suspend () -> Unit)? = null)

enum class Round { OK, RETRY, STOP }

data class Delays(
    val sync: Long = 400,
    val poll: Long = 30_000,
    val firstBackoff: Long = 2_000,
    val maxBackoff: Long = 60_000,
    val reconnect: Long = 5_000,
    val failuresBeforeTrouble: Int = 3,
    val graceBeforeTrouble: Long = 3 * 60_000,
)

// The phone's copy of every project and the queue of edits made on it, synced with ordo serve as
// app/src/lib/state.svelte.ts does. One lane runs every core call and state change in order.
class SyncEngine(
    private val store: Store,
    private val core: CoreApi,
    private val scope: CoroutineScope,
    private val deviceName: String,
    private val delays: Delays = Delays(),
    private val clock: () -> Long = System::currentTimeMillis,
    private val api: (Session) -> Api = { Api(it.server, it.token) },
    private val background: () -> Unit = {},
    private val io: CoroutineDispatcher = Dispatchers.IO,
) {
    @OptIn(kotlinx.coroutines.ExperimentalCoroutinesApi::class)
    private val lane = Dispatchers.Default.limitedParallelism(1)

    private val _state = MutableStateFlow(SyncState())
    val state: StateFlow<SyncState> = _state.asStateFlow()

    private val _snacks = MutableSharedFlow<Snack>(extraBufferCapacity = 16)
    val snacks: SharedFlow<Snack> = _snacks.asSharedFlow()

    private val booted = CompletableDeferred<Unit>()
    private var queue: List<Queued> = emptyList()
    private var seq = 0L
    private val opened = HashSet<String>()
    private var timer: Job? = null
    private var inflight: Deferred<Round>? = null
    private var again = false
    private var backoff = 0L
    private var failures = 0
    private var failingSince = 0L
    private var foreground = false
    private var stream: EventSource? = null
    private var streamGen = 0

    private val byName = Collator.getInstance().apply { strength = Collator.PRIMARY }
        .let { c -> Comparator<Entry> { a, b -> c.compare(a.name, b.name) } }

    private inline fun publish(f: SyncState.() -> SyncState) = _state.update(f)

    private val session: Session? get() = _state.value.session

    suspend fun boot() = withContext(lane) {
        if (booted.isCompleted) return@withContext
        val session = store.session()
        val entries = store.projects().mapNotNull { (meta, raw) ->
            runCatching { Entry(meta.id, meta.name, meta.stamp, raw, Project.parse(raw), meta.error) }.getOrNull()
        }
        queue = store.queue()
        seq = queue.lastOrNull()?.seq ?: 0
        publish {
            copy(
                booted = true,
                session = session,
                projects = entries.sortedWith(byName),
                prefs = store.prefs(),
                pending = queue.size,
                trouble = if (session?.expired == true) Trouble.Expired else null,
            )
        }
        booted.complete(Unit)
        if (session?.expired == false) scope.launch { sync() }
    }

    suspend fun awaitBoot() = booted.await()

    private fun ensureOpen(e: Entry) {
        if (e.id in opened) return
        try {
            core.open(e.raw)
            opened += e.id
        } catch (_: Exception) {
            // The core refuses the copy: the edit then answers gone.
        }
    }

    // Runs a user's edit on the phone's copy and queues what the server replays.
    suspend fun edit(pid: String, op: JsonObject, quiet: Boolean = false): Answer = withContext(lane) {
        _state.value.project(pid)?.let(::ensureOpen)
        val a = core.perform(pid, op)
        val queued = a.queue
        if (!a.applied || queued == null) {
            when {
                a.status == "conflict" && a.theirs != null ->
                    snack(a.msg, "Keep mine") { edit(pid, op.with("base", a.theirs)) }
                a.status != "unchanged" && a.msg != "" -> snack(a.msg)
            }
            return@withContext a
        }
        land(pid, listOf(queued))
        if (!quiet && a.msg != "") {
            val undo = a.undo
            if (undo.isEmpty()) snack(a.msg) else snack(a.msg, "Undo") { replay(pid, undo) }
        }
        a
    }

    // Applies ops as they are, an undo's, and queues those that changed something.
    suspend fun replay(pid: String, ops: List<JsonObject>) = withContext(lane) {
        _state.value.project(pid)?.let(::ensureOpen)
        val landed = ops.filter { core.replay(pid, it).applied }
        if (landed.isNotEmpty()) land(pid, landed)
    }

    suspend fun createProject(name: String): String? {
        val pid = Ops.newId()
        return if (edit(pid, Ops.projectCreate(name), quiet = true).applied) pid else null
    }

    private fun land(pid: String, ops: List<JsonObject>) {
        val queued = ops.map { Queued(++seq, Ops.newId(), pid, it) }
        queue = queue + queued
        if (ops.any { it.action == "project.delete" }) {
            core.close(pid)
            opened -= pid
            publish { copy(projects = projects.filter { it.id != pid }, pending = queue.size) }
            store.saveQueue(queue)
            store.deleteProject(pid)
        } else {
            val raw = core.project(pid).ifEmpty { error("The project is not open") }
            val p = Project.parse(raw)
            val prev = _state.value.project(pid)
            opened += pid
            val e = Entry(pid, p.name, prev?.stamp.orEmpty(), raw, p, prev?.error)
            publish { copy(projects = (projects.filter { it.id != pid } + e).sortedWith(byName), pending = queue.size) }
            // The queue first: an op that is queued brings the server's copy back, a copy saved without its op stays behind.
            store.saveQueue(queue)
            store.saveProject(StoredProject(pid, p.name, e.stamp, e.error), raw)
        }
        schedule(delays.sync)
        background()
    }

    private fun schedule(ms: Long) {
        timer?.cancel()
        timer = null
        if (ms < 0 || !_state.value.signedIn) return
        timer = scope.launch {
            delay(ms)
            sync()
        }
    }

    // One round at a time; a round asked for meanwhile runs once after it.
    suspend fun sync(): Round {
        val round = withContext(lane) {
            inflight?.also { again = true } ?: scope.async(lane) {
                try {
                    run()
                } finally {
                    inflight = null
                    if (again) {
                        again = false
                        scope.launch { sync() }
                    }
                }
            }.also { inflight = it }
        }
        return round.await()
    }

    private suspend fun run(): Round {
        val s = session
        if (s == null || s.expired) return Round.STOP
        timer?.cancel()
        publish { copy(syncing = true) }
        try {
            val sent = queue
            val last = sent.lastOrNull()?.seq ?: seq
            val busy = sent.mapTo(HashSet()) { it.project }
            val have = _state.value.projects
                .filter { it.id !in busy && it.stamp != "" && it.error == null }
                .associate { it.id to it.stamp }
            val answer = withContext(io) { api(s).sync(sent.map { it.op.withIds(it.qid, it.project) }, have) }
            if (session != s) return Round.STOP
            adopt(answer.projects, last)
            report(sent, answer.results)
            backoff = 0
            failures = 0
            failingSince = 0
            publish { copy(lastSync = clock(), trouble = null) }
            schedule(if (queue.isNotEmpty()) delays.sync else if (foreground) delays.poll else -1)
            return Round.OK
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            return if (session == s) failed(e) else Round.STOP
        } finally {
            publish { copy(syncing = false) }
        }
    }

    private fun failed(e: Exception): Round {
        if (e is ServerError && e.status == 401) {
            expire()
            return Round.STOP
        }
        val now = clock()
        failures++
        if (failingSince == 0L) failingSince = now
        val message = e.message.orEmpty()
        val trouble = when {
            e is ServerError && e.status == 409 -> Trouble.Outdated(message)
            e is ServerError && e.status in 400..499 -> Trouble.Refused(message)
            queue.isNotEmpty() && (failures >= delays.failuresBeforeTrouble || now - failingSince >= delays.graceBeforeTrouble) ->
                Trouble.Failing(failingSince, message, e is Offline)
            else -> _state.value.trouble
        }
        publish { copy(trouble = trouble) }
        if (trouble is Trouble.Outdated) return Round.STOP
        backoff = minOf(delays.maxBackoff, if (backoff == 0L) delays.firstBackoff else backoff * 2)
        schedule(backoff)
        if (queue.isNotEmpty()) background()
        return Round.RETRY
    }

    // Takes the server's copy of each project, with the edits queued since the request put back on top.
    private fun adopt(states: List<ProjectState>, last: Long) {
        queue = queue.filter { it.seq > last }
        val later = queue
        val onServer = states.mapTo(HashSet()) { it.id }
        val mine = _state.value.projects
        val next = ArrayList<Entry>()
        val saves = ArrayList<Entry>()
        for (st in states) {
            val held = mine.firstOrNull { it.id == st.id }
            val raw = st.raw()
            if (raw == null) {
                if (held != null) {
                    val e = held.copy(error = st.error)
                    next += e
                    if (st.error != held.error) saves += e
                }
                continue
            }
            try {
                core.open(raw)
            } catch (err: Exception) {
                val e = held?.copy(error = err.message ?: "Can't open this project") ?: continue
                next += e
                saves += e
                continue
            }
            opened += st.id
            for (q in later) if (q.project == st.id) core.replay(st.id, q.op)
            val merged = core.project(st.id)
            val p = Project.parse(merged)
            val e = Entry(st.id, p.name, st.stamp, merged, p)
            next += e
            saves += e
        }
        val createdSince = later.filter { it.op.action == "project.create" }.mapTo(HashSet()) { it.project }
        val gone = ArrayList<String>()
        for (p in mine) {
            if (p.id in onServer) continue
            if (p.id in createdSince) {
                next += p
                continue
            }
            core.close(p.id)
            opened -= p.id
            gone += p.id
        }
        publish { copy(projects = next.sortedWith(byName), pending = queue.size) }
        store.saveQueue(queue)
        for (e in saves) store.saveProject(StoredProject(e.id, e.name, e.stamp, e.error), e.raw)
        for (id in gone) store.deleteProject(id)
    }

    private fun report(sent: List<Queued>, results: List<SyncResult>) {
        for (r in results) {
            val q = sent.firstOrNull { it.qid == r.id } ?: continue
            if (!r.isProblem()) continue
            val theirs = r.theirs
            if (r.status == "conflict" && theirs != null) {
                val mine = q.op.with("base", theirs)
                snack(r.msg.ifEmpty { "Changed on another device" }, "Keep mine") { edit(q.project, mine) }
            } else if (r.msg != "") {
                snack(r.msg)
            }
        }
    }

    private fun snack(message: String, action: String? = null, run: (suspend () -> Unit)? = null) {
        _snacks.tryEmit(Snack(message, action, run))
    }

    // The app came to the front or left it: the front polls and keeps the live stream open.
    suspend fun foreground(on: Boolean) {
        withContext(lane) {
            foreground = on
            listen(on)
            if (!on && queue.isEmpty()) {
                timer?.cancel()
                timer = null
            }
        }
        if (on) sync()
    }

    // The network came back.
    suspend fun online() {
        if (withContext(lane) { foreground }) sync()
    }

    private fun listen(on: Boolean) {
        stream?.cancel()
        stream = null
        val gen = ++streamGen
        val s = session
        if (!on || s == null || s.expired) return
        stream = api(s).listen(
            changed = { scope.launch(lane) { if (gen == streamGen) schedule(delays.sync) } },
            closed = {
                scope.launch {
                    delay(delays.reconnect)
                    withContext(lane) { if (gen == streamGen && foreground) listen(true) }
                }
            },
        )
    }

    suspend fun pair(link: String): String? {
        val p = parsePairing(link) ?: return "Paste the whole pairing link, https://…/pair/…"
        return try {
            val token = withContext(io) { api(Session(p.server, "")).pair(p.code, deviceName) }
            signIn(Session(p.server, token))
            null
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            e.message ?: "Can't pair"
        }
    }

    suspend fun signIn(next: Session) {
        withContext(lane) {
            val previous = session
            if (previous != null && previous.server != next.server) wipe()
            store.saveSession(next)
            backoff = 0
            failures = 0
            failingSince = 0
            publish { copy(session = next, trouble = null) }
            if (foreground) listen(true)
        }
        sync()
    }

    private fun expire() {
        listen(false)
        val s = session ?: return
        val next = s.copy(expired = true)
        store.saveSession(next)
        timer?.cancel()
        publish { copy(session = next, trouble = Trouble.Expired) }
    }

    suspend fun signOut() {
        val s = withContext(lane) {
            listen(false)
            session
        }
        if (s != null && !s.expired) {
            try {
                withContext(io) { api(s).logout() }
            } catch (_: Exception) {
                // Signed out on the phone either way.
            }
        }
        withContext(lane) {
            wipe()
            store.saveSession(null)
            timer?.cancel()
            publish { copy(session = null, trouble = null, lastSync = 0) }
        }
    }

    private fun wipe() {
        store.wipe()
        for (id in opened) core.close(id)
        opened.clear()
        queue = emptyList()
        publish { copy(projects = emptyList(), pending = 0, prefs = Prefs()) }
    }

    suspend fun setCurrent(pid: String) = withContext(lane) {
        val prefs = _state.value.prefs.copy(current = pid)
        publish { copy(prefs = prefs) }
        store.savePrefs(prefs)
    }

    suspend fun setFilter(pid: String, filter: Filter) = withContext(lane) {
        val prefs = _state.value.prefs.let { it.copy(filters = it.filters + (pid to filter)) }
        publish { copy(prefs = prefs) }
        store.savePrefs(prefs)
    }
}

private fun JsonObject.with(key: String, value: String): JsonObject = JsonObject(this + (key to JsonPrimitive(value)))
