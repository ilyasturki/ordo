package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.data.Prefs
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.View
import io.github.ilyasturki.ordo.compose.model.Project
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.async
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.File
import java.nio.file.Files
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

private const val PID = "0rd0-web-fixture"
private const val EXPORT = "aaaaaaaa-0000-4000-8000-000000000005"
private const val RECURRING = "aaaaaaaa-0000-4000-8000-000000000007"
private const val DOCUMENTATION = "cccccccc-0000-4000-8000-000000000019"
private const val FEATURE = "cccccccc-0000-4000-8000-000000000008"

// Holds the next /api/sync request until released, so a test can edit while a round is out.
class Gate : Interceptor {
    @Volatile
    var armed = false
    val entered = CountDownLatch(1)
    val release = CountDownLatch(1)

    override fun intercept(chain: Interceptor.Chain): Response {
        if (armed && chain.request().url.encodedPath == "/api/sync") {
            armed = false
            entered.countDown()
            release.await(20, TimeUnit.SECONDS)
        }
        return chain.proceed(chain.request())
    }
}

class SyncEngineTest {
    private lateinit var server: Server
    private lateinit var dir: File
    private val cores = mutableListOf<PipeCore>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val quiet = Delays(
        sync = 600_000,
        poll = 600_000,
        firstBackoff = 600_000,
        maxBackoff = 600_000,
        reconnect = 600_000,
    )

    @Before
    fun setUp() {
        server = Server()
        dir = Files.createTempDirectory("ordo-phone").toFile()
    }

    @After
    fun tearDown() {
        scope.cancel()
        cores.forEach(PipeCore::close)
        server.close()
        dir.deleteRecursively()
    }

    private fun engine(
        http: OkHttpClient = Api.client,
        clock: () -> Long = System::currentTimeMillis,
        delays: Delays = quiet,
    ): SyncEngine {
        val core = PipeCore().also(cores::add)
        return SyncEngine(Store(dir), { core }, scope, "test phone", delays, clock, { Api(it.server, it.token, http) })
    }

    private fun <T> test(block: suspend CoroutineScope.() -> T): T = runBlocking { withTimeout(60_000) { block() } }

    private suspend fun paired(e: SyncEngine): SyncEngine {
        e.boot()
        assertNull(e.pair(server.pairLink()))
        return e
    }

    private fun SyncEngine.task(id: String) = state.value.project(PID)!!.project.find(id)!!.second

    private fun serverTask(id: String) = Project.parse(server.project(PID)).find(id)!!.second

    @Test
    fun pairingAdoptsTheServerProjects() = test {
        val e = paired(engine())
        val s = e.state.value
        assertTrue(s.signedIn)
        assertEquals(listOf("Ordo"), s.projects.map { it.name })
        assertTrue(s.projects[0].stamp.isNotEmpty())
        assertNotNull(Store(dir).session())
    }

    @Test
    fun aSignInWhoseScreenClosesStillSyncs() = test {
        val e = engine()
        e.boot()
        val link = server.pairLink()
        val pairing = launch(Dispatchers.Default) { e.pair(link) }
        e.state.first { it.session != null }
        pairing.cancel()
        withTimeout(10_000) { e.state.first { it.projects.isNotEmpty() } }
        assertEquals(listOf("Ordo"), e.state.value.projects.map { it.name })
    }

    @Test
    fun anEditReachesTheServerFile() = test {
        val e = paired(engine())
        assertTrue(e.edit(PID, Ops.cycle(EXPORT)).applied)
        assertEquals("in_progress", e.task(EXPORT).status)
        assertEquals(1, e.state.value.pending)
        assertEquals(Round.OK, e.sync())
        assertEquals(0, e.state.value.pending)
        assertEquals("in_progress", serverTask(EXPORT).status)
    }

    @Test
    fun aPinReachesTheServerFirstAmongItsSiblings() = test {
        val e = paired(engine())
        val a = e.edit(PID, Ops.categoryPin(DOCUMENTATION, true))
        assertTrue(a.applied)
        assertEquals("Pinned “Documentation”", a.msg)
        assertEquals(DOCUMENTATION, e.state.value.project(PID)!!.project.categories.first().id)
        assertEquals(Round.OK, e.sync())
        val first = Project.parse(server.project(PID)).categories.first()
        assertEquals(DOCUMENTATION, first.id)
        assertTrue(first.pinned)
    }

