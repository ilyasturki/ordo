package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Project
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import java.io.File
import java.nio.file.Files
import java.time.Instant

private fun entry(name: String, updated: String) = Entry(name, name, "", "", Project(name, name, updated))

private fun Instant.ms() = toEpochMilli()

class RankTest {
    private val jan = "2026-01-01T00:00:00Z"
    private val feb = "2026-02-01T00:00:00Z"
    private val mar = "2026-03-01T00:00:00Z"
    private val apr = Instant.parse("2026-04-01T00:00:00Z").ms()

    private fun names(entries: List<Entry>, opened: Map<String, Long> = emptyMap()) = rankProjects(entries, opened).map { it.name }

    @Test
    fun aProjectOpenedHereAfterItsLastEditLeads() {
        assertEquals(listOf("Alpha", "Beta"), names(listOf(entry("Beta", mar), entry("Alpha", feb)), mapOf("Alpha" to apr)))
    }

    @Test
    fun aProjectEditedElsewhereAfterItWasOpenedHereLeads() {
        val opened = mapOf("Alpha" to Instant.parse(jan).ms(), "Beta" to Instant.parse(feb).ms())
        assertEquals(listOf("Alpha", "Beta"), names(listOf(entry("Beta", feb), entry("Alpha", mar)), opened))
    }

    @Test
    fun aProjectNeverOpenedHereRanksByItsLastEdit() {
        assertEquals(listOf("Gamma", "Beta", "Alpha"), names(listOf(entry("Alpha", jan), entry("Beta", feb), entry("Gamma", mar))))
    }

    @Test
    fun aTieFallsToTheName() {
        assertEquals(listOf("Alpha", "beta", "Cedar"), names(listOf(entry("beta", feb), entry("Cedar", feb), entry("Alpha", feb))))
    }

    @Test
    fun anUnreadableEditTimeRanksLast() {
        assertEquals(listOf("Beta", "Alpha"), names(listOf(entry("Alpha", ""), entry("Beta", jan))))
    }
}

// The tab order across a sync: ranked at boot and on a return to the front, frozen in between.
class OrderTest {
    private lateinit var server: Server
    private lateinit var dir: File
    private val cores = mutableListOf<PipeCore>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val quiet = Delays(sync = 600_000, poll = 600_000, firstBackoff = 600_000, maxBackoff = 600_000, reconnect = 600_000)

    // Ahead of every edit time the server stamps, so an open here outranks an edit.
    private var now = System.currentTimeMillis() + 3_600_000

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

    private fun engine(): SyncEngine {
        val core = PipeCore().also(cores::add)
        return SyncEngine(Store(dir), { core }, scope, "test phone", quiet, { now++ }, { Api(it.server, it.token) })
    }

    private fun SyncEngine.names() = state.value.projects.map { it.name }

    private fun SyncEngine.id(name: String) = state.value.projects.first { it.name == name }.id

    @Test
    fun theTabsStayPutWhileTheAppIsInFrontAndRankAgainOnReturn() = runBlocking {
        withTimeout(90_000) {
            server.ordo("project", "add", "Alpha", "-q")
            server.ordo("project", "add", "Bravo", "-q")
            val e = engine()
            e.boot()
            assertNull(e.pair(server.pairLink()))
            val first = e.names()
            assertEquals(setOf("Alpha", "Bravo", "Ordo"), first.toSet())

            // Opening, editing and a project that appears on the server move nothing.
            val ordo = e.id("Ordo")
            e.setCurrent(ordo)
            e.edit(e.id("Alpha"), Ops.projectRename("Alpha two"), quiet = true)
            delay(1_100)
            server.ordo("project", "add", "Charlie", "-q")
            e.sync()
            assertEquals(first.map { if (it == "Alpha") "Alpha two" else it } + "Charlie", e.names())

            // Back from the background: the project on screen and the one edited here lead, then what changed on the server.
            e.foreground(false)
            e.setCurrent(ordo)
            e.foreground(true)
            assertEquals(listOf("Ordo", "Alpha two", "Charlie"), e.names().take(3))

            // A cold start ranks from what was stored.
            val cold = engine()
            cold.boot()
            assertEquals(e.names(), cold.names())
        }
    }
}
