package io.github.ilyasturki.ordo.compose

import android.graphics.Rect
import android.os.SystemClock
import android.view.InputDevice
import android.view.MotionEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage
import io.github.ilyasturki.ordo.compose.sync.Delays
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import kotlin.math.abs

// The autoscroll runs on the app's own dispatchers and its own clock: a compose rule's eager effects end
// its loop before it starts, so these drags go through the system's input in real time. Where a held row
// lands tells how far the list went under it.
@RunWith(AndroidJUnit4::class)
class AutoscrollTest : AppFixture(composing = false) {
    override val delays = Delays(600_000, 600_000, 600_000, 600_000, 600_000)

    private val ins get() = InstrumentationRegistry.getInstrumentation()

    private val density get() = ins.targetContext.resources.displayMetrics.density

    private val project get() = engine.state.value.project(PID)!!.project

    private fun categoryOf(title: String) = project.categories.first { c -> c.tasks.any { it.title == title } }.name

    private fun waitFor(what: String, check: () -> Boolean) {
        val until = SystemClock.uptimeMillis() + 20_000
        while (!check()) {
            assertTrue(what, SystemClock.uptimeMillis() < until)
            Thread.sleep(100)
        }
    }

    // The rows and headers holding the text: a title's node sits in its clickable row, the top bar's name of the
    // category doesn't. Each node is read afresh, as the cached tree lags a scroll behind.
    private fun rows(text: String): List<AccessibilityNodeInfo> {
        val found = ArrayList<AccessibilityNodeInfo>()
        val next = ArrayDeque(listOfNotNull(ins.uiAutomation.rootInActiveWindow))
        while (next.isNotEmpty()) {
            val n = next.removeFirst()
            if (!n.refresh()) continue
            if (n.text?.toString() == text) n.parent?.takeIf { it.refresh() && it.isClickable }?.let(found::add)
            for (i in 0 until n.childCount) n.getChild(i)?.let(next::add)
        }
        return found
    }

    private fun onScreen(text: String) = rows(text).any { n -> !Rect().also(n::getBoundsInScreen).isEmpty }

    // The row's bounds on the screen, once the app shows it.
    private fun bounds(text: String): Rect {
        waitFor("$text shows") { onScreen(text) }
        return rows(text).map { n -> Rect().also(n::getBoundsInScreen) }.first { !it.isEmpty }
    }

    private fun screen(): Rect {
        var r = Rect()
        ins.runOnMainSync {
            val shown = ActivityLifecycleMonitorRegistry.getInstance().getActivitiesInStage(Stage.RESUMED).single()
            r = Rect(0, 0, shown.window.decorView.width, shown.window.decorView.height)
        }
        return r
    }

    private fun navigationBar(): Int {
        var bottom = 0
        ins.runOnMainSync {
            val shown = ActivityLifecycleMonitorRegistry.getInstance().getActivitiesInStage(Stage.RESUMED).single()
            bottom = ViewCompat.getRootWindowInsets(shown.window.decorView)!!.getInsets(WindowInsetsCompat.Type.navigationBars()).bottom
        }
        return bottom
    }

    private var downAt = 0L

    private fun touch(action: Int, x: Float, y: Float) {
        val now = SystemClock.uptimeMillis()
        if (action == MotionEvent.ACTION_DOWN) downAt = now
        val event = MotionEvent.obtain(downAt, now, action, x, y, 0).apply { source = InputDevice.SOURCE_TOUCHSCREEN }
        ins.uiAutomation.injectInputEvent(event, true)
        event.recycle()
    }

    private fun slide(x: Float, from: Float, to: Float, steps: Int = 20) {
        for (i in 1..steps) {
            touch(MotionEvent.ACTION_MOVE, x, from + (to - from) * i / steps)
            Thread.sleep(30)
        }
    }