    @Test
    fun aTakenCategoryNameAnswersTheNamesThatFreeIt() = test {
        val e = paired(engine())
        val a = e.edit(PID, Ops.categoryAdd("feature"), quiet = true)
        assertEquals("refused", a.status)
        assertEquals(FEATURE, a.clash!!.taken)
        assertEquals("feature (2)", a.clash!!.mine)
        assertEquals("Feature (2)", a.clash!!.other)
        assertEquals(0, e.state.value.pending)
    }

    @Test
    fun anEditMadeDuringARoundLandsOnTheServerCopy() = test {
        val gate = Gate()
        val e = paired(engine(Api.client.newBuilder().addInterceptor(gate).build()))
        server.ordo("task", "edit", EXPORT, "-p", PID, "--priority", "high")
        gate.armed = true
        val round = async(Dispatchers.IO) { e.sync() }
        assertTrue(gate.entered.await(20, TimeUnit.SECONDS))
        e.edit(PID, Ops.title(RECURRING, "Recurring tasks, weekly", "Recurring tasks"))
        gate.release.countDown()
        assertEquals(Round.OK, round.await())
        assertEquals("high", e.task(EXPORT).priority)
        assertEquals("Recurring tasks, weekly", e.task(RECURRING).title)
        assertEquals(1, e.state.value.pending)
        assertEquals(Round.OK, e.sync())
        assertEquals("Recurring tasks, weekly", serverTask(RECURRING).title)
    }

    @Test
    fun aProjectDeletedOnTheServerGoesAndOneCreatedSinceStays() = test {
        val gate = Gate()
        val e = paired(engine(Api.client.newBuilder().addInterceptor(gate).build()))
        server.ordo("project", "delete", PID, "-f")
        gate.armed = true
        val round = async(Dispatchers.IO) { e.sync() }
        assertTrue(gate.entered.await(20, TimeUnit.SECONDS))
        val phone = e.createProject("Phone")!!
        gate.release.countDown()
        assertEquals(Round.OK, round.await())
        assertEquals(listOf("Phone"), e.state.value.projects.map { it.name })
        assertEquals(Round.OK, e.sync())
        assertTrue(server.ordo("projects").contains("Phone"))
        assertEquals(phone, e.state.value.projects.single().id)
    }

    @Test
    fun anExpiredSessionKeepsTheCopyAndTheQueue() = test {
        val e = paired(engine())
        server.revoke("test phone")
        e.edit(PID, Ops.cycle(EXPORT))
        assertEquals(Round.STOP, e.sync())
        val s = e.state.value
        assertEquals(Trouble.Expired, s.trouble)
        assertTrue(s.session!!.expired)
        assertEquals(1, s.pending)
        assertEquals("in_progress", e.task(EXPORT).status)
        assertNull(e.pair(server.pairLink()))
        assertEquals(0, e.state.value.pending)
        assertEquals("in_progress", serverTask(EXPORT).status)
    }

    @Test
    fun aProtocolMismatchStopsWithTheServersWords() = test {
        val refuse = Interceptor { chain ->
            if (chain.request().url.encodedPath != "/api/sync") return@Interceptor chain.proceed(chain.request())
            Response.Builder()
                .request(chain.request())
                .protocol(Protocol.HTTP_1_1)
                .code(409)
                .message("Conflict")
                .body("""{"error":"Update the app","protocol":2}""".toResponseBody("application/json".toMediaType()))
                .build()
        }
        val e = paired(engine(Api.client.newBuilder().addInterceptor(refuse).build()))
        assertEquals(Trouble.Outdated("Update the app"), e.state.value.trouble)
        assertEquals(Round.STOP, e.sync())
    }

    @Test
    fun aConflictOffersToKeepMine() = test {
        val e = paired(engine())
        val snacks = Collections.synchronizedList(mutableListOf<Snack>())
        val collecting = launch { e.snacks.collect { snacks += it } }
        e.edit(PID, Ops.title(RECURRING, "Mine", "Recurring tasks"), quiet = true)
        server.ordo("task", "edit", RECURRING, "-p", PID, "--title", "Theirs")
        assertEquals(Round.OK, e.sync())
        assertEquals("Theirs", e.task(RECURRING).title)
        val keep = withTimeout(5_000) {
            while (snacks.none { it.action == "Keep mine" }) kotlinx.coroutines.delay(20)
            snacks.first { it.action == "Keep mine" }
        }
        keep.run!!()
        assertEquals("Mine", e.task(RECURRING).title)
        assertEquals(Round.OK, e.sync())
        assertEquals("Mine", serverTask(RECURRING).title)
        collecting.cancel()
    }

