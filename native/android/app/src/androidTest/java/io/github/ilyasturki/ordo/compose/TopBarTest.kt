package io.github.ilyasturki.ordo.compose

import android.view.KeyEvent
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.isPopup
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performImeAction
import androidx.compose.ui.test.performScrollToIndex
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.unit.dp
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Project
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class TopBarTest : AppFixture() {
    private val project: Project get() = engine.state.value.project(PID)!!.project

    private fun category(name: String): Category = project.categories.first { it.name == name }

    private fun ready() = compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)

    // Enough rows that the category takes the whole screen once it is scrolled to the top.
    private fun fill(name: String) = runBlocking {
        repeat(12) { engine.edit(PID, Ops.insert(category(name).id, "", "Filler $it"), quiet = true) }
    }

    private fun scrollToHeader(name: String) {
        ready()
        compose.waitUntil(20_000) { Board.lines(project, Filter.NONE).any { it is Header && it.category.name == name } }
        val i = Board.lines(project, engine.state.value.prefs.filters[PID] ?: Filter.NONE).indexOfFirst { it is Header && it.category.name == name }
        compose.onNodeWithTag("list").performScrollToIndex(i)
    }

    private fun assertLabel(name: String) {
        compose.waitUntil(20_000) { compose.onAllNodes(hasTestTag("focus") and hasText(name)).fetchSemanticsNodes().isNotEmpty() }
    }

    private fun add(title: String) {
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNode(hasSetTextAction()).performTextInput(title)
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.waitUntil(20_000) { project.categories.any { c -> c.tasks.any { it.title == title } } }
    }

    private fun categoryOf(title: String): String = project.categories.first { c -> c.tasks.any { it.title == title } }.name

    private fun closeSheet() {
        compose.waitUntil(30_000) {
            if (compose.onAllNodes(isDialog()).fetchSemanticsNodes().isEmpty()) return@waitUntil true
            InstrumentationRegistry.getInstrumentation().sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
            compose.waitForIdle()
            compose.onAllNodes(isDialog()).fetchSemanticsNodes().isEmpty()
        }
    }

    @Test
    fun theProjectsHaveTheirOwnLineUnderTheActions() {
        ready()
        assertLabel("Feature")
        val settings = compose.onNodeWithContentDescription("Settings").getUnclippedBoundsInRoot()
        val tab = compose.onNodeWithText("Ordo", useUnmergedTree = true).getUnclippedBoundsInRoot()
        val newProject = compose.onNodeWithContentDescription("New project").getUnclippedBoundsInRoot()
        val label = compose.onNodeWithTag("focus").getUnclippedBoundsInRoot()
        assertTrue("the tab is below the actions", tab.top >= settings.bottom)
        assertTrue("the + tab is on the tabs' line", newProject.top >= settings.bottom)
        assertTrue("the + tab comes after the tab", newProject.left >= tab.right)
        assertTrue("the name is on the actions' line", label.top < settings.bottom && label.bottom > settings.top)
        compose.onNodeWithContentDescription("Account and settings").assertDoesNotExist()
    }

    @Test
    fun aLongCategoryNameStaysOnOneLineBeforeTheActions() {
        val long = "A category name far too long to fit next to the icons of the top bar"
        runBlocking { engine.edit(PID, Ops.categoryRename(category("Feature").id, long), quiet = true) }
        ready()
        assertLabel(long)
        val label = compose.onNodeWithTag("focus").getUnclippedBoundsInRoot()
        val search = compose.onNodeWithContentDescription("Search").getUnclippedBoundsInRoot()
        assertTrue("one line, ${label.bottom - label.top} tall", label.bottom - label.top < 40.dp)
        assertTrue("it ends before the icons", label.right <= search.left)
    }

    @Test
    fun theSearchReplacesBothLines() {
        ready()
        compose.onNodeWithContentDescription("Search").performClick()
        compose.waitUntilAtLeastOneExists(hasContentDescription("Close search"), 20_000)
        compose.onNodeWithTag("focus").assertDoesNotExist()
        compose.onNodeWithContentDescription("New project").assertDoesNotExist()
    }

    @Test
    fun anUnscrolledListNamesTheCategoryTakingMostOfTheScreenNotTheOneAbove() {
        runBlocking {
            val feature = category("Feature")
            feature.tasks.filter { it.parentId == "" }.forEach { engine.edit(PID, Ops.delete(it.id), quiet = true) }
        }
        fill("Fix")
        ready()
        assertLabel("Fix")
    }

    @Test
    fun anUnscrolledListWhoseFirstCategoryFillsTheScreenNamesIt() {
        ready()
        assertLabel("Feature")
    }

    @Test
    fun theBarNamesTheCategoryTakingMostOfTheScreen() {
        fill("Documentation")
        scrollToHeader("Documentation")
        assertLabel("Documentation")
        scrollToHeader("Feature")
        assertLabel("Feature")
    }

    @Test
    fun aSubCategoryShowsItsOwnName() {
        fill("API")
        scrollToHeader("API")
        assertLabel("API")
    }

    @Test
    fun aProjectWithNoCategoryNamesNothing() {
        runBlocking { engine.setCurrent(engine.createProject("Bare")!!) }
        compose.waitUntilAtLeastOneExists(hasText("No categories yet"), 20_000)
        assertEquals("", compose.onNodeWithTag("focus").fetchSemanticsNode().config[SemanticsProperties.Text].joinToString(""))
    }

    @Test
    fun thePlusAddsToTheCategoryOnScreenNotTheOneAbove() {
        fill("Research")
        scrollToHeader("Research")
        assertLabel("Research")
        add("Water the plants")
        assertEquals("Research", categoryOf("Water the plants"))
    }

    @Test
    fun thePlusFollowsTheFocusNotTheCategoryUsedLast() {
        ready()
        assertLabel("Feature")
        compose.onNodeWithContentDescription("Add a task").performClick()
        compose.onNode(hasText("Feature") and hasAnyAncestor(isDialog())).performClick()
        compose.onNode(hasText("Research") and hasAnyAncestor(isPopup())).performClick()
        compose.onNode(hasSetTextAction()).performTextInput("Water the plants")
        compose.onNode(hasSetTextAction()).performImeAction()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        assertEquals("Research", categoryOf("Water the plants"))
        closeSheet()
        assertLabel("Feature")
        add("Feed the cat")
        assertEquals("Feature", categoryOf("Feed the cat"))
    }

    @Test
    fun aFocusTheFilterHidesFallsBackToTheFirstCategoryItShows() {
        fill("Documentation")
        scrollToHeader("Documentation")
        assertLabel("Documentation")
        runBlocking { engine.setFilter(PID, Filter(categories = setOf(category("Research").id))) }
        assertLabel("Research")
        add("Water the plants")
        assertEquals("Research", categoryOf("Water the plants"))
    }
}
