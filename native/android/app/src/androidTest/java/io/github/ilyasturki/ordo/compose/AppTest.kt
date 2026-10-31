package io.github.ilyasturki.ordo.compose

import android.content.Intent
import android.view.KeyEvent
import android.view.inspector.WindowInspector
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasAnyDescendant
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasParent
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
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
import androidx.compose.ui.unit.height
import androidx.compose.ui.unit.width
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
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

    // The top bar can name the same category, without the header's click.
    private fun header(name: String) = compose.onNode(hasText(name) and hasClickAction())

    // No activity is on screen for a moment as one replaces another, and the rule then throws.
    private fun waitAcrossPages(matcher: SemanticsMatcher) = compose.waitUntil(20_000) {
        runCatching { compose.onAllNodes(matcher).fetchSemanticsNodes().isNotEmpty() }.getOrDefault(false)
    }

    private fun openTask(title: String) {
        scrollTo(title)
        compose.onNodeWithText(title).performClick()
        waitAcrossPages(hasContentDescription("More"))
    }

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

    // The first back may only close the keyboard.
    private fun backUntil(gone: String) {
        repeat(3) {
            if (compose.onAllNodesWithContentDescription(gone).fetchSemanticsNodes().isNotEmpty()) back()
        }
        compose.onNodeWithContentDescription(gone).assertDoesNotExist()
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
        header("Research").performClick()
        compose.onNodeWithText("Add a separator").performClick()
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }.tasks.first().separator }
        assertEquals("", engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }.tasks.first().title)
    }

    @Test
    fun anUnlabelledSeparatorOpensLikeALabelledOne() {
        scrollTo("Research")
        header("Research").performClick()
        compose.onNodeWithText("Add a separator").performClick()
        compose.onNodeWithText("Add").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("Separator"), 20_000)
        compose.onNodeWithContentDescription("Separator").performClick()
        waitAcrossPages(hasText("Separator label"))
    }

    @Test
    fun theAddSheetNamesPriorityAndColourUntilTheyAreSet() {
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 20_000)
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNodeWithText("Priority").performClick()
        compose.onNodeWithText("Medium").performClick()
        compose.onNodeWithText("Priority").assertDoesNotExist()
        compose.onNodeWithText("Medium").assertExists()
        compose.onNodeWithText("Colour").performClick()
        compose.onNodeWithText("Green").performClick()
        compose.onNodeWithText("Colour").assertDoesNotExist()
        compose.onNodeWithText("Green").assertExists()
    }

    @Test
    fun theTaskPageNamesPriorityAndColourUntilTheyAreSet() {
        openTask("Drag to reorder")
        compose.onNodeWithText("Colour").performClick()
        compose.onNodeWithText("Cyan").performClick()
        compose.waitUntil(20_000) { task("Drag to reorder")?.tagColor == "cyan" }
        compose.onNodeWithText("Colour").assertDoesNotExist()
        compose.onNodeWithText("Priority").performClick()
        compose.onNodeWithText("High").performClick()
        compose.waitUntil(20_000) { task("Drag to reorder")?.priority == "high" }
        compose.onNodeWithText("Priority").assertDoesNotExist()
    }

    @Test
    fun theFilterRefusesASeparator() {
        runBlocking { engine.setFilter(PID, Filter(statuses = setOf("todo"))) }
        scrollTo("Research")
        header("Research").performClick()
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
        compose.onNodeWithText("Priority").performClick()
        compose.onNodeWithText("Medium").performClick()
        compose.onNodeWithText("Clear the filter to add a task here").assertDoesNotExist()
        compose.onNodeWithText("Save").performClick()
        compose.waitUntil(20_000) { task("Water the plants")?.priority == "medium" }
    }

    @Test
    fun theAddSheetPutsTheTaskFirstInItsCategory() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNode(hasSetTextAction()).performTextInput("Water the plants")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        assertEquals("Water the plants", engine.state.value.project(PID)!!.project.categories[0].tasks.first().title)
    }

    // Where the keyboard starts on the screen, in dp; null while it is down.
    private fun keyboardTop(): Float? {
        var top: Float? = null
        InstrumentationRegistry.getInstrumentation().runOnMainSync {
            for (window in WindowInspector.getGlobalWindowViews()) {
                val insets = ViewCompat.getRootWindowInsets(window) ?: continue
                if (insets.isVisible(WindowInsetsCompat.Type.ime())) {
                    top = (window.height - insets.getInsets(WindowInsetsCompat.Type.ime()).bottom) / compose.density.density
                }
            }
        }
        return top
    }

    // A sheet whose height follows the keyboard moves its own anchor on every keyboard frame, and slides up only once the keyboard has stopped.
    private fun assertTheSheetKeepsItsSizeOnTheKeyboard() {
        val pane = SemanticsMatcher.keyIsDefined(SemanticsProperties.PaneTitle) and hasAnyDescendant(hasSetTextAction())
        // Under instrumentation the sheet's window takes focus too late for the focus request to raise the keyboard.
        compose.onNode(hasSetTextAction()).performClick()
        compose.waitUntil(30_000) { keyboardTop() != null }
        compose.waitUntil(30_000) {
            compose.onNode(hasSetTextAction()).getUnclippedBoundsInRoot().bottom.value <= keyboardTop()!!
        }
        val up = compose.onNode(pane).getUnclippedBoundsInRoot()
        back()
        compose.waitUntil(30_000) { keyboardTop() == null }
        compose.waitForIdle()
        val down = compose.onNode(pane).getUnclippedBoundsInRoot()
        assertEquals(down.height.value, up.height.value, 1f)
    }

    @Test
    fun theNameSheetKeepsItsSizeOnTheKeyboard() {
        scrollTo("Tap a row to focus it")
        compose.onNodeWithText("Ordo").performClick()
        assertTheSheetKeepsItsSizeOnTheKeyboard()
    }

    @Test
    fun theAddSheetKeepsItsSizeOnTheKeyboard() {
        scrollTo("Tap a row to focus it")
        compose.onNodeWithContentDescription("Add a task").performClick()
        assertTheSheetKeepsItsSizeOnTheKeyboard()
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
    fun aPhotoWaitingOnThePhoneShowsAndCountsInTheExplanation() {
        val images = engine.images
        val temp = images.newTemp().apply { writeBytes(byteArrayOf(0x89.toByte(), 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3)) }
        val kept = images.keep(temp, "offline.png")
        runBlocking { engine.attach(PID, task("Recurring tasks")!!.id, listOf(kept)) }
        compose.waitUntilAtLeastOneExists(hasContentDescription("Not synced"), 20_000)
        compose.onNodeWithContentDescription("Not synced").performClick()
        compose.waitUntilAtLeastOneExists(hasText("1 edit and 1 photo wait safely on this phone."), 20_000)
        back()
        compose.waitUntil(20_000) { compose.onAllNodes(hasText("1 edit and 1 photo wait safely on this phone.")).fetchSemanticsNodes().isEmpty() }
        openTask("Recurring tasks")
        compose.waitUntilAtLeastOneExists(hasContentDescription("offline"), 20_000)
        compose.onNodeWithContentDescription("Remove offline").assertIsDisplayed()
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
        waitAcrossPages(hasContentDescription("Add a task"))
        compose.waitUntil(20_000) { task("Tap a row to edit it") != null }
        assertNull(task("Tap a row to focus it"))
    }

    @Test
    fun aTaskDeletedOnItsPageOffersUndoOnTheList() {
        openTask("Tap a row to focus it")
        compose.onNodeWithContentDescription("More").performClick()
        compose.onNodeWithText("Delete").performClick()
        waitAcrossPages(hasContentDescription("Add a task"))
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(20_000) { task("Tap a row to focus it") != null }
    }

    @Test
    fun theSearchNarrowsTheListInPlaceAndBackClosesIt() {
        compose.waitUntilAtLeastOneExists(hasContentDescription("Search"), 20_000)
        compose.onNodeWithContentDescription("Search").performClick()
        compose.waitUntil(30_000) { keyboardIsUp() }
        compose.onNode(hasSetTextAction()).performTextInput("pairing")
        compose.onNodeWithText("Pairing link login").assertIsDisplayed()
        compose.onNodeWithText("Recurring tasks").assertDoesNotExist()
        compose.onNodeWithContentDescription("Add a task").assertDoesNotExist()
        compose.onNodeWithText("Pairing link login").performClick()
        waitAcrossPages(hasContentDescription("More"))
        closePage(hasContentDescription("Close search"))
        compose.onNodeWithText("Pairing link login").assertIsDisplayed()
        compose.onNodeWithText("Recurring tasks").assertDoesNotExist()
        compose.onNode(hasSetTextAction()).performTextReplacement("no such task anywhere")
        compose.onNodeWithText("Nothing matches").assertIsDisplayed()
        backUntil("Close search")
        compose.waitUntilAtLeastOneExists(hasContentDescription("Search"), 20_000)
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
        header("Research").performClick()
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
        header("Documentation").performClick()
        compose.onNodeWithText("Delete category").performClick()
        compose.onNodeWithText("Delete category “Documentation”?").assertIsDisplayed()
        compose.onNodeWithText("Its sub-category and 5 tasks go too.").assertIsDisplayed()
        compose.onNodeWithText("Delete").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.none { it.name == "API" } }
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.map { it.name } == before }
    }

    @Test
    fun theLauncherOpensTheAppThroughTheOldAppsComponentName() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val launcher = Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER).setPackage(context.packageName)
        val found = context.packageManager.resolveActivity(launcher, 0)!!.activityInfo
        assertEquals("io.github.ilyasturki.ordo.MainActivity", found.name)
        assertEquals(MainActivity::class.java.name, found.targetActivity)
    }

    @Test
    fun theChevronFoldsACategoryAndATapStillOpensItsMenu() {
        scrollTo("Research")
        compose.onNodeWithContentDescription("Fold Research").performClick()
        compose.onNodeWithContentDescription("Unfold Research").assertExists()
        compose.waitUntil(20_000) { engine.state.value.views[PID]?.folds?.size == 1 }
        val folded = engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }
        assertEquals(setOf(folded.id), engine.state.value.views[PID]!!.folds)
        for (t in folded.tasks) compose.onNodeWithText(t.title).assertDoesNotExist()

        header("Research").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Delete category"), 20_000)
        back()

        compose.onNodeWithContentDescription("Unfold Research").performClick()
        compose.waitUntil(20_000) { engine.state.value.views[PID]?.folds?.isEmpty() == true }
        scrollTo(folded.tasks.first { !it.separator }.title)
    }

    @Test
    fun aFoldedParentHidesItsSubCategories() {
        scrollTo("Documentation")
        val cats = engine.state.value.project(PID)!!.project.categories
        val parent = cats.first { it.name == "Documentation" }
        val sub = cats.first { it.parentId == parent.id }
        compose.onNodeWithContentDescription("Fold Documentation").performClick()
        compose.waitUntil(20_000) { engine.state.value.views[PID]?.folds == setOf(parent.id) }
        compose.onNodeWithContentDescription("Fold ${sub.name}").assertDoesNotExist()
    }

    @Test
    fun theChevronFoldsARowWithoutOpeningIt() {
        val parent = engine.state.value.project(PID)!!.project.categories.flatMap { it.tasks }.first { t ->
            engine.state.value.project(PID)!!.project.categories.flatMap { it.tasks }.any { it.parentId == t.id }
        }
        val kid = engine.state.value.project(PID)!!.project.categories.flatMap { it.tasks }.first { it.parentId == parent.id }
        scrollTo(parent.title)
        compose.onNodeWithText(kid.title).assertExists()
        compose.onNodeWithContentDescription("Fold subtasks of ${parent.title}").performClick()
        compose.waitUntil(20_000) { engine.state.value.views[PID]?.folds == setOf(parent.id) }
        compose.onNodeWithText(kid.title).assertDoesNotExist()
        compose.onNodeWithContentDescription("More").assertDoesNotExist()
        compose.onNodeWithContentDescription("Unfold subtasks of ${parent.title}").performClick()
        compose.waitUntilAtLeastOneExists(hasText(kid.title), 20_000)
    }

    @Test
    fun aFoldArrivingAppliesWithTheListInPlace() {
        scrollTo("Research")
        val research = engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }
        runBlocking { engine.setFolded(PID, research.id, true) }
        compose.waitUntilAtLeastOneExists(hasContentDescription("Unfold Research"), 20_000)
        header("Research").assertIsDisplayed()
    }

    @Test
    fun searchLooksThroughAFold() {
        val research = engine.state.value.project(PID)!!.project.categories.first { it.name == "Research" }
        val title = research.tasks.first { !it.separator }.title
        runBlocking { engine.setFolded(PID, research.id, true) }
        scrollTo("Research")
        compose.onNodeWithContentDescription("Search").performClick()
        compose.onNode(hasSetTextAction()).performTextInput(title)
        compose.waitUntilAtLeastOneExists(hasText(title), 20_000)
        compose.onNodeWithContentDescription("Unfold Research").assertDoesNotExist()
        assertEquals(setOf(research.id), engine.state.value.views[PID]!!.folds)
    }
}
