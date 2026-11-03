package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.getBoundsInRoot
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasParent
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.text.TextLayoutResult
import androidx.compose.ui.unit.DpRect
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.ilyasturki.ordo.compose.model.Ops
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

private const val ERGONOMY = "cccccccc-0000-4000-8000-000000000014"
private const val LONG = "Let the list wrap a title this long over two lines at most, then cut it short with an ellipsis so the row stays small"
private const val NOTE = "A description long enough to need more than a single line of preview on a phone held upright, and then some more words."

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class ListTest : AppFixture() {
    private fun statusOf(title: String, status: String): SemanticsMatcher =
        hasContentDescription("$status, change status") and hasParent(hasText(title, substring = true))

    private fun lines(text: String): TextLayoutResult {
        val out = mutableListOf<TextLayoutResult>()
        compose.onNodeWithText(text, useUnmergedTree = true).fetchSemanticsNode().config[SemanticsActions.GetTextLayoutResult].action!!(out)
        return out.single()
    }

    private fun insert(title: String, description: String = "") = runBlocking {
        engine.edit(PID, Ops.insert(ERGONOMY, "", title), quiet = true)
        if (description != "") engine.edit(PID, Ops.description(task(title)!!.id, description, ""), quiet = true)
    }

    private fun bounds(description: String): DpRect = compose.onNodeWithContentDescription(description).getBoundsInRoot()

    private fun snackTop() = compose.onNodeWithTag("snackbar").getBoundsInRoot().top

    private fun gone(text: String) = compose.onAllNodes(hasText(text)).fetchSemanticsNodes().isEmpty()

    @Test
    fun aLongTitleClampsToTwoLinesAndItsPreviewToOne() {
        insert(LONG, NOTE)
        scrollTo(LONG)
        val title = lines(LONG)
        assertEquals(2, title.lineCount)
        assertTrue("the title ends in an ellipsis", title.hasVisualOverflow)
        assertEquals(1, lines(NOTE).lineCount)
    }

    @Test
    fun aOneLineTitleKeepsTwoLinesOfPreview() {
        insert("Short", NOTE)
        scrollTo("Short")
        assertEquals(1, lines("Short").lineCount)
        assertEquals(2, lines(NOTE).lineCount)
    }

    @Test
    fun aStatusTapIsQuietUntilItLandsOnDone() {
        val title = "Tap a row to focus it"
        scrollTo(title)
        compose.onNode(statusOf(title, "Todo")).performClick()
        compose.waitUntil(20_000) { task(title)?.status == "in_progress" }
        Thread.sleep(1_500)
        assertTrue("no snack for a step shown in place", gone("Undo") && compose.onAllNodes(hasText("Marked", substring = true)).fetchSemanticsNodes().isEmpty())
        compose.onNode(statusOf(title, "In Progress")).performClick()
        compose.waitUntil(20_000) { task(title)?.status == "completed" }
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.onNode(hasText("Marked completed")).assertExists()
        compose.waitUntil(8_000) { gone("Undo") }
    }

    @Test
    fun aSubtaskStatusTapIsQuiet() {
        scrollTo("Web UI for the phone")
        compose.onNodeWithText("Web UI for the phone").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("More"), 20_000)
        compose.onNode(statusOf("Drag to reorder", "Todo")).performClick()
        compose.waitUntil(20_000) { task("Drag to reorder")?.status == "in_progress" }
        Thread.sleep(1_500)
        assertTrue(gone("Undo"))
    }

    @Test
    fun thePlusRisesAboveASnackAndComesBack() {
        compose.waitUntilAtLeastOneExists(hasContentDescription("Add a task"), 20_000)
        val rest = bounds("Add a task")
        engine.say("A word from the test")
        compose.waitUntilAtLeastOneExists(hasText("A word from the test"), 20_000)
        compose.waitUntil(5_000) { bounds("Add a task").bottom <= snackTop() }
        assertTrue("the snack sits lower than the + did", compose.onNodeWithTag("snackbar").getBoundsInRoot().bottom > rest.bottom)
        compose.waitUntil(10_000) { gone("A word from the test") }
        compose.waitUntil(5_000) { bounds("Add a task") == rest }
    }

    @Test
    fun markDoneRisesAboveItsUndo() {
        scrollTo("Tap a row to focus it")
        compose.onNodeWithText("Tap a row to focus it").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("Mark done"), 20_000)
        val rest = bounds("Mark done")
        compose.onNodeWithContentDescription("Mark done").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Undo"), 20_000)
        compose.waitUntil(5_000) { bounds("Mark todo").bottom <= snackTop() }
        compose.waitUntil(10_000) { gone("Undo") }
        compose.waitUntil(5_000) { bounds("Mark todo").bottom == rest.bottom }
    }
}