    // A long press on the row by, a slide to y, a hold there with the finger trembling, the release.
    private fun hold(by: String, y: Float) {
        val at = bounds(by)
        val x = at.exactCenterX()
        touch(MotionEvent.ACTION_DOWN, x, at.exactCenterY())
        Thread.sleep(800)
        slide(x, at.exactCenterY(), y)
        val until = SystemClock.uptimeMillis() + 4_000
        var i = 0
        while (SystemClock.uptimeMillis() < until) {
            touch(MotionEvent.ACTION_MOVE, x, y + if (i++ % 2 == 0) 1f else -1f)
            Thread.sleep(50)
        }
        touch(MotionEvent.ACTION_UP, x, y)
    }

    // Slow and still before the release, so the list stops where the finger does.
    private fun scroll(dy: Float) {
        val x = screen().exactCenterX()
        val from = screen().bottom * 0.75f
        touch(MotionEvent.ACTION_DOWN, x, from)
        slide(x, from, from - dy, steps = 40)
        Thread.sleep(500)
        touch(MotionEvent.ACTION_UP, x, from - dy)
        Thread.sleep(500)
    }

    private fun aboveTheNavigationBar() = screen().bottom - navigationBar() - 24 * density

    @Test
    fun aRowHeldJustAboveTheNavigationBarScrollsTheListDown() {
        hold("Export a project as Markdown", aboveTheNavigationBar())
        waitFor("the row lands past the first screen") { categoryOf("Export a project as Markdown") !in setOf("Feature", "Fix") }
        assertTrue("the list scrolled", !onScreen("Web UI for the phone"))
    }

    @Test
    fun aTallBlockHeldByItsLastSubtaskScrollsTheListDown() {
        hold("Drag to reorder", aboveTheNavigationBar())
        waitFor("the block lands past the first screen") { categoryOf("Web UI for the phone") !in setOf("Feature", "Fix") }
        assertEquals(listOf("Shared action layer", "Pairing link login", "Drag to reorder"), project.categories.flatMap { it.tasks }.let { all -> all.filter { it.parentId == all.first { t -> t.title == "Web UI for the phone" }.id } }.map { it.title })
    }

    @Test
    fun aRowLiftedAtTheTopOfAScrolledListDoesntScrollItBack() {
        val top = bounds("Feature").top.toFloat()
        val x = screen().exactCenterX()
        scroll(screen().height() * 0.4f)
        scroll(bounds("Crash on an empty clipboard").top - top)
        val crash = bounds("Crash on an empty clipboard")
        assertTrue("the row starts the list", abs(crash.top - top) < 20 * density)
        touch(MotionEvent.ACTION_DOWN, x, crash.exactCenterY())
        Thread.sleep(800)
        slide(x, crash.exactCenterY(), crash.exactCenterY() + 12 * density, steps = 4)
        Thread.sleep(1_500)
        assertTrue("the list stayed", !onScreen("Wrapped titles overflow the tag column"))
        val ergonomy = bounds("Ergonomy")
        slide(x, crash.exactCenterY() + 12 * density, crash.exactCenterY() + (ergonomy.bottom - crash.bottom))
        Thread.sleep(300)
        touch(MotionEvent.ACTION_UP, x, crash.exactCenterY() + (ergonomy.bottom - crash.bottom))
        waitFor("the row lands first in Ergonomy") { categoryOf("Crash on an empty clipboard") == "Ergonomy" }
    }

    @Test
    fun aRowHeldAtTheTopScrollsTheListUp() {
        val top = bounds("Feature").top.toFloat()
        val x = screen().exactCenterX()
        repeat(3) {
            touch(MotionEvent.ACTION_DOWN, x, screen().bottom * 0.8f)
            slide(x, screen().bottom * 0.8f, top + 10, steps = 5)
            touch(MotionEvent.ACTION_UP, x, top + 10)
            Thread.sleep(300)
        }
        waitFor("the list reached its end") { onScreen("Material 3 motion specs") && !onScreen("Web UI for the phone") }
        Thread.sleep(1_000)
        hold("Compare htmx and plain fetch", top + 24 * density)
        waitFor("the row lands on the first screen") { categoryOf("Compare htmx and plain fetch") == "Feature" }
    }
}
