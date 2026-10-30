package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.onFirst
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.unit.DpRect
import androidx.compose.ui.unit.lerp
import androidx.compose.ui.unit.width
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

private val NAMES = listOf("Ab", "Ordo app", "Planning the next big release", "Zed", "Zulu field notes", "Zzz the very last project")

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class TabsTest : AppFixture(NAMES) {
    private fun bar(): DpRect = compose.onNodeWithTag("indicator", useUnmergedTree = true).getUnclippedBoundsInRoot()

    private fun label(name: String): DpRect = compose.onNodeWithText(name, useUnmergedTree = true).getUnclippedBoundsInRoot()

    private fun assertUnder(name: String) {
        val bar = bar()
        val label = label(name)
        assertEquals("$name centre", (label.left + label.right).value / 2, (bar.left + bar.right).value / 2, 1f)
        assertEquals("$name width", maxOf(label.width.value, 24f), bar.width.value, 1f)
    }

    private fun settled() {
        compose.waitUntilAtLeastOneExists(hasTestTag("indicator"), 5_000)
        compose.waitForIdle()
    }

    @Test
    fun theUnderlineSitsUnderATabNarrowerThanItsMinimum() {
        settled()
        assertUnder("Ab")
    }

    @Test
    fun theUnderlineSitsUnderEachTabItIsMovedToAsTheRowScrolls() {
        settled()
        for (name in listOf("Ordo app", "Planning the next big release", "Zed", "Zulu field notes", "Zzz the very last project", "Zed", "Ab")) {
            compose.onNodeWithText(name, useUnmergedTree = true).performClick()
            compose.waitForIdle()
            assertUnder(name)
        }
    }

    @Test
    fun deletingTheOpenProjectOpensTheFirstOne() {
        settled()
        compose.onNodeWithText("Zed", useUnmergedTree = true).performClick()
        compose.waitForIdle()
        compose.onNodeWithText("Zed", useUnmergedTree = true).performClick()
        compose.onNodeWithText("Delete project").performClick()
        compose.onNodeWithText("Delete", useUnmergedTree = true).performClick()
        compose.waitUntil(5_000) { engine.state.value.projects.none { it.name == "Zed" } }
        compose.waitUntil(5_000) { engine.state.value.prefs.current == PID }
        compose.waitForIdle()
        assertUnder("Ab")
    }

    @Test
    fun theUnderlineFollowsTheFingerBetweenTwoTabs() {
        settled()
        compose.onNodeWithText("Ordo app", useUnmergedTree = true).performClick()
        compose.waitForIdle()
        val a = label("Ordo app")
        val b = label("Planning the next big release")
        compose.onAllNodesWithTag("list").onFirst().performTouchInput {
            down(center)
            moveBy(Offset(-width * 0.2f, 0f), 100)
            moveBy(Offset(-width * 0.2f, 0f), 100)
        }
        compose.waitForIdle()
        val bar = bar()
        val aStart = a.left
        val bStart = b.left
        val f = (bar.left - aStart) / (bStart - aStart)
        assertTrue("the finger is between the tabs, at $f", f > 0.1f && f < 0.9f)
        assertEquals(lerp(a.width, b.width, f).value, bar.width.value, 1f)
        compose.onAllNodesWithTag("list").onFirst().performTouchInput { up() }
    }
}
