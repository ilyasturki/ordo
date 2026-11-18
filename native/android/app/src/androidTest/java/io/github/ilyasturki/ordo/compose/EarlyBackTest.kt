package io.github.ilyasturki.ordo.compose

import android.os.Bundle
import android.os.SystemClock
import android.view.KeyEvent
import android.view.accessibility.AccessibilityNodeInfo
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

// One back at every moment of the keyboard's start, through the system's input in real time: the keyboard takes a back
// while it starts, and the sheet still closes on it and keeps what was typed.
@RunWith(AndroidJUnit4::class)
class EarlyBackTest : AppFixture(composing = false) {
    private val ins get() = InstrumentationRegistry.getInstrumentation()

    private fun find(match: (AccessibilityNodeInfo) -> Boolean): AccessibilityNodeInfo? {
        val next = ArrayDeque(listOfNotNull(ins.uiAutomation.rootInActiveWindow))
        while (next.isNotEmpty()) {
            val n = next.removeFirst()
            if (!n.refresh()) continue
            if (match(n)) return n
            for (i in 0 until n.childCount) n.getChild(i)?.let(next::add)
        }
        return null
    }

    private fun described(text: String) = find { it.contentDescription?.toString() == text }

    private fun waitFor(what: String, ms: Long = 20_000, check: () -> Boolean) {
        val until = SystemClock.uptimeMillis() + ms
        while (!check()) {
            assertTrue(what, SystemClock.uptimeMillis() < until)
            Thread.sleep(20)
        }
    }

    private fun backWhileTheKeyboardStarts(open: () -> Unit, text: String, wait: Long, saved: () -> Boolean) {
        open()
        waitFor("the sheet opens") { described("Close sheet") != null }
        find { it.isEditable }!!.performAction(
            AccessibilityNodeInfo.ACTION_SET_TEXT,
            Bundle().apply { putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE, text) },
        )
        Thread.sleep(wait)
        ins.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
        waitFor("one back $wait ms in closes the sheet", 4_000) { described("Close sheet") == null }
        waitFor("one back $wait ms in saves $text", 4_000, saved)
        Thread.sleep(800)
    }

    @Test
    fun oneBackOnTheAddSheetClosesItAndSaves() {
        waitFor("the list shows") { described("Add a task") != null }
        for (wait in 0L..150L step 10) {
            backWhileTheKeyboardStarts({ described("Add a task")!!.performAction(AccessibilityNodeInfo.ACTION_CLICK) }, "Early $wait", wait) { task("Early $wait") != null }
        }
    }

    @Test
    fun oneBackOnTheNameSheetClosesItAndSaves() {
        waitFor("the list shows") { described("Add a task") != null }
        for (wait in 0L..150L step 30) {
            val name = engine.state.value.project(PID)!!.name
            backWhileTheKeyboardStarts(
                {
                    val title = find { it.text?.toString() == name }!!
                    (if (title.isClickable) title else title.parent).performAction(AccessibilityNodeInfo.ACTION_CLICK)
                },
                "Ordo $wait",
                wait,
            ) { engine.state.value.project(PID)?.name == "Ordo $wait" }
        }
        assertEquals("Ordo 150", engine.state.value.project(PID)!!.name)
    }
}
