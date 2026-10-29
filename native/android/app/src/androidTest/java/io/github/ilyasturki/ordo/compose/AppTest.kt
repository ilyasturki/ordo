package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasParent
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithContentDescription
import androidx.compose.ui.test.onFirst
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.performTextInput
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.data.Session
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.data.StoredProject
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Task
import io.github.ilyasturki.ordo.compose.sync.Delays
import io.github.ilyasturki.ordo.compose.sync.SyncEngine
import io.github.ilyasturki.ordo.compose.ui.App
import io.github.ilyasturki.ordo.compose.ui.OrdoTheme
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.job
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.io.File

private const val PID = "0rd0-web-fixture"

// The app on the seed project, signed in to a server that never answers, so nothing leaves the phone.
@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class AppTest {
    @get:Rule
    val compose = createComposeRule()

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private lateinit var engine: SyncEngine
    private lateinit var dir: File

    @Before
    fun setUp() {
        val ins = InstrumentationRegistry.getInstrumentation()
        val raw = ins.context.assets.open("seed.json").bufferedReader().use { it.readText() }
        ins.targetContext.cacheDir.listFiles { f -> f.name.startsWith("ui-") }?.forEach { it.deleteRecursively() }
        dir = File(ins.targetContext.cacheDir, "ui-${System.nanoTime()}")
        val store = Store(dir)
        store.saveSession(Session("http://127.0.0.1:9", "token"))
        store.saveProject(StoredProject(PID, "Ordo", "stamp"), raw)
        engine = SyncEngine(
            store, ::GoCore, scope, "test",
            Delays(sync = 50, poll = 600_000, firstBackoff = 100, maxBackoff = 200, reconnect = 600_000),
        )
        runBlocking {
            engine.boot()
            engine.foreground(true)
        }
        compose.setContent { OrdoTheme { App(engine, scope) } }
    }

    // The next test's setUp deletes the files: this one's screen may still be writing until the rule disposes it.
    @After
    fun tearDown() = runBlocking { scope.coroutineContext.job.cancelAndJoin() }

    private fun task(title: String): Task? =
        engine.state.value.project(PID)?.project?.categories?.flatMap { it.tasks }?.firstOrNull { it.title == title }

    private fun scrollTo(text: String) {
        compose.onNodeWithTag("list").performScrollToNode(hasText(text, substring = true))
    }

    private fun statusOf(title: String, status: String): SemanticsMatcher =
        hasContentDescription("$status, change status") and hasParent(hasText(title, substring = true))

    @Test
    fun theFilterCountsWhatItHidesAndAPeekShowsIt() {
        runBlocking { engine.setFilter(PID, Filter(statuses = setOf("todo"))) }
        scrollTo("3 hidden by the filter")
        compose.onNodeWithText("Write the install guide").assertDoesNotExist()
        compose.onNodeWithText("3 hidden by the filter").performClick()
        scrollTo("Hide the 3 filtered out")
        compose.onNodeWithText("Write the install guide", substring = true).assertExists()
        compose.onNodeWithText("Hide the 3 filtered out").performClick()
        compose.onNodeWithText("Write the install guide").assertDoesNotExist()
        compose.onNodeWithText("3 hidden by the filter").assertExists()
    }

    @Test
    fun aFinishedRowStaysWhereItIs() {
        val before = engine.state.value.project(PID)!!.project.categories[0].tasks.map { it.id }
        scrollTo("Export a project as Markdown")
        compose.onNode(statusOf("Export a project as Markdown", "To do")).performClick()
        compose.waitUntil(5_000) { task("Export a project as Markdown")?.status == "in_progress" }
        compose.onNode(statusOf("Export a project as Markdown", "Doing")).performClick()
        compose.waitUntil(5_000) { task("Export a project as Markdown")?.status == "completed" }
        compose.onNode(statusOf("Export a project as Markdown", "Done")).assertExists()
        assertEquals(before, engine.state.value.project(PID)!!.project.categories[0].tasks.map { it.id })
    }

    @Test
    fun theAddSheetPutsTheTaskFirstInItsCategory() {
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNode(hasSetTextAction()).performTextInput("Water the plants")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.waitUntil(5_000) { task("Water the plants") != null }
        assertEquals("Water the plants", engine.state.value.project(PID)!!.project.categories[0].tasks.first().title)
    }

    @Test
    fun theCloudShowsOnlyOnceAnEditWaitsThroughFailures() {
        Thread.sleep(1_500)
        compose.onNodeWithContentDescription("Not synced").assertDoesNotExist()
        scrollTo("Recurring tasks")
        compose.onNode(statusOf("Recurring tasks", "To do")).performClick()
        compose.waitUntil(15_000) { compose.onAllNodesWithContentDescription("Not synced").fetchSemanticsNodes().isNotEmpty() }
        compose.onNodeWithContentDescription("Not synced").performClick()
        compose.onNodeWithText("Can't reach your server").assertIsDisplayed()
        compose.onNodeWithText("1 edit waits safely on this phone.").assertIsDisplayed()
        compose.onNodeWithText("Retry").assertIsDisplayed()
    }

    @Test
    fun theTaskPageShowsWhatIsSetAndMarksDone() {
        scrollTo("Tap a row to focus it")
        compose.onNodeWithText("Tap a row to focus it").performClick()
        compose.onNodeWithContentDescription("Mark done").assertIsDisplayed()
        compose.onNodeWithText("Medium").assertExists()
        compose.onNodeWithText("Green").assertExists()
        compose.onNodeWithText("2h").assertExists()
        compose.onNodeWithContentDescription("Status: To do").assertExists()
        compose.onNodeWithText("Add details").assertExists()
        compose.onNodeWithContentDescription("Mark done").performClick()
        compose.waitUntil(5_000) { task("Tap a row to focus it")?.status == "completed" }
        compose.waitUntil(5_000) { compose.onAllNodesWithContentDescription("Add a task").fetchSemanticsNodes().isNotEmpty() }
    }

    @Test
    fun deletingACategoryAsksFirst() {
        scrollTo("Research")
        compose.onNodeWithText("Research").performClick()
        compose.onNodeWithText("Delete category").performClick()
        compose.onNodeWithText("Delete “Research”?").assertIsDisplayed()
        compose.onNodeWithText("Cancel").performClick()
        compose.onNodeWithText("Delete “Research”?").assertDoesNotExist()
        assertNotNull(engine.state.value.project(PID)!!.project.categories.firstOrNull { it.name == "Research" })
        compose.onAllNodesWithContentDescription("Add a task").onFirst().assertExists()
    }
}
