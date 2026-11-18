package io.github.ilyasturki.ordo.compose

import android.graphics.Rect
import android.os.SystemClock
import android.view.InputDevice
import android.view.MotionEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.sync.Delays
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

private val OPEN = Filter(statuses = setOf(Status.TODO, Status.IN_PROGRESS))

// Drags on drag.json through the system's input in real time, as a finger makes them: a flick's events
// come faster than the list lays out its swaps, and a still finger sends none.
@RunWith(AndroidJUnit4::class)
class DragTest : AppFixture(composing = false, seed = "drag.json") {
    override val delays = Delays(600_000, 600_000, 600_000, 600_000, 600_000)

    private val ins get() = InstrumentationRegistry.getInstrumentation()

    private val project get() = engine.state.value.project(PID)!!.project

    private fun titles(category: String) = project.categories.first { it.name == category }.tasks.filter { it.parentId == "" }.map { it.title }

    private fun waitFor(what: String, check: () -> Boolean) {
        val until = SystemClock.uptimeMillis() + 20_000
        while (!check()) {
            assertTrue(what, SystemClock.uptimeMillis() < until)
            Thread.sleep(100)
        }
    }

    private fun waitFor(category: String, want: List<String>) {
        val until = SystemClock.uptimeMillis() + 20_000
        while (titles(category) != want && SystemClock.uptimeMillis() < until) Thread.sleep(100)
        assertEquals(want, titles(category))
    }

    // The rows, headers and lines holding the text, read afresh: the cached tree lags a swap behind. The top
    // bar's name of the category isn't one.
    private fun rows(text: String): List<AccessibilityNodeInfo> {
        val found = ArrayList<AccessibilityNodeInfo>()
        val next = ArrayDeque(listOfNotNull(ins.uiAutomation.rootInActiveWindow))
        while (next.isNotEmpty()) {
            val n = next.removeFirst()
            if (!n.refresh()) continue
            if (n.text?.toString() == text) (if (n.isClickable) n else n.parent?.takeIf { it.refresh() && it.isClickable })?.let(found::add)
            for (i in 0 until n.childCount) n.getChild(i)?.let(next::add)
        }
        return found
    }

    private fun onScreen(text: String) = rows(text).any { n -> !Rect().also(n::getBoundsInScreen).isEmpty }

    private fun bounds(text: String): Rect {
        waitFor("$text shows") { onScreen(text) }
        return rows(text).map { n -> Rect().also(n::getBoundsInScreen) }.first { !it.isEmpty }
    }

    private var downAt = 0L

    private fun touch(action: Int, x: Float, y: Float, wait: Boolean = true) {
        val now = SystemClock.uptimeMillis()
        if (action == MotionEvent.ACTION_DOWN) downAt = now
        val event = MotionEvent.obtain(downAt, now, action, x, y, 0).apply { source = InputDevice.SOURCE_TOUCHSCREEN }
        ins.uiAutomation.injectInputEvent(event, wait)
        event.recycle()
    }

    private fun slide(x: Float, from: Float, to: Float, steps: Int, pause: Long) {
        for (i in 1..steps) {
            touch(MotionEvent.ACTION_MOVE, x, from + (to - from) * i / steps)
            Thread.sleep(pause)
        }
    }

    // A long press on the row, a slow move of dy, then a hold of the trembling finger before the release.
    private fun drag(title: String, dy: Float, hold: Long = 600) {
        val at = bounds(title)
        val x = at.exactCenterX()
        val from = at.exactCenterY()
        touch(MotionEvent.ACTION_DOWN, x, from)
        Thread.sleep(800)
        slide(x, from, from + dy, steps = 20, pause = 30)
        val until = SystemClock.uptimeMillis() + hold
        var i = 0
        while (SystemClock.uptimeMillis() < until) {
            touch(MotionEvent.ACTION_MOVE, x, from + dy + if (i++ % 2 == 0) 1f else -1f)
            Thread.sleep(50)
        }
        touch(MotionEvent.ACTION_UP, x, from + dy)
    }

    // A long press, then moves at a touchscreen's pace that bring the row's centre onto the line's, and the
    // finger lifted at once: none waits for the app, so the last moves reach it with the release.
    private fun flick(title: String, onto: String) {
        val at = bounds(title)
        val x = at.exactCenterX()
        val from = at.exactCenterY()
        val to = bounds(onto).exactCenterY()
        touch(MotionEvent.ACTION_DOWN, x, from)
        Thread.sleep(800)
        for (i in 1..4) {
            touch(MotionEvent.ACTION_MOVE, x, from + (to - from) * i / 4, wait = false)
            Thread.sleep(8)
        }
        touch(MotionEvent.ACTION_UP, x, to, wait = false)
    }

    private fun filter(filter: Filter) {
        runBlocking { engine.setFilter(PID, filter) }
        waitFor("the filter applies") { onScreen("2 hidden by the filter") }
    }

