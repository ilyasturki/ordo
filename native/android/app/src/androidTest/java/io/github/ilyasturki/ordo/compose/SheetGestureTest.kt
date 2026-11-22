package io.github.ilyasturki.ordo.compose

import android.os.ParcelFileDescriptor
import android.view.KeyEvent
import androidx.activity.BackEventCompat
import androidx.activity.ComponentActivity
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assert
import androidx.compose.ui.test.assertIsFocused
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasAnyDescendant
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.swipeDown
import androidx.compose.ui.unit.dp
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import org.junit.runner.RunWith

// Each way out of the add sheet: what it saves, whether the sheet stays, and where the keyboard ends once the motion settles.
@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class SheetGestureTest : AppFixture() {
    private val ins get() = InstrumentationRegistry.getInstrumentation()
    private val pane = SemanticsMatcher.keyIsDefined(SemanticsProperties.PaneTitle) and hasAnyDescendant(hasSetTextAction())
    private val field = hasSetTextAction() and hasAnyAncestor(pane)

    private fun page(): ComponentActivity =
        ActivityLifecycleMonitorRegistry.getInstance().getActivitiesInStage(Stage.RESUMED).single() as ComponentActivity

    private fun keyboardIsUp(): Boolean {
        var up = false
        ins.runOnMainSync { up = ViewCompat.getRootWindowInsets(page().window.decorView)?.isVisible(WindowInsetsCompat.Type.ime()) == true }
        return up
    }

    // The keyboard where the motion leaves it, and still there a second later: a show that lands late counts.
    private fun keyboardEnds(up: Boolean) {
        compose.waitUntil(10_000) { keyboardIsUp() == up }
        Thread.sleep(1_000)
        assertEquals(if (up) "the keyboard stays up" else "the keyboard stays down", up, keyboardIsUp())
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
        Thread.sleep(500)
    }

    private fun backKey() {
        ins.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
        compose.waitForIdle()
    }

    private fun backGesture(vararg progress: Float) {
        ins.runOnMainSync { page().onBackPressedDispatcher.dispatchOnBackStarted(BackEventCompat(0f, 1200f, 0f, BackEventCompat.EDGE_LEFT)) }
        for (p in progress) {
            ins.runOnMainSync { page().onBackPressedDispatcher.dispatchOnBackProgressed(BackEventCompat(p * 400f, 1200f, p, BackEventCompat.EDGE_LEFT)) }
            Thread.sleep(50)
        }
    }

    private fun fieldBottom() = compose.onNode(field).getUnclippedBoundsInRoot().bottom

    private fun shell(command: String) {
        ParcelFileDescriptor.AutoCloseInputStream(ins.uiAutomation.executeShellCommand(command)).use { it.readBytes() }
    }

    // A swipe from the screen's left edge through the system's input, as a finger makes it: the system decides who takes it, the keyboard or the sheet.
    private fun edgeSwipe(vararg xs: Int, release: Boolean = true) {
        shell("input motionevent DOWN 3 $edgeY")
        for (x in xs) {
            shell("input motionevent MOVE $x $edgeY")
            Thread.sleep(80)
        }
        if (release) shell("input motionevent UP ${xs.last()} $edgeY")
    }

    private val edgeY get() = ins.targetContext.resources.displayMetrics.heightPixels / 2

    @Test
    fun aBackKeyBeforeTheKeyboardShowsSavesAndNothingRises() {
        openAdd()
        compose.onNode(field).performTextInput("Water the plants")
        backKey()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitSheetGone()
        keyboardEnds(up = false)
    }

    @Test
    fun aBackKeyWithTheKeyboardUpSavesAndTakesItDown() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        backKey()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitSheetGone()
        keyboardEnds(up = false)
    }

    @Test
    fun aBackGestureMovesTheSheetAndSavesOnlyWhenItCompletes() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        val before = fieldBottom()
        backGesture(0.3f, 0.6f, 0.9f)
        compose.waitUntil(5_000) { fieldBottom() > before + 8.dp }
        assertNull(task("Water the plants"))
        ins.runOnMainSync { page().onBackPressedDispatcher.onBackPressed() }
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitSheetGone()
        keyboardEnds(up = false)
    }

    @Test
    fun aCancelledBackGestureLeavesTheSheetTheKeyboardAndTheText() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        val before = fieldBottom()
        backGesture(0.3f, 0.6f)
        ins.runOnMainSync { page().onBackPressedDispatcher.dispatchOnBackCancelled() }
        keyboardEnds(up = true)
        assertNull(task("Water the plants"))
        compose.onNode(field).assertIsFocused().assert(hasText("Water the plants"))
        assertEquals(before, fieldBottom())
        backKey()
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitSheetGone()
    }

    @Test
    fun aBackSwipeFromTheEdgeSavesOnlyOnRelease() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        edgeSwipe(60, 120, 180, 240, 300, 360, 420, release = false)
        Thread.sleep(500)
        assertNull(task("Water the plants"))
        compose.onNode(pane).assertExists()
        shell("input motionevent UP 420 $edgeY")
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitSheetGone()
        keyboardEnds(up = false)
    }

    @Test
    fun aBackSwipeTakenBackLeavesTheSheetTheKeyboardAndTheText() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        val before = fieldBottom()
        edgeSwipe(60, 120, 180, 240, 300, 360, 420, 300, 180, 60, 3)
        keyboardEnds(up = true)
        assertNull(task("Water the plants"))
        compose.onNode(field).assertIsFocused().assert(hasText("Water the plants"))
        assertEquals(before, fieldBottom())
    }

    @Test
    fun aSwipeDownCarriesTheKeyboardAndSaves() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        compose.onNode(pane).performTouchInput { swipeDown(top + 10f, bottom + 600f, 300) }
        compose.waitUntil(20_000) { task("Water the plants") != null }
        waitSheetGone()
        keyboardEnds(up = false)
    }

    @Test
    fun aShortSwipeDownPutsTheSheetAndTheKeyboardBack() {
        openAdd()
        typeWithTheKeyboardUp("Water the plants")
        val before = fieldBottom()
        compose.onNode(pane).performTouchInput { swipeDown(top + 10f, top + 80f, 800) }
        keyboardEnds(up = true)
        assertNull(task("Water the plants"))
        compose.onNode(field).assertIsFocused().assert(hasText("Water the plants"))
        assertEquals(before, fieldBottom())
    }
}