    @Test
    fun theQueueSurvivesARestart() = test {
        paired(engine()).also {
            server.stop()
            it.edit(PID, Ops.cycle(EXPORT))
            assertEquals(Round.RETRY, it.sync())
        }
        val again = engine()
        again.boot()
        assertEquals(1, again.state.value.pending)
        assertEquals("in_progress", again.task(EXPORT).status)
        server.start()
        assertEquals(Round.OK, again.sync())
        assertEquals("in_progress", serverTask(EXPORT).status)
    }

    @Test
    fun troubleWaitsForRepeatedFailuresWithEditsWaiting() = test {
        val e = paired(engine())
        server.stop()
        assertEquals(Round.RETRY, e.sync())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(Round.RETRY, e.sync())
        assertNull("nothing waits: no trouble", e.state.value.trouble)
        e.edit(PID, Ops.cycle(EXPORT))
        assertEquals(Round.RETRY, e.sync())
        val t = e.state.value.trouble as Trouble.Failing
        assertTrue(t.offline)
        server.start()
        assertEquals(Round.OK, e.sync())
        assertNull(e.state.value.trouble)
    }

    @Test
    fun troubleShowsAfterTheGraceEvenOnTheFirstFailures() = test {
        var now = 1_000_000L
        val e = paired(engine(clock = { now }))
        server.stop()
        e.edit(PID, Ops.cycle(EXPORT))
        assertEquals(Round.RETRY, e.sync())
        assertNull(e.state.value.trouble)
        now += quiet.graceBeforeTrouble
        assertEquals(Round.RETRY, e.sync())
        assertTrue(e.state.value.trouble is Trouble.Failing)
    }

    @Test
    fun aFailureRetriesByItselfOnlyInSight() = test {
        val rounds = java.util.concurrent.atomic.AtomicInteger()
        val count = Interceptor { chain ->
            if (chain.request().url.encodedPath == "/api/sync") rounds.incrementAndGet()
            chain.proceed(chain.request())
        }
        val e = paired(engine(Api.client.newBuilder().addInterceptor(count).build(), delays = quiet.copy(firstBackoff = 50, maxBackoff = 50)))
        server.stop()
        val before = rounds.get()
        assertEquals(Round.RETRY, e.sync())
        Thread.sleep(600)
        assertEquals("out of sight, WorkManager retries", before + 1, rounds.get())
        e.foreground(true)
        Thread.sleep(600)
        assertTrue("in sight, the backoff retries", rounds.get() > before + 3)
    }

    @Test
    fun whatWaitsOutOfSightGoesToWorkManager() = test {
        val asked = java.util.concurrent.atomic.AtomicInteger()
        val core = PipeCore().also(cores::add)
        val e = paired(SyncEngine(Store(dir), { core }, scope, "test phone", quiet, background = { asked.incrementAndGet() }))
        e.foreground(true)
        e.edit(PID, Ops.cycle(EXPORT))
        assertEquals("in sight, the app's own timer syncs it", 0, asked.get())
        e.foreground(false)
        assertEquals("leaving with an edit waiting", 1, asked.get())
        server.stop()
        assertEquals(Round.RETRY, e.sync())
        assertEquals("a failed round out of sight", 2, asked.get())
        e.edit(PID, Ops.cycle(RECURRING))
        assertEquals("an edit out of sight", 3, asked.get())
        server.start()
        assertEquals(Round.OK, e.sync())
        e.foreground(false)
        assertEquals("nothing waits, nothing to hand over", 3, asked.get())
    }

    @Test
    fun undoPutsBackWhatAnEditChanged() = test {
        val e = paired(engine())
        val snacks = Collections.synchronizedList(mutableListOf<Snack>())
        val collecting = launch { e.snacks.collect { snacks += it } }
        e.edit(PID, Ops.delete(EXPORT))
        assertNull(e.state.value.project(PID)!!.project.find(EXPORT))
        val undo = withTimeout(5_000) {
            while (snacks.none { it.action == "Undo" }) kotlinx.coroutines.delay(20)
            snacks.first { it.action == "Undo" }
        }
        undo.run!!()
        assertEquals("Export a project as Markdown", e.task(EXPORT).title)
        assertEquals(Round.OK, e.sync())
        assertEquals("Export a project as Markdown", serverTask(EXPORT).title)
        collecting.cancel()
    }

