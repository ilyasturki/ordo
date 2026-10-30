package io.github.ilyasturki.ordo.compose

import android.view.KeyEvent
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
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
import androidx.test.espresso.Espresso
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage
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
        compose.waitUntilAtLeastOneExists(hasContentDescription("More"), 20_000)
    }

    private fun pageHasFocus(): Boolean {
        var focused = false
        InstrumentationRegistry.getInstrumentation().runOnMainSync {
            val top = ActivityLifecycleMonitorRegistry.getInstance().getActivitiesInStage(Stage.RESUMED).singleOrNull()
            focused = top is PageActivity && top.hasWindowFocus()
        }
        return focused
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
        compose.onNode(statusOf("Export a project as Markdown", "Todo")).performClick()
        compose.waitUntil(20_000) { task("Export a project as Markdown")?.status == "in_progress" }
        compose.onNode(statusOf("Export a project as Markdown", "In Progress")).performClick()
        compose.waitUntil(20_000) { task("Export a project as Markdown")?.status == "completed" }
        compose.onNode(statusOf("Export a project as Markdown", "Completed")).assertExists()
        assertEquals(before, engine.state.value.project(PID)!!.project.categories[0].tasks.map { it.id })
    }

    @Test
    fun aFinishedRowKeepsItsMarks() {
        scrollTo("Shared action layer")
        assertEquals("completed", task("Shared action layer")?.status)
        compose.onNodeWithText("1d", useUnmergedTree = true).assertExists()
    }

    @Test
    fun aSeparatorReadsAsTyped() {
        scrollTo("Later")
        compose.onNodeWithText("Later").assertExists()
    }

    @Test
    fun aSeparatorLandsFirstInItsCategoryWithNoLabelNeeded() {
        scrollTo("Research")
        compose.onNodeWithText("Research").performClick()
        compose.onNodeWithText("Add a separator").performClick()
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }.tasks.first().separator }
        assertEquals("", engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }.tasks.first().title)
    }

    @Test
    fun theFilterRefusesASeparator() {
        runBlocking { engine.setFilter(PID, Filter(statuses = setOf("todo"))) }
        scrollTo("Research")
        compose.onNodeWithText("Research").performClick()
        compose.onNodeWithText("Add a separator").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Clear the filter to add a separator"), 20_000)
    }

    @Test
    fun theFilterRefusesATaskItWouldHide() {
        runBlocking { engine.setFilter(PID, Filter(priorities = setOf("medium"))) }
        scrollTo("hidden by the filter")
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNodeWithText("Clear the filter to add a task here").assertExists()
        compose.onNode(hasSetTextAction()).performTextInput("Water the plants")
        compose.onNodeWithText("Save").assertIsNotEnabled()
        compose.onNodeWithContentDescription("Priority").performClick()
        compose.onNodeWithText("Medium").performClick()
        compose.onNodeWithText("Clear the filter to add a task here").assertDoesNotExist()
        compose.onNodeWithText("Save").performClick()
        compose.waitUntil(20_000) { task("Water the plants")?.priority == "medium" }
    }

    @Test
    fun theAddSheetPutsTheTaskFirstInItsCategory() {
        scrollTo("Tap a row to focus it")
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNode(hasSetTextAction()).performTextInput("Water the plants")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        assertEquals("Water the plants", engine.state.value.project(PID)!!.project.categories[0].tasks.first().title)
    }

    @Test
    fun theCloudShowsOnlyOnceAnEditWaitsThroughFailures() {
        Thread.sleep(1_500)
        compose.onNodeWithContentDescription("Not synced").assertDoesNotExist()
        scrollTo("Recurring tasks")
        compose.onNode(statusOf("Recurring tasks", "Todo")).performClick()
        compose.waitUntil(15_000) { compose.onAllNodesWithContentDescription("Not synced").fetchSemanticsNodes().isNotEmpty() }
        compose.onNodeWithContentDescription("Not synced").performClick()
        compose.onNodeWithText("Can't reach your server").assertIsDisplayed()
        compose.onNodeWithText("1 edit waits safely on this phone.").assertIsDisplayed()
        compose.onNodeWithText("Retry").assertIsDisplayed()
    }

    @Test
    fun theTaskPageShowsWhatIsSetAndMarkDoneStaysOnIt() {
        openTask("Tap a row to focus it")
        compose.onNodeWithContentDescription("Mark done").assertIsDisplayed()
        compose.onNodeWithText("Medium").assertExists()
        compose.onNodeWithText("Green").assertExists()
        compose.onNodeWithText("2h").assertExists()
        compose.onNodeWithContentDescription("Status: Todo").assertExists()
        compose.onNodeWithText("Add a description").assertExists()
        compose.onNodeWithContentDescription("Mark done").performClick()
        compose.waitUntil(20_000) { task("Tap a row to focus it")?.status == "completed" }
        compose.waitUntilAtLeastOneExists(hasContentDescription("Mark todo"), 20_000)
        compose.onNodeWithContentDescription("More").assertIsDisplayed()
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(20_000) { task("Tap a row to focus it")?.status == "todo" }
    }

    @Test
    fun theBackKeyClosesThePage() {
        openTask("Tap a row to focus it")
        val list = hasContentDescription("Add a task")
        // A key goes to the focused window, the page's only once it has opened; each try waits for that, so none reaches the list.
        compose.waitUntil(30_000) {
            if (pageHasFocus()) InstrumentationRegistry.getInstrumentation().sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
            Thread.sleep(2_000)
            compose.onAllNodes(list).fetchSemanticsNodes().isNotEmpty()
        }
    }

    @Test
    fun whatWasTypedLandsAsThePageCloses() {
        openTask("Tap a row to focus it")
        compose.onNodeWithText("Tap a row to focus it").performTextReplacement("Tap a row to edit it")
        compose.onNodeWithContentDescription("Back").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 20_000)
        compose.waitUntil(20_000) { task("Tap a row to edit it") != null }
        assertNull(task("Tap a row to focus it"))
    }

    @Test
    fun aTaskDeletedOnItsPageOffersUndoOnTheList() {
        openTask("Tap a row to focus it")
        compose.onNodeWithContentDescription("More").performClick()
        compose.onNodeWithText("Delete").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 10_000)
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(20_000) { task("Tap a row to focus it") != null }
    }

    @Test
    fun theSearchNarrowsTheListInPlaceAndBackClosesIt() {
        compose.onNodeWithContentDescription("Search").performClick()
        compose.onNode(hasSetTextAction()).performTextInput("pairing")
        compose.onNodeWithText("Pairing link login").assertIsDisplayed()
        compose.onNodeWithText("Recurring tasks").assertDoesNotExist()
        compose.onNodeWithContentDescription("Add a task").assertDoesNotExist()
        compose.onNodeWithText("Pairing link login").performClick()
        compose.onNodeWithContentDescription("Mark done").assertIsDisplayed()
        Espresso.pressBack()
        compose.onNodeWithText("Pairing link login").assertIsDisplayed()
        compose.onNodeWithText("Recurring tasks").assertDoesNotExist()
        compose.onNode(hasSetTextAction()).performTextReplacement("no such task anywhere")
        compose.onNodeWithText("Nothing matches").assertIsDisplayed()
        Espresso.closeSoftKeyboard()
        Espresso.pressBack()
        compose.onNodeWithContentDescription("Search").assertIsDisplayed()
        compose.onNodeWithContentDescription("Add a task").assertIsDisplayed()
        scrollTo("Recurring tasks")
        compose.onNodeWithText("Recurring tasks").assertExists()
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
        compose.onNodeWithText("Delete category “Research”?").assertIsDisplayed()
        compose.onNodeWithText("Its 2 tasks go too.").assertIsDisplayed()
        compose.onNodeWithText("Cancel").performClick()
        compose.onNodeWithText("Delete category “Research”?").assertDoesNotExist()
        assertNotNull(engine.state.value.project(PID)!!.project.categories.firstOrNull { it.name == "Research" })
        compose.onAllNodesWithContentDescription("Add a task").onFirst().assertExists()
    }

    @Test
    fun undoBringsADeletedCategoryBackWhereItWas() {
        val before = engine.state.value.project(PID)!!.project.categories.map { it.name }
        scrollTo("Documentation")
        compose.onNodeWithText("Documentation").performClick()
        compose.onNodeWithText("Delete category").performClick()
        compose.onNodeWithText("Delete category “Documentation”?").assertIsDisplayed()
        compose.onNodeWithText("Its sub-category and 5 tasks go too.").assertIsDisplayed()
        compose.onNodeWithText("Delete").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.none { it.name == "API" } }
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.map { it.name } == before }
    }
}
