package io.github.ilyasturki.ordo.compose

import android.view.KeyEvent
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasParent
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.onAllNodesWithContentDescription
import androidx.compose.ui.test.onFirst
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTextReplacement
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.longClick
import androidx.compose.ui.unit.width
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.ilyasturki.ordo.compose.model.Filter
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test
import org.junit.runner.RunWith

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class AppTest : AppFixture() {
    private fun statusOf(title: String, status: String): SemanticsMatcher =
        hasContentDescription("$status, change status") and hasParent(hasText(title, substring = true))

    private fun openTask(title: String) {
        scrollTo(title)
        compose.onNodeWithText(title).performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("More"), 10_000)
    }

    private fun backToList() {
        InstrumentationRegistry.getInstrumentation().sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 10_000)
    }

    @Test
    fun theUnderlineSitsUnderTheOnlyTab() {
        scrollTo("Tap a row to focus it")
        val bar = compose.onNodeWithTag("indicator", useUnmergedTree = true).getUnclippedBoundsInRoot()
        val label = compose.onNodeWithText("Ordo", useUnmergedTree = true).getUnclippedBoundsInRoot()
        assertEquals(label.left.value, bar.left.value, 1f)
        assertEquals(label.width.value, bar.width.value, 1f)
    }

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
        scrollTo("Tap a row to focus it")
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
        openTask("Tap a row to focus it")
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
    fun theBackKeyClosesThePageAndWhatWasTypedLands() {
        openTask("Tap a row to focus it")
        compose.onNodeWithText("Tap a row to focus it").performTextReplacement("Tap a row to edit it")
        backToList()
        compose.waitUntil(5_000) { task("Tap a row to edit it") != null }
        assertNull(task("Tap a row to focus it"))
    }

    @Test
    fun aTaskDeletedOnItsPageOffersUndoOnTheList() {
        openTask("Tap a row to focus it")
        compose.onNodeWithContentDescription("More").performClick()
        compose.onNodeWithText("Delete").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 10_000)
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 5_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(5_000) { task("Tap a row to focus it") != null }
    }

    @Test
    fun aRowLiftedAndLetGoInPlaceStaysShut() {
        scrollTo("Recurring tasks")
        compose.onNodeWithText("Recurring tasks").performTouchInput { longClick() }
        compose.waitForIdle()
        Thread.sleep(1_000)
        compose.onNodeWithContentDescription("Mark done").assertDoesNotExist()
        assertEquals("todo", task("Recurring tasks")?.status)
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