    @Test
    fun undoBringsADeletedCategoryBackOnTheServerToo() = test {
        val e = paired(engine())
        val names = { p: Project -> p.categories.map { it.name } }
        val before = names(e.state.value.project(PID)!!.project)
        val snacks = Collections.synchronizedList(mutableListOf<Snack>())
        val collecting = launch { e.snacks.collect { snacks += it } }
        e.edit(PID, Ops.categoryDelete(DOCUMENTATION))
        assertEquals(Round.OK, e.sync())
        assertEquals(before - "Documentation" - "API", names(Project.parse(server.project(PID))))
        val undo = withTimeout(5_000) {
            while (snacks.none { it.action == "Undo" }) kotlinx.coroutines.delay(20)
            snacks.first { it.action == "Undo" }
        }
        undo.run!!()
        assertEquals(before, names(e.state.value.project(PID)!!.project))
        assertEquals(Round.OK, e.sync())
        val back = Project.parse(server.project(PID))
        assertEquals(before, names(back))
        assertEquals(4, back.categories.first { it.name == "Documentation" }.tasks.size)
        assertEquals("Documentation", back.categories.first { it.id == back.categories.first { c -> c.name == "API" }.parentId }.name)
        collecting.cancel()
    }

    @Test
    fun aDraggedCategoryReachesTheServerAndUndoPutsItBack() = test {
        val e = paired(engine())
        val names = { p: Project -> p.categories.map { it.name } }
        val before = names(e.state.value.project(PID)!!.project)
        val snacks = Collections.synchronizedList(mutableListOf<Snack>())
        val collecting = launch { e.snacks.collect { snacks += it } }
        assertTrue(e.edit(PID, Ops.categoryPlace(DOCUMENTATION, FEATURE)).applied)
        val moved = listOf("Documentation", "API", "Feature", "Fix", "Ergonomy", "Research")
        assertEquals(moved, names(e.state.value.project(PID)!!.project))
        assertEquals(Round.OK, e.sync())
        assertEquals(moved, names(Project.parse(server.project(PID))))
        val undo = withTimeout(5_000) {
            while (snacks.none { it.action == "Undo" }) kotlinx.coroutines.delay(20)
            snacks.first { it.action == "Undo" }
        }
        assertEquals("Moved “Documentation”", undo.message)
        undo.run!!()
        assertEquals(before, names(e.state.value.project(PID)!!.project))
        assertEquals(Round.OK, e.sync())
        assertEquals(before, names(Project.parse(server.project(PID))))
        collecting.cancel()
    }

    @Test
    fun aDraggedCategoryLandsOnAFileChangedSince() = test {
        val e = paired(engine())
        val names = { p: Project -> p.categories.map { it.name } }
        assertTrue(e.edit(PID, Ops.categoryPlace(DOCUMENTATION, FEATURE)).applied)
        server.ordo("category", "delete", FEATURE, "-p", PID, "--force")
        assertEquals(Round.OK, e.sync())
        val landed = listOf("Fix", "Ergonomy", "Research", "Documentation", "API")
        assertEquals(landed, names(Project.parse(server.project(PID))))
        assertEquals(landed, names(e.state.value.project(PID)!!.project))
        assertEquals(0, e.state.value.pending)
    }

    @Test
    fun signingOutWipesThePhoneCopy() = test {
        val e = paired(engine())
        e.edit(PID, Ops.cycle(EXPORT))
        e.signOut()
        val s = e.state.value
        assertNull(s.session)
        assertTrue(s.projects.isEmpty())
        assertEquals(0, s.pending)
        assertNull(Store(dir).session())
        assertTrue(Store(dir).projects().isEmpty())
        assertTrue(server.ordo("serve", "devices").lines().none { it.contains("test phone") })
    }

    @Test
    fun aBrokenLinkSaysWhatToPaste() = test {
        val e = engine()
        e.boot()
        assertEquals("Paste the whole pairing link, https://…/pair/…", e.pair("ABCD"))
        assertTrue(e.pair("${server.url}/pair/NOTACODE")!!.isNotEmpty())
        assertNull(e.state.value.session)
    }

    private fun SyncEngine.view(pid: String = PID) = state.value.views[pid] ?: View()

