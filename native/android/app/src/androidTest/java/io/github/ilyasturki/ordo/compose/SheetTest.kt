package io.github.ilyasturki.ordo.compose

import android.view.KeyEvent
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasAnyDescendant
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isPopup
import androidx.compose.ui.test.click
import androidx.compose.ui.test.longClick
import androidx.compose.ui.test.onLast
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTextReplacement
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeDown
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

// The keyboard sheets: every way out keeps what was typed, the chips leave the keyboard up, and the held + adds every kind.
@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class SheetTest : AppFixture() {
    private val pane = SemanticsMatcher.keyIsDefined(SemanticsProperties.PaneTitle) and hasAnyDescendant(hasSetTextAction())
    private val field = hasSetTextAction() and hasAnyAncestor(pane)

    private fun keyboardIsUp(): Boolean {
        var up = false
        InstrumentationRegistry.getInstrumentation().runOnMainSync {
            val top = ActivityLifecycleMonitorRegistry.getInstance().getActivitiesInStage(Stage.RESUMED).singleOrNull()
            up = top != null && ViewCompat.getRootWindowInsets(top.window.decorView)?.isVisible(WindowInsetsCompat.Type.ime()) == true
        }
        return up
    }

    private fun back() {
        InstrumentationRegistry.getInstrumentation().sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
        compose.waitForIdle()
    }

    private fun menuItem(text: String): SemanticsNodeInteraction {
        compose.waitUntilAtLeastOneExists(isPopup(), 20_000)
        return compose.onNode(hasText(text) and hasAnyAncestor(isPopup()))
    }

    private fun openAdd() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
    }

    private fun typeWithTheKeyboardUp(text: String) {
        compose.onNode(field).performClick()
        compose.waitUntil(30_000) { keyboardIsUp() }
        compose.onNode(field).performTextInput(text)
    }

    private fun waitGone() {
        compose.waitUntil(30_000) { compose.onAllNodes(pane).fetchSemanticsNodes().isEmpty() }
    }

    // Above the sheet and the keyboard: the scrim's top edge.
    private fun tapOutside() {
        compose.onNodeWithContentDescription("Close sheet").performTouchInput { click(Offset(centerX, top + 200f)) }
    }

    private fun categoryNames() = engine.state.value.project(PID)!!.project.categories.map { it.name }

    @Test
    fun aTapOutsideTheAddSheetSavesWhatIsTyped() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        tapOutside()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitGone()
        compose.waitUntil(30_000) { !keyboardIsUp() }
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
    }

    @Test
    fun backLeavesWithTheKeyboardAndSaves() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        back()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitGone()
        assertTrue(!keyboardIsUp())
    }

    @Test
    fun aSwipeDownSaves() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        compose.onNode(pane).performTouchInput { swipeDown(top + 10f, bottom + 600f, 200) }
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitGone()
    }

    @Test
    fun anEmptySheetLeavesAndAddsNothing() {
        val before = engine.state.value.project(PID)!!.project.categories.sumOf { it.tasks.size }
        openAdd()
        tapOutside()
        waitGone()
        assertEquals(before, engine.state.value.project(PID)!!.project.categories.sumOf { it.tasks.size })
    }

    @Test
    fun aSaveTheFilterRefusesKeepsTheSheetOnceThenDrops() {
        runBlocking { engine.setFilter(PID, Filter(priorities = setOf("high"))) }
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        tapOutside()
        compose.waitUntilAtLeastOneExists(hasText("Clear the filter to add a task here"), 20_000)
        Thread.sleep(1_000)
        compose.onNode(pane).assertExists()
        tapOutside()
        waitGone()
        assertEquals(null, task("Water the plants"))
    }

    @Test
    fun theChipsLeaveTheKeyboardUp() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        compose.onNodeWithContentDescription("Priority").performClick()
        menuItem("High").assertExists()
        Thread.sleep(1_000)
        assertTrue("the keyboard stays up under the menu", keyboardIsUp())
        compose.onNode(field).assertIsFocused()
        menuItem("High").performClick()
        Thread.sleep(1_000)
        assertTrue(keyboardIsUp())
        compose.onNode(field).assertIsFocused()
        compose.onNodeWithContentDescription("Colour").performClick()
        menuItem("Green").performClick()
        Thread.sleep(1_000)
        assertTrue(keyboardIsUp())
    }

    @Test
    fun theAddSheetKeepsItsTextAcrossARotation() {
        openAdd()
        compose.onNode(field).performTextInput("Water the plants")
        recreate()
        compose.waitUntilAtLeastOneExists(hasText("Water the plants") and hasSetTextAction(), 20_000)
    }

    @Test
    fun aRenameLeftByATapOutsideIsSaved() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithText("Ordo").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextReplacement("Ordo app")
        tapOutside()
        compose.waitUntil(20_000) { engine.state.value.project(PID)?.name == "Ordo app" }
        waitGone()
    }

    @Test
    fun aNewCategoryLeftByBackIsAddedAndAnEmptyOneIsNot() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithContentDescription("More options").performClick()
        menuItem("New category").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        back()
        waitGone()
        val before = categoryNames()
        compose.onNodeWithContentDescription("More options").performClick()
        menuItem("New category").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextInput("Ideas")
        back()
        compose.waitUntil(20_000) { "Ideas" in categoryNames() }
        assertEquals(before.size + 1, categoryNames().size)
    }

    private fun hold() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithContentDescription("Add a task").performTouchInput { longClick() }
        compose.waitUntilAtLeastOneExists(hasText("Project"), 20_000)
    }

    @Test
    fun aTapOnPlusStillOpensATask() {
        openAdd()
        compose.onNodeWithText("New task").assertExists()
        compose.onNodeWithText("Project").assertDoesNotExist()
    }

    @Test
    fun theHeldPlusAddsATask() {
        hold()
        compose.onNodeWithText("Task").performClick()
        compose.waitUntilAtLeastOneExists(hasText("New task"), 20_000)
    }

    @Test
    fun theHeldPlusAddsANoteToTheFocusedCategory() {
        hold()
        compose.onNodeWithText("Note").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextInput("Ask Sam first")
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.first { it.name == "Feature" }.tasks.first().let { it.note && it.title == "Ask Sam first" } }
    }

    @Test
    fun theHeldPlusAddsASeparatorToTheFocusedCategory() {
        hold()
        compose.onNodeWithText("Separator").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.first { it.name == "Feature" }.tasks.first().separator }
    }

    @Test
    fun theHeldPlusAddsACategory() {
        hold()
        compose.onNodeWithText("Category").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextInput("Ideas")
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { "Ideas" in categoryNames() }
    }

    @Test
    fun theHeldPlusAddsAProjectAndShowsIt() {
        hold()
        compose.onNodeWithText("Project").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextInput("Garden")
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { engine.state.value.projects.any { it.name == "Garden" && it.id == engine.state.value.prefs.current } }
    }

    @Test
    fun theHeldPlusClosesOnBackAndAddsNothing() {
        hold()
        back()
        compose.waitUntil(20_000) { compose.onAllNodes(hasText("Project")).fetchSemanticsNodes().isEmpty() }
        compose.onNode(pane).assertDoesNotExist()
    }

    @Test
    fun aProjectWithoutCategoriesKeepsThePlusForANewCategory() {
        hold()
        compose.onNodeWithText("Project").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextInput("Garden")
        compose.onNodeWithText("Add").performClick()
        waitGone()
        compose.waitUntilAtLeastOneExists(hasContentDescription("New category"), 20_000)
        compose.onNodeWithContentDescription("New category").performClick()
        compose.waitUntilAtLeastOneExists(field, 20_000)
        compose.onNode(field).performTextInput("Beds")
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { engine.state.value.projects.first { it.name == "Garden" }.project.categories.any { it.name == "Beds" } }
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 20_000)
    }

    private fun openTask(title: String) {
        scrollTo(title)
        compose.onNodeWithText(title).performClick()
        compose.waitUntil(20_000) {
            runCatching { compose.onAllNodes(hasContentDescription("More")).fetchSemanticsNodes().isNotEmpty() }.getOrDefault(false)
        }
    }

    @Test
    fun aSubtaskTypedAndLeftIsAdded() {
        openTask("Tap a row to focus it")
        compose.onAllNodes(hasSetTextAction()).onLast().performTextInput("Check the edge")
        closePage(hasContentDescription("Add a task"))
        compose.waitUntil(20_000) { task("Check the edge")?.parentId == task("Tap a row to focus it")?.id }
    }

    @Test
    fun aDescriptionRewrittenElsewhereWhileTypingKeepsBothTexts() {
        openTask("Drag to reorder")
        val id = task("Drag to reorder")!!.id
        compose.onAllNodes(hasSetTextAction())[1].performClick()
        compose.onAllNodes(hasSetTextAction())[1].performTextInput("Mine first")
        compose.waitUntil(20_000) { task("Drag to reorder")?.description == "Mine first" }
        runBlocking { engine.edit(PID, Ops.description(id, "Theirs", "Mine first"), quiet = true) }
        compose.onAllNodes(hasSetTextAction())[1].performTextInput(", then more")
        compose.waitUntil(20_000) { task("Drag to reorder")?.description == "Theirs\n\nMine first, then more" }
    }
}
