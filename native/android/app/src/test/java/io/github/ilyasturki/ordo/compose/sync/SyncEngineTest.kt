package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.model.Ops
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
}