    // Boot's own round and the test's may run either way round: what a round finds to send goes out with the next.
    private suspend fun settle(e: SyncEngine) {
        repeat(3) { assertEquals(Round.OK, e.sync()) }
    }

    private fun anotherPhone(): SyncEngine {
        val core = PipeCore().also(cores::add)
        return SyncEngine(Store(Files.createTempDirectory("ordo-phone-2").toFile()), { core }, scope, "other phone", quiet)
    }

    @Test
    fun aFilterAndAFoldReachTheFileAndAnotherPhone() = test {
        val e = paired(engine())
        val before = e.state.value.project(PID)!!.stamp
        e.setFilter(PID, Filter(statuses = setOf("todo"), priorities = setOf(""), tags = setOf("", "green")))
        e.setFolded(PID, DOCUMENTATION, true)
        e.setFolded(PID, EXPORT, true)
        assertEquals(1, e.state.value.viewsWaiting)
        assertEquals(Round.OK, e.sync())
        assertEquals(0, e.state.value.viewsWaiting)

        val file = server.view(PID)!!
        val onDisk = kotlinx.serialization.json.Json.parseToJsonElement(file).toString()
        assertTrue(onDisk, onDisk.contains("\"priorities\":[\"none\"]"))
        assertTrue(onDisk, onDisk.contains("\"tags\":[\"green\",\"none\"]"))
        assertTrue(onDisk, onDisk.contains(DOCUMENTATION) && onDisk.contains(EXPORT))
        assertEquals("a view change restamps nothing", before, e.state.value.project(PID)!!.stamp)

        val other = anotherPhone()
        other.boot()
        assertNull(other.pair(server.pairLink()))
        assertEquals(setOf(DOCUMENTATION, EXPORT), other.view().folds)
        assertEquals(setOf(""), other.view().filter.priorities)
        assertEquals(setOf("", "green"), other.view().filter.tags)
        assertEquals(setOf("todo"), other.view().filter.statuses)
    }

    @Test
    fun aViewWrittenBesideTheProjectReachesThePhone() = test {
        val e = paired(engine())
        server.writeView(PID, """{"folds":["$DOCUMENTATION"],"filter":{"priorities":["none","high"],"tags":["none"]}}""")
        assertEquals(Round.OK, e.sync())
        assertEquals(setOf(DOCUMENTATION), e.view().folds)
        assertEquals(setOf("", "high"), e.view().filter.priorities)
        assertEquals(setOf(""), e.view().filter.tags)
        assertEquals(0, e.state.value.viewsWaiting)

        server.writeView(PID, "{}")
        assertEquals(Round.OK, e.sync())
        assertTrue(e.view().empty)
    }

    @Test
    fun aFoldKeepsAFilterSetElsewhere() = test {
        val e = paired(engine())
        assertEquals(Round.OK, e.sync())
        server.writeView(PID, """{"filter":{"statuses":["todo"]}}""")
        e.setFolded(PID, DOCUMENTATION, true)
        assertEquals(Round.OK, e.sync())
        assertEquals(setOf("todo"), e.view().filter.statuses)
        assertEquals(setOf(DOCUMENTATION), e.view().folds)
        val file = server.view(PID)!!
        assertTrue(file, file.contains("todo") && file.contains(DOCUMENTATION))
    }

    @Test
    fun aViewChangeMadeOfflineSurvivesARestartAndGoesOutWhenBack() = test {
        val e = paired(engine())
        server.stop()
        e.setFolded(PID, DOCUMENTATION, true)
        assertEquals(Round.RETRY, e.sync())
        assertEquals(1, e.state.value.viewsWaiting)
        assertEquals(setOf(DOCUMENTATION), e.view().folds)
        assertTrue(e.state.value.waitingWords, e.state.value.waitingWords.contains("fold"))

        val again = engine()
        again.boot()
        assertEquals(setOf(DOCUMENTATION), again.view().folds)
        assertEquals(1, again.state.value.viewsWaiting)

        server.start()
        assertEquals(Round.OK, again.sync())
        assertEquals(0, again.state.value.viewsWaiting)
        assertTrue(server.view(PID)!!.contains(DOCUMENTATION))
    }