    private val screen get() = ins.targetContext.resources.displayMetrics

    // Slow and still before the release, so the list stops where the finger does.
    private fun scrollBy(dy: Float) {
        val x = screen.widthPixels / 2f
        val from = screen.heightPixels * 0.8f
        touch(MotionEvent.ACTION_DOWN, x, from)
        slide(x, from, from - dy, steps = 40, pause = 30)
        Thread.sleep(500)
        touch(MotionEvent.ACTION_UP, x, from - dy)
        Thread.sleep(500)
    }

    // Scrolls until the header starts the list, where the first one started.
    private fun bringToTop(header: String) {
        val top = bounds("Inbox").top
        while (!onScreen(header)) scrollBy(screen.heightPixels * 0.3f)
        scrollBy((bounds(header).top - top).toFloat())
    }

    @Test
    fun aRowDroppedJustAboveASeparatorStaysInTheSectionShown() {
        filter(OPEN)
        assertFalse(onScreen("Done"))
        drag("Call the bank", bounds("Next").exactCenterY() - bounds("Call the bank").exactCenterY())
        waitFor("Inbox", listOf("Plan the trip", "Call the bank", "Done", "Book the ferry", "Pack the bags", "Next", "Renew passport"))
        Thread.sleep(500)
        assertFalse("no separator appears", onScreen("Done"))
    }

    @Test
    fun underATagFilterARowDroppedJustAboveASeparatorStaysInTheSectionShown() {
        filter(Filter(tags = setOf("green")))
        assertFalse(onScreen("Done"))
        drag("Call the bank", bounds("Next").exactCenterY() - bounds("Call the bank").exactCenterY())
        waitFor("Inbox", listOf("Plan the trip", "Call the bank", "Done", "Book the ferry", "Pack the bags", "Next", "Renew passport"))
        Thread.sleep(500)
        assertFalse("no separator appears", onScreen("Done"))
    }

    @Test
    fun aRowMovedLastInACategoryLeavesItsHiddenTailHidden() {
        filter(OPEN)
        drag("Write the report", bounds("Review the slides").exactCenterY() - bounds("Write the report").exactCenterY())
        waitFor("Work", listOf("Review the slides", "Write the report", "Later", "Old ticket", "Older ticket"))
        Thread.sleep(500)
        assertFalse("no separator appears", onScreen("Later"))
    }

    @Test
    fun aRowDraggedUpAcrossAHeaderStaysAboveTheHiddenLine() {
        filter(OPEN)
        val row = bounds("Write the report")
        drag("Write the report", bounds("Work").exactCenterY() - row.top - row.height() * 0.15f)
        waitFor("Inbox", listOf("Plan the trip", "Done", "Book the ferry", "Pack the bags", "Next", "Call the bank", "Renew passport", "Write the report"))
        Thread.sleep(500)
        assertTrue(bounds("Write the report").bottom <= bounds("2 hidden by the filter").top)
    }

    @Test
    fun aRowDraggedJustIntoAnEmptyCategoryAboveStaysThere() {
        bringToTop("Work")
        val row = bounds("Fix the sink")
        drag("Fix the sink", bounds("Home").exactCenterY() - row.top - row.height() * 0.15f, hold = 1_500)
        waitFor("Empty", listOf("Fix the sink"))
        assertEquals(listOf("Paint the door", "Mow the lawn", "Clean the gutter"), titles("Home"))
    }

    @Test
    fun underAFilterARowDraggedJustIntoAnEmptyCategoryAboveStaysThere() {
        filter(OPEN)
        val row = bounds("Fix the sink")
        drag("Fix the sink", bounds("Home").exactCenterY() - row.top - row.height() * 0.15f, hold = 1_500)
        waitFor("Empty", listOf("Fix the sink"))
    }

    @Test
    fun aFlickedRowLandsWhereItIsLetGo() {
        flick("Renew passport", onto = "Book the ferry")
        waitFor("Inbox", listOf("Plan the trip", "Done", "Renew passport", "Book the ferry", "Pack the bags", "Next", "Call the bank"))
    }

    @Test
    fun underAFilterARowFlickedAcrossAHeaderLandsWhereItIsLetGo() {
        filter(OPEN)
        flick("Write the report", onto = "Renew passport")
        waitFor("Inbox", listOf("Plan the trip", "Done", "Book the ferry", "Pack the bags", "Next", "Call the bank", "Write the report", "Renew passport"))
    }

    @Test
    fun aRowFlickedUpAScrolledListLandsWhereItIsLetGo() {
        bringToTop("Reading")
        flick("Book 10", onto = "Book 4")
        waitFor("Reading", (listOf(1, 2, 3, 10) + (4..9) + listOf(11, 12)).map { "Book $it" })
    }
}
