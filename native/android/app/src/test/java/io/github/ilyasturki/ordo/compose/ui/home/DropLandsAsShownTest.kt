package io.github.ilyasturki.ordo.compose.ui.home

import io.github.ilyasturki.ordo.compose.core.perform
import io.github.ilyasturki.ordo.compose.model.Block
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Line
import io.github.ilyasturki.ordo.compose.model.Project
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.sync.PipeCore
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

private const val PID = "0rd0-web-fixture"
private const val FEATURE = "cccccccc-0000-4000-8000-000000000008"
private const val FIX = "cccccccc-0000-4000-8000-000000000012"
private const val DOCUMENTATION = "cccccccc-0000-4000-8000-000000000019"
private const val WEB_UI = "aaaaaaaa-0000-4000-8000-000000000001"

// Every drop of every row onto every slot of the seed, as the list shows it, run through the real core:
// the list the edit leaves is the one shown at the release.
class DropLandsAsShownTest {
    private val raw = File(System.getProperty("ordo.repo")!!, "testdata/native/seed.json").readText()
    private val core = PipeCore()

    @After
    fun tearDown() = core.close()

    private data class View(val name: String, val filter: Filter = Filter.NONE, val peek: String? = null, val folds: Set<String> = emptySet())

    private val views = listOf(
        View("all"),
        View("todo", Filter(statuses = setOf(Status.TODO))),
        View("a peek", Filter(statuses = setOf(Status.TODO)), peek = DOCUMENTATION),
        View("one category", Filter(categories = setOf(FEATURE))),
        View("folded categories", folds = setOf(FIX, DOCUMENTATION)),
        View("a folded row", folds = setOf(WEB_UI)),
    )

    private fun shape(lines: List<Line>) = lines.filter { it is Header || it is Block }.map { it.key }

    @Test
    fun everyDropLandsWhereItWasShown() {
        val failures = ArrayList<String>()
        var drops = 0
        for (v in views) {
            val lines = Board.lines(Project.parse(raw), v.filter, v.peek, folds = v.folds)
            for (block in lines.filterIsInstance<Block>()) {
                for ((t, over) in lines.withIndex()) {
                    if (over.key == block.key || !swappable(over, t)) continue
                    val what = "${v.name}: ${block.row.task.title} onto ${over.key}"
                    val order = moved(lines, block.key, over.key)!!
                    val op = dropOp(order, block, lines)
                    if (shape(order) == shape(lines)) {
                        if (op != null) failures += "$what: sends $op for an order it didn't change"
                        continue
                    }
                    if (op == null) {
                        failures += "$what: sends nothing"
                        continue
                    }
                    drops++
                    core.open(raw)
                    val answer = core.perform(PID, op)
                    if (!answer.applied) {
                        failures += "$what: $op answered ${answer.status}"
                        continue
                    }
                    val into = headerAbove(order, block)!!
                    val folds = if (into.folded) v.folds - into.category.id else v.folds
                    val held = if (v.filter.active) setOf(block.row.task.id) else emptySet()
                    val after = shape(Board.lines(Project.parse(core.project(PID)), v.filter, v.peek, held, folds))
                    // A separator or a note whose section the drop emptied hides with it.
                    val emptied = order.filterIsInstance<Block>().filter { it.row.task.line && it.key != block.key && it.key !in after }.map { it.key }
                    val shown = shape(order) - emptied.toSet()
                    val landed = if (into.folded) after.filter { it in shown } else after
                    val above = { s: List<String> -> s.getOrNull(s.indexOf(block.key) - 1) }
                    if (landed != shown || above(after) != above(shown)) failures += "$what: shown $shown, landed $after"
                }
            }
        }
        assertEquals("${failures.size} of $drops drops:\n" + failures.take(20).joinToString("\n"), 0, failures.size)
        assertTrue("only $drops drops", drops > 500)
    }
}