    @Test
    fun aViewChangeMadeDuringARoundStaysOnTopOfTheAnswer() = test {
        val gate = Gate()
        val e = paired(engine(Api.client.newBuilder().addInterceptor(gate).build()))
        e.setFolded(PID, DOCUMENTATION, true)
        gate.armed = true
        val round = async(Dispatchers.IO) { e.sync() }
        assertTrue(gate.entered.await(20, TimeUnit.SECONDS))
        e.setFilter(PID, Filter(statuses = setOf("todo")))
        gate.release.countDown()
        assertEquals(Round.OK, round.await())

        assertEquals(setOf("todo"), e.view().filter.statuses)
        assertEquals(setOf(DOCUMENTATION), e.view().folds)
        assertEquals("the filter still waits", 1, e.state.value.viewsWaiting)
        assertEquals(Round.OK, e.sync())
        assertEquals(0, e.state.value.viewsWaiting)
        assertTrue(server.view(PID)!!.contains("todo"))
    }

    @Test
    fun theFiltersOfPrefsMoveOnceAndReachTheServer() = test {
        paired(engine()).also { it.sync() }
        Store(dir).savePrefs(Prefs(current = PID, filters = mapOf(PID to Filter(priorities = setOf("")))))

        val e = engine()
        e.boot()
        assertEquals(setOf(""), e.view().filter.priorities)
        assertEquals(PID, e.state.value.prefs.current)
        assertTrue("moved", Store(dir).prefs().filters.isEmpty())
        assertEquals(PID, Store(dir).prefs().current)

        settle(e)
        assertEquals("the server had none: this phone's goes up", 0, e.state.value.viewsWaiting)
        assertTrue(server.view(PID)!!.contains("none"))
    }

    @Test
    fun aViewFileAlreadyThereWinsOverTheFiltersOfPrefs() = test {
        paired(engine()).also { it.sync() }
        Store(dir).savePrefs(Prefs(current = PID, filters = mapOf(PID to Filter(statuses = setOf("todo")))))
        server.writeView(PID, """{"filter":{"statuses":["completed"]}}""")

        val e = engine()
        e.boot()
        assertEquals(Round.OK, e.sync())
        assertEquals(setOf("completed"), e.view().filter.statuses)
        assertEquals(0, e.state.value.viewsWaiting)
        assertTrue(server.view(PID)!!.contains("completed"))
    }

    @Test
    fun anOldServerLeavesTheViewOnThePhoneAndStopsSending() = test {
        val strip = Interceptor { chain ->
            val res = chain.proceed(chain.request())
            if (chain.request().url.encodedPath != "/api/sync") return@Interceptor res
            val body = kotlinx.serialization.json.Json.parseToJsonElement(res.body.string()) as kotlinx.serialization.json.JsonObject
            res.newBuilder().body(kotlinx.serialization.json.JsonObject(body - "views").toString().toResponseBody("application/json".toMediaType())).build()
        }
        val http = Api.client.newBuilder().addInterceptor(strip).build()
        val e = paired(engine(http))
        e.setFilter(PID, Filter(statuses = setOf("todo")))
        assertEquals(Round.OK, e.sync())
        assertEquals("no round waits for what it can't store", 0, e.state.value.viewsWaiting)
        assertEquals(setOf("todo"), e.view().filter.statuses)

        val updated = engine()
        updated.boot()
        assertEquals(setOf("todo"), updated.view().filter.statuses)
        settle(updated)
        assertEquals(0, updated.state.value.viewsWaiting)
        assertTrue("a newer server has no view: this phone's goes up", server.view(PID)!!.contains("todo"))
    }

    @Test
    fun aChangeToTheViewFileWakesAnOpenApp() = test {
        val e = paired(engine(delays = quiet.copy(sync = 50)))
        e.foreground(true)
        withTimeout(10_000) { e.state.first { it.lastSync > 0L } }
        Thread.sleep(500)
        server.writeView(PID, """{"folds":["$DOCUMENTATION"]}""")
        withTimeout(10_000) { e.state.first { it.views[PID]?.folds == setOf(DOCUMENTATION) } }
        e.foreground(false)
    }

    @Test
    fun signingOutAndDeletingDropTheViews() = test {
        val e = paired(engine())
        val pid = e.createProject("Doomed")!!
        e.setFolded(pid, "c1", true)
        e.edit(pid, Ops.projectDelete())
        assertNull(e.state.value.views[pid])
        e.setFolded(PID, DOCUMENTATION, true)
        e.signOut()
        assertTrue(e.state.value.views.isEmpty())
        assertEquals(0, e.state.value.viewsWaiting)
        assertTrue(Store(dir).views().held.isEmpty())
    }
}
