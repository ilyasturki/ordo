package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.TouchInjectionScope
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.getBoundsInRoot
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.longClick
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performScrollToIndex
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTextReplacement
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeUp
import androidx.compose.ui.unit.height
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.ilyasturki.ordo.compose.core.CoreApi
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

private const val DOCUMENTATION = "cccccccc-0000-4000-8000-000000000019"

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class CategoriesTest : AppFixture() {
    private lateinit var door: Door

    override fun newCore(): CoreApi = Door(GoCore()).also { door = it }

    private fun order() = engine.state.value.project(PID)!!.project.categories.map { it.name }

    // The top bar can name the same category, without the row's click.
    private fun header(name: String) = compose.onNode(hasText(name) and hasClickAction())

    private fun research() = header("Research")

    private fun openPage(name: String = "Research") {
        scrollTo(name)
        header(name).performTouchInput { longClick() }
        compose.waitUntil(20_000) { pageHasFocus() }
        compose.waitUntilAtLeastOneExists(hasTestTag("categories"), 20_000)
    }

    // Unmerged: the row's clickable merges its icon into the row, whose centre is no handle.
    private fun handle(name: String) = compose.onNodeWithContentDescription("Move $name", useUnmergedTree = true)

    private fun SemanticsNodeInteraction.centerY(): Float {
        val b = getBoundsInRoot()
        return with(compose.density) { (b.top + b.height / 2).toPx() }
    }

    private fun TouchInjectionScope.glide(dy: Float) {
        repeat(12) { moveBy(Offset(0f, dy / 12), 80) }
        advanceEventTime(300)
        up()
    }

    private fun dragHandle(name: String, over: String) {
        val dy = handle(over).centerY() - handle(name).centerY()
        handle(name).performTouchInput {
            down(center)
            glide(dy)
        }
        compose.waitForIdle()
    }

    private fun heldHeaderOpensThePageAndADragStays(header: String = "Research") {
        openPage(header)
        compose.onNodeWithText("Ordo categories").assertIsDisplayed()
        dragHandle("Research", over = "Feature")
        val moved = listOf("Research", "Feature", "Fix", "Ergonomy", "Documentation", "API")
        compose.waitUntil(20_000) { order() == moved }
        assertEquals("nothing else moved, and the op waits for the server", 1, engine.state.value.pending)
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        closePage(until = hasContentDescription("Add a task"))
        compose.onNodeWithTag("list").performScrollToIndex(0)
        research().assertIsDisplayed()
    }

    @Test
    fun aHeldHeaderOpensThePageAndADragStays() = heldHeaderOpensThePageAndADragStays()

    @Test
    fun aHeldFoldedHeaderOpensThePageAndStaysFolded() {
        val ergonomy = engine.state.value.project(PID)!!.project.categories.first { it.name == "Ergonomy" }
        runBlocking { engine.setFolded(PID, ergonomy.id, true) }
        scrollTo("Ergonomy")
        compose.waitUntilAtLeastOneExists(hasContentDescription("Unfold Ergonomy"), 20_000)
        heldHeaderOpensThePageAndADragStays("Ergonomy")
        assertEquals(setOf(ergonomy.id), engine.state.value.views[PID]!!.folds)
        scrollTo("Ergonomy")
        compose.onNodeWithContentDescription("Unfold Ergonomy").assertExists()
    }

    @Test
    fun aTapOnAFoldedHeaderStillOpensItsMenu() {
        val ergonomy = engine.state.value.project(PID)!!.project.categories.first { it.name == "Ergonomy" }
        runBlocking { engine.setFolded(PID, ergonomy.id, true) }
        scrollTo("Ergonomy")
        compose.waitUntilAtLeastOneExists(hasContentDescription("Unfold Ergonomy"), 20_000)
        header("Ergonomy").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Delete category"), 20_000)
        assertEquals(setOf(ergonomy.id), engine.state.value.views[PID]!!.folds)
    }

    @Test
    fun aHeldRowDragsAndAReleaseInPlaceOpensNothing() {
        openPage()
        compose.onNodeWithText("Fix").performTouchInput { longClick() }
        compose.waitForIdle()
        Thread.sleep(1_000)
        compose.onNodeWithText("Rename").assertDoesNotExist()
        assertEquals(0, engine.state.value.pending)
        val dy = handle("Feature").centerY() - handle("Fix").centerY()
        compose.onNodeWithText("Fix").performTouchInput {
            down(center)
            advanceEventTime(1_000)
            glide(dy)
        }
        compose.waitUntil(20_000) { order().first() == "Fix" }
        compose.onNodeWithText("Rename").assertDoesNotExist()
        assertTrue("a release goes nowhere", pageHasFocus())
    }

    @Test
    fun undoPutsADraggedCategoryBackFromThePage() {
        val before = order()
        openPage()
        dragHandle("Fix", over = "Feature")
        compose.waitUntil(20_000) { order() != before }
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(20_000) { order() == before }
    }

    @Test
    fun aCategoryDraggedBackWhereItWasStays() {
        val before = order()
        openPage()
        dragHandle("Fix", over = "Feature")
        compose.waitUntil(20_000) { order().take(2) == listOf("Fix", "Feature") }
        dragHandle("Fix", over = "Feature")
        compose.waitUntil(20_000) { order() == before }
        Thread.sleep(1_000)
        assertEquals(before, order())
    }

    @Test
    fun aCategoryLiftedWhileTheLastDropStillLandsKeepsItsPlace() {
        openPage()
        door.shut()
        dragHandle("Fix", over = "Feature")
        compose.waitUntil(20_000) { door.knocks.get() == 1 }
        val dy = handle("Ergonomy").centerY() - handle("Research").centerY()
        handle("Research").performTouchInput {
            down(center)
            repeat(12) { moveBy(Offset(0f, dy / 12), 80) }
        }
        compose.waitForIdle()
        door.open()
        compose.waitUntil(20_000) { order().take(2) == listOf("Fix", "Feature") }
        compose.waitForIdle()
        compose.onNodeWithTag("categories").performTouchInput {
            advanceEventTime(300)
            up()
        }
        compose.waitUntil(20_000) { order() == listOf("Fix", "Feature", "Research", "Ergonomy", "Documentation", "API") }
    }

    @Test
    fun aBlockTakesItsSubCategoriesAlong() {
        openPage()
        dragHandle("Documentation", over = "Feature")
        compose.waitUntil(20_000) { order() == listOf("Documentation", "API", "Feature", "Fix", "Ergonomy", "Research") }
    }

    @Test
    fun aSubCategoryMovesAmongItsSiblingsOnly() {
        runBlocking { engine.edit(PID, Ops.categoryAdd("Guides", DOCUMENTATION), quiet = true) }
        openPage()
        dragHandle("Guides", over = "API")
        compose.waitUntil(20_000) { order().filter { it == "API" || it == "Guides" } == listOf("Guides", "API") }
        assertEquals(listOf("Feature", "Fix", "Ergonomy", "Documentation", "Guides", "API", "Research"), order())
        dragHandle("Guides", over = "Feature")
        compose.waitForIdle()
        Thread.sleep(1_000)
        val parents = engine.state.value.project(PID)!!.project.categories.filter { it.parentId == DOCUMENTATION }.map { it.name }.toSet()
        assertEquals(setOf("API", "Guides"), parents)
        assertEquals(listOf("Feature", "Fix", "Ergonomy", "Documentation"), order().take(4))
    }

    @Test
    fun theDotsOpenWhatTheHeadersMenuOffers() {
        openPage()
        compose.onNodeWithContentDescription("Edit Research").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Rename"), 20_000)
        compose.onNodeWithText("New sub-category").assertIsDisplayed()
        compose.onNodeWithText("Add a separator").assertIsDisplayed()
        compose.onNodeWithText("Add a note").assertIsDisplayed()
        compose.onNodeWithText("Delete category").performClick()
        compose.onNodeWithText("Delete category “Research”?").assertIsDisplayed()
        compose.onNodeWithText("Cancel").performClick()
        research().assertIsDisplayed()
    }

    private fun category(name: String) = engine.state.value.project(PID)!!.project.categories.first { it.name == name }

    private fun assertFocus(name: String) = compose.waitUntil(20_000) {
        compose.onAllNodes(hasTestTag("focus") and hasText(name)).fetchSemanticsNodes().isNotEmpty()
    }

    // The page closes on the tap, and the list underneath takes the keys again.
    private fun goTo(name: String) {
        compose.onNode(hasText(name) and hasClickAction() and hasAnyAncestor(hasTestTag("categories"))).performClick()
        compose.waitUntil(20_000) { !pageHasFocus() }
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 20_000)
    }

    private fun waitShown(matcher: SemanticsMatcher) = compose.waitUntil(20_000) {
        runCatching { compose.onNode(matcher).assertIsDisplayed() }.isSuccess
    }

    private fun addHere(title: String): String {
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNode(hasSetTextAction()).performTextInput(title)
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.waitUntil(20_000) { task(title) != null }
        return engine.state.value.project(PID)!!.project.categories.first { c -> c.tasks.any { it.title == title } }.name
    }

    @Test
    fun aTapGoesToTheCategoryAndTheFocusHoldsUntilAScroll() {
        openPage("Feature")
        goTo("Ergonomy")
        waitShown(hasText("Ergonomy") and hasClickAction())
        // Ergonomy is too short to take most of the screen: only the jump keeps it the focus.
        Thread.sleep(1_000)
        assertFocus("Ergonomy")
        compose.onNodeWithTag("list").performTouchInput { swipeUp(startY = centerY, endY = centerY - 200f) }
        compose.waitUntil(20_000) { compose.onAllNodes(hasTestTag("focus") and hasText("Ergonomy")).fetchSemanticsNodes().isEmpty() }
    }

    @Test
    fun aTapGoesToACategoryFarDownAndThePlusAddsThere() {
        openPage("Feature")
        goTo("Research")
        waitShown(hasText("Research") and hasClickAction())
        assertFocus("Research")
        assertEquals("Research", addHere("Water the plants"))
    }

    @Test
    fun aTapUnfoldsTheSubCategoryItGoesToAndItsParent() {
        val docs = category("Documentation")
        runBlocking { engine.setFolded(PID, docs.id, true) }
        openPage("Feature")
        goTo("API")
        compose.waitUntil(20_000) { engine.state.value.views[PID]?.folds.orEmpty().isEmpty() }
        waitShown(hasText("API", substring = true) and hasClickAction() and hasAnyAncestor(hasTestTag("list")))
        assertFocus("API")
    }

    @Test
    fun aTapOnACategoryTheFilterHidesSaysSoAndStays() {
        runBlocking { engine.setFilter(PID, Filter(categories = setOf(category("Feature").id))) }
        openPage("Feature")
        compose.onNode(hasText("Research") and hasClickAction() and hasAnyAncestor(hasTestTag("categories"))).performClick()
        compose.waitUntilAtLeastOneExists(hasText("“Research” is hidden by the filter"), 20_000)
        assertTrue(pageHasFocus())
    }

    @Test
    fun thePlusAddsACategoryLast() {
        openPage()
        compose.onNodeWithContentDescription("New category").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Add"), 20_000)
        compose.onNode(hasSetTextAction()).performTextInput("Backlog")
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { order().last() == "Backlog" }
        // The closing sheet's field still reads "Backlog" until it is gone.
        compose.waitUntilExactlyOneExists(hasText("Backlog"), 20_000)
        compose.onNodeWithText("Backlog").assertIsDisplayed()
    }

    @Test
    fun aPinPutsTheCategoryFirstAndMarksIt() {
        scrollTo("Research")
        research().performClick()
        compose.waitUntilAtLeastOneExists(hasText("Pin"), 20_000)
        compose.onNodeWithText("Pin").performClick()
        compose.waitUntil(20_000) { order().first() == "Research" }
        assertEquals(true, engine.state.value.project(PID)!!.project.categories.first().pinned)
        compose.onNodeWithTag("list").performScrollToIndex(0)
        compose.waitUntilAtLeastOneExists(hasContentDescription("Pinned"), 20_000)
        compose.onNode(hasText("Research", substring = true) and hasClickAction()).performClick()
        compose.waitUntilAtLeastOneExists(hasText("Unpin"), 20_000)
        compose.onNodeWithText("Unpin").performClick()
        compose.waitUntil(20_000) { engine.state.value.project(PID)!!.project.categories.first().pinned == false }
        assertEquals("Research", order().first())
    }

    private fun renameResearchTo(name: String) {
        scrollTo("Research")
        research().performClick()
        compose.waitUntilAtLeastOneExists(hasText("Rename"), 20_000)
        compose.onNode(hasSetTextAction()).performTextReplacement(name)
        compose.onNodeWithText("Rename").performClick()
        compose.waitUntilAtLeastOneExists(hasText("“Feature” is taken"), 20_000)
    }

    @Test
    fun aTakenNameOffersAFreeOne() {
        renameResearchTo("Feature")
        compose.onNodeWithText("Name it “Feature (2)”").performClick()
        compose.waitUntil(20_000) { order().last() == "Feature (2)" }
        compose.waitUntilDoesNotExist(hasText("Rename"), 20_000)
        assertEquals("Feature", order().first())
    }

    @Test
    fun aTakenNameCanRenameTheOther() {
        renameResearchTo("feature")
        compose.onNodeWithText("Rename the other to “Feature (2)”").performClick()
        compose.waitUntil(20_000) { order().first() == "Feature (2)" && order().last() == "feature" }
        compose.waitUntilDoesNotExist(hasText("Rename"), 20_000)
    }

    @Test
    fun cancellingATakenNameKeepsTheSheet() {
        renameResearchTo("Feature")
        compose.onNodeWithText("Cancel").performClick()
        compose.waitUntilDoesNotExist(hasText("“Feature” is taken"), 20_000)
        compose.onNodeWithText("Rename").assertIsDisplayed()
        assertEquals("Research", order().last())
        assertEquals(0, engine.state.value.pending)
    }

    @Test
    fun tappingTheCurrentTabStillOpensTheProjectMenu() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithText("Ordo").performClick()
        compose.onNodeWithText("Delete project").assertIsDisplayed()
    }
}
