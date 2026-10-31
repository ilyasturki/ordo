package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.TouchInjectionScope
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.getBoundsInRoot
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
import androidx.compose.ui.test.performScrollToIndex
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.unit.height
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.ilyasturki.ordo.compose.model.Ops
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith

private const val DOCUMENTATION = "cccccccc-0000-4000-8000-000000000019"

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class CategoriesTest : AppFixture() {
    private fun order() = engine.state.value.project(PID)!!.project.categories.map { it.name }

    // The top bar can name the same category, without the row's click.
    private fun research() = compose.onNode(hasText("Research") and hasClickAction())

    private fun openPage() {
        scrollTo("Research")
        research().performTouchInput { longClick() }
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

    @Test
    fun aHeldHeaderOpensThePageAndADragStays() {
        openPage()
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
    fun aTapOpensWhatTheHeadersMenuOffers() {
        openPage()
        research().performClick()
        compose.waitUntilAtLeastOneExists(hasText("Rename"), 20_000)
        compose.onNodeWithText("New sub-category").assertIsDisplayed()
        compose.onNodeWithText("Add a separator").assertIsDisplayed()
        compose.onNodeWithText("Delete category").performClick()
        compose.onNodeWithText("Delete category “Research”?").assertIsDisplayed()
        compose.onNodeWithText("Cancel").performClick()
        research().assertIsDisplayed()
    }

    @Test
    fun thePlusAddsACategoryLast() {
        openPage()
        compose.onNodeWithContentDescription("New category").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Add"), 20_000)
        compose.onNode(hasSetTextAction()).performTextInput("Backlog")
        compose.onNodeWithText("Add").performClick()
        compose.waitUntil(20_000) { order().last() == "Backlog" }
        compose.waitUntilAtLeastOneExists(hasText("Backlog"), 20_000)
        compose.onNodeWithText("Backlog").assertIsDisplayed()
    }

    @Test
    fun tappingTheCurrentTabStillOpensTheProjectMenu() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithText("Ordo").performClick()
        compose.onNodeWithText("Delete project").assertIsDisplayed()
    }
}
