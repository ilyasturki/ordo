package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.semantics.SemanticsNode
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.TouchInjectionScope
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.onAllNodesWithTag
import androidx.compose.ui.test.performScrollToKey
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.unit.dp
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.ilyasturki.ordo.compose.core.CoreApi
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.sync.Delays
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TestName
import org.junit.runner.RunWith
import kotlin.math.abs
import kotlin.math.ceil
import kotlin.math.max

private const val WEB = "Web UI for the phone"
private const val EXPORT = "Export a project as Markdown"
private const val LATER = "Later"
private const val RECURRING = "Recurring tasks"
private const val CURSOR = "Cursor jumps after a sync reload"
private const val WRAPPED = "Wrapped titles overflow the tag column"
private const val CRASH = "Crash on an empty clipboard"
private const val TAP = "Tap a row to focus it"
private const val WRITE = "Write the install guide"
private const val DOCUMENT = "Document the file format"
private const val SHOTS = "Screenshots for the README"
private const val EXPLAIN = "Explain the web app pairing"
private const val HTMX = "Compare htmx and plain fetch"
private const val MOTION = "Material 3 motion specs"

// Drags on the home list, each one gesture, checked on what the engine holds once it lands; the
// autoscroll's are in AutoscrollTest.
@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class ReorderTest : AppFixture(listOf("Ordo", "Spare")) {
    @get:Rule
    val name = TestName()

    private lateinit var door: Door

    override fun newCore(): CoreApi = Door(GoCore()).also { door = it }

    // No round retries under a moving finger, but in the tests of a sync tick.
    override val delays: Delays
        get() = if ("Sync" in name.methodName) super.delays else Delays(600_000, 600_000, 600_000, 600_000, 600_000)

    private val project get() = engine.state.value.project(PID)!!.project

    private fun cat(name: String) = project.categories.first { it.name == name }

    private fun rows(cat: String) = cat(cat).tasks.filter { it.parentId == "" }.map { it.title }

    private fun subtasks(title: String) = project.categories.flatMap { it.tasks }.let { all -> all.filter { it.parentId == all.first { t -> t.title == title }.id } }.map { it.title }

    private fun <T> waitFor(want: T, what: () -> T) {
        runCatching { compose.waitUntil(20_000) { what() == want } }.onFailure { assertEquals(want, what()) }
    }

    private val screen = InstrumentationRegistry.getInstrumentation().targetContext.resources.displayMetrics.widthPixels

    // The pager composes the next project's page beside this one, with the same rows.
    private fun SemanticsNode.shown() = boundsInRoot.left < screen

    private fun list(): SemanticsNodeInteraction {
        val all = compose.onAllNodesWithTag("list")
        return all[all.fetchSemanticsNodes().indexOfFirst { it.shown() }]
    }

    private fun top(key: String) = list().performScrollToKey(key)

    private fun row(title: String) = hasText(title) and hasClickAction()

    private fun header(name: String) = hasText(name) and hasClickAction()

    // Where the node starts and ends in the list, in pixels; null while it isn't on the screen.
    private fun span(matcher: SemanticsMatcher): ClosedFloatingPointRange<Float>? {
        val b = compose.onAllNodes(matcher).fetchSemanticsNodes().firstOrNull { it.shown() }?.boundsInRoot ?: return null
        val list = list().fetchSemanticsNode().boundsInRoot.top
        return (b.top - list)..(b.bottom - list)
    }

    private val ClosedFloatingPointRange<Float>.mid get() = (start + endInclusive) / 2

    // A row with the subtasks it shows: what a drag lifts.
    private fun block(title: String): ClosedFloatingPointRange<Float> {
        val spans = (listOf(title) + subtasks(title)).mapNotNull { span(row(it)) }
        return spans.minOf { it.start }..spans.maxOf { it.endInclusive }
    }

    private fun px(dp: Float) = with(compose.density) { dp.dp.toPx() }

    private fun height() = list().fetchSemanticsNode().size.height.toFloat()

    // Steps of a quarter row at most, two frames apart, so every swap lands before the next.
    private fun TouchInjectionScope.glide(dy: Float, dx: Float = 0f) {
        val n = max(12, ceil(abs(dy) / px(12f)).toInt())
        repeat(n) { moveBy(Offset(dx / n, dy / n), 40) }
    }

    private fun TouchInjectionScope.lift(at: Float) {
        down(Offset(centerX, at))
        advanceEventTime(1_000)
    }

    private fun TouchInjectionScope.release() {
        advanceEventTime(300)
        up()
    }

    private fun drag(title: String, dy: Float, dx: Float = 0f, by: String = title) {
        val at = span(row(by))!!.mid
        list().performTouchInput {
            lift(at)
            glide(dy, dx)
            release()
        }
        compose.waitForIdle()
    }

    // Lifts the row's block and lets it go where the line over shows.
    private fun dragOnto(title: String, over: ClosedFloatingPointRange<Float>, dx: Float = 0f) {
        val me = block(title)
        drag(title, if (over.start > me.start) over.endInclusive - me.endInclusive else over.start - me.start, dx)
    }

    private fun featureTop() = top("c:${cat("Feature").id}")

    @Test
    fun aRowMovesDownByOneAndBackUp() {
        top("c:${cat("Fix").id}")
        dragOnto(CURSOR, block(WRAPPED))
        waitFor(listOf(WRAPPED, CURSOR, CRASH)) { rows("Fix") }
        dragOnto(CURSOR, block(WRAPPED))
        waitFor(listOf(CURSOR, WRAPPED, CRASH)) { rows("Fix") }
        assertEquals(2, engine.state.value.pending)
    }

    @Test
    fun aRowCrossesAHeaderBothWays() {
        top("t:${task(LATER)!!.id}")
        dragOnto(RECURRING, span(header("Fix"))!!)
        waitFor(listOf(RECURRING, CURSOR, WRAPPED, CRASH)) { rows("Fix") }
        dragOnto(RECURRING, span(header("Fix"))!!)
        waitFor(listOf(WEB, EXPORT, LATER, RECURRING)) { rows("Feature") }
        dragOnto(CURSOR, span(header("Fix"))!!)
        waitFor(listOf(WEB, EXPORT, LATER, RECURRING, CURSOR)) { rows("Feature") }
    }

    @Test
    fun aRowMovesPastASeparatorAndTheSeparatorMoves() {
        featureTop()
        dragOnto(EXPORT, block(LATER))
        waitFor(listOf(WEB, LATER, EXPORT, RECURRING)) { rows("Feature") }
        dragOnto(RECURRING, block(LATER))
        waitFor(listOf(WEB, RECURRING, LATER, EXPORT)) { rows("Feature") }
        dragOnto(LATER, block(WEB))
        waitFor(listOf(LATER, WEB, RECURRING, EXPORT)) { rows("Feature") }
    }

    @Test
    fun aRowGoesPastATallBlockAndTheBlockMovesWhole() {
        featureTop()
        val kids = subtasks(WEB)
        dragOnto(RECURRING, block(WEB))
        waitFor(listOf(RECURRING, WEB, EXPORT, LATER)) { rows("Feature") }
        dragOnto(WEB, block(LATER))
        waitFor(listOf(RECURRING, EXPORT, LATER, WEB)) { rows("Feature") }
        assertEquals(kids, subtasks(WEB))
    }

    @Test
    fun aSubtaskLiftsItsParentsBlock() {
        featureTop()
        val me = block(WEB)
        drag(WEB, block(EXPORT).endInclusive - me.endInclusive, by = "Pairing link login")
        waitFor(listOf(EXPORT, WEB, LATER, RECURRING)) { rows("Feature") }
        assertEquals(listOf("Shared action layer", "Pairing link login", "Drag to reorder"), subtasks(WEB))
    }

    @Test
    fun underAFilterARowSkipsWhatItHidesAndTheHiddenLineIsNoPlace() {
        runBlocking { engine.setFilter(PID, Filter(statuses = setOf("todo"))) }
        compose.waitUntilAtLeastOneExists(hasText("2 hidden by the filter"), 20_000)
        featureTop()
        dragOnto(RECURRING, block(EXPORT))
        waitFor(listOf(WEB, RECURRING, EXPORT, LATER)) { rows("Feature") }
        val pending = engine.state.value.pending
        val hidden = compose.onAllNodes(hasText("2 hidden by the filter")).fetchSemanticsNodes().filter { it.shown() }
            .map { it.boundsInRoot.top - list().fetchSemanticsNode().boundsInRoot.top }.first { it > block(EXPORT).start }
        dragOnto(EXPORT, hidden..hidden + px(48f))
        compose.waitForIdle()
        Thread.sleep(1_000)
        assertEquals("a drop that moves nothing sends nothing", pending, engine.state.value.pending)
        assertEquals(listOf(WEB, RECURRING, EXPORT, LATER), rows("Feature"))
        dragOnto(EXPORT, span(header("Fix"))!!)
        waitFor(listOf(EXPORT, CURSOR, WRAPPED, CRASH)) { rows("Fix") }
    }

    @Test
    fun aRowDroppedWhereTheFilterHidesItStaysInSight() {
        runBlocking { engine.setFilter(PID, Filter(categories = setOf(cat("Feature").id))) }
        compose.waitUntilAtLeastOneExists(hasText("3 hidden by the filter"), 20_000)
        featureTop()
        dragOnto(RECURRING, span(header("Fix"))!!)
        waitFor(listOf(CURSOR, WRAPPED, CRASH, RECURRING)) { rows("Fix") }
        compose.waitForIdle()
        assertTrue(span(row(RECURRING))!!.start > span(header("Fix"))!!.start)
    }

    @Test
    fun aRowDroppedUnderAFoldedHeaderLandsFirstThereAndUnfoldsIt() {
        runBlocking { engine.setFolded(PID, cat("Fix").id, true) }
        compose.waitUntilAtLeastOneExists(hasContentDescription("Unfold Fix"), 20_000)
        featureTop()
        dragOnto(RECURRING, span(header("Fix"))!!)
        waitFor(listOf(RECURRING, CURSOR, WRAPPED, CRASH)) { rows("Fix") }
        waitFor(emptySet<String>()) { engine.state.value.views[PID]!!.folds }
        compose.waitForIdle()
        assertTrue(span(row(RECURRING))!!.endInclusive <= span(row(CURSOR))!!.start)
    }

    @Test
    fun aRowPushedAgainstTheFirstHeaderStillMovesRightAway() {
        featureTop()
        val at = span(row(WEB))!!.mid
        val up = px(48f)
        val down = up + (block(EXPORT).endInclusive - block(WEB).endInclusive)
        list().performTouchInput {
            lift(at)
            repeat(4) { moveBy(Offset(0f, -up / 4), 40) }
            glide(down)
            release()
        }
        waitFor(listOf(EXPORT, WEB, LATER, RECURRING)) { rows("Feature") }
    }

    @Test
    fun aRowMovesOnAScrolledList() {
        list().performScrollToNode(row(MOTION))
        dragOnto(WRITE, block(SHOTS))
        waitFor(listOf(DOCUMENT, SHOTS, WRITE, EXPLAIN)) { rows("Documentation") }
    }

    @Test
    fun aDragLiftedWhileTheLastDropStillLandsKeepsItsPlace() {
        top("c:${cat("Fix").id}")
        door.shut()
        dragOnto(CURSOR, block(WRAPPED))
        compose.waitUntil(20_000) { door.knocks.get() == 1 }
        val at = span(row(TAP))!!
        val dy = span(header("Ergonomy"))!!.start - at.start
        list().performTouchInput {
            lift(at.mid)
            glide(dy)
        }
        compose.waitForIdle()
        door.open()
        waitFor(listOf(WRAPPED, CURSOR, CRASH)) { rows("Fix") }
        compose.waitForIdle()
        list().performTouchInput { release() }
        waitFor(listOf(WRAPPED, CURSOR, CRASH, TAP)) { rows("Fix") }
    }

    @Test
    fun aSyncTickBeforeTheFirstSwapLeavesTheDropWhereItShows() {
        top("c:${cat("Fix").id}")
        val dy = block(CURSOR).start - block(WRAPPED).start
        val at = span(row(WRAPPED))!!.mid
        list().performTouchInput {
            lift(at)
            moveBy(Offset(0f, 1f), 40)
        }
        compose.waitForIdle()
        runBlocking { engine.edit(PID, Ops.add(cat("Feature").id, "Added elsewhere"), quiet = true) }
        compose.waitForIdle()
        list().performTouchInput {
            glide(dy)
            release()
        }
        waitFor(listOf(WRAPPED, CURSOR, CRASH)) { rows("Fix") }
        assertTrue("Added elsewhere" in rows("Feature"))
    }

    @Test
    fun aSyncTickAfterTheFirstSwapLeavesTheDropWhereItShows() {
        top("c:${cat("Fix").id}")
        val dy = block(CURSOR).start - block(WRAPPED).start
        val at = span(row(WRAPPED))!!.mid
        list().performTouchInput {
            lift(at)
            glide(dy)
        }
        compose.waitForIdle()
        runBlocking { engine.edit(PID, Ops.add(cat("Feature").id, "Added elsewhere"), quiet = true) }
        compose.waitForIdle()
        list().performTouchInput { release() }
        waitFor(listOf(WRAPPED, CURSOR, CRASH)) { rows("Fix") }
        assertTrue("Added elsewhere" in rows("Feature"))
    }

    @Test
    fun aRowReleasedAboveTheListLandsFirst() {
        featureTop()
        drag(RECURRING, -(span(row(RECURRING))!!.mid + px(80f)))
        waitFor(listOf(RECURRING, WEB, EXPORT, LATER)) { rows("Feature") }
    }

    @Test
    fun aRowReleasedPastTheLastRowLandsLast() {
        list().performScrollToNode(row(MOTION))
        drag(HTMX, height() - px(8f) - span(row(HTMX))!!.mid)
        waitFor(listOf(MOTION, HTMX)) { rows("Research") }
    }

    @Test
    fun aSidewaysDriftAfterTheLiftKeepsThePage() {
        featureTop()
        val current = engine.state.value.prefs.current
        dragOnto(RECURRING, block(EXPORT), dx = -px(200f))
        waitFor(listOf(WEB, RECURRING, EXPORT, LATER)) { rows("Feature") }
        compose.waitForIdle()
        assertEquals(current, engine.state.value.prefs.current)
        assertNotNull(span(row(RECURRING)))
    }
}
