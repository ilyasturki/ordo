package io.github.ilyasturki.ordo.compose.ui.home

import io.github.ilyasturki.ordo.compose.core.perform
import io.github.ilyasturki.ordo.compose.model.Block
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Line
import io.github.ilyasturki.ordo.compose.model.Movable
import io.github.ilyasturki.ordo.compose.model.Project
import io.github.ilyasturki.ordo.compose.model.Section
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
private const val INBOX = "cccccccc-0000-4000-8000-000000000018"
private const val HOME = "cccccccc-0000-4000-8000-000000000031"
private const val SHAPES = "cccccccc-0000-4000-8000-000000000010"
private const val LATER = "bbbbbbbb-0000-4000-8000-000000000006"
private const val DONE = "dddddddd-0000-4000-8000-000000000012"
private const val MID = "dddddddd-0000-4000-8000-000000000004"

// Every drop of every row and section line onto every slot of the seed and of drag.json, as the list shows it, run through the
// real core: the list the edit leaves is the one shown at the release, and the one the release lays out.
class DropLandsAsShownTest {
    private val core = PipeCore()

    @After
    fun tearDown() = core.close()

    private data class View(val name: String, val filter: Filter = Filter.NONE, val peek: String? = null, val folds: Set<String> = emptySet())

    private val todo = Filter(statuses = setOf(Status.TODO))

    private val fixtures = mapOf(
        "seed.json" to listOf(
            View("all"),
            View("todo", todo),
            View("a peek", todo, peek = DOCUMENTATION),
            View("high", Filter(priorities = setOf("high"))),
            View("a revealed section", Filter(priorities = setOf("high")), peek = LATER),
            View("one category", Filter(categories = setOf(FEATURE))),
            View("folded categories", folds = setOf(FIX, DOCUMENTATION)),
            View("a folded row", folds = setOf(WEB_UI)),
        ),
        // Sections the filters hide whole: at a category's head, in its middle, at its tail; an empty category,
        // a separator and a note shown by their own tag, a sub-category.
        "drag.json" to listOf(
            View("all"),
            View("todo", todo),
            View("open", Filter(statuses = setOf(Status.TODO, Status.IN_PROGRESS))),
            View("green", Filter(tags = setOf("green"))),
            View("blue", Filter(tags = setOf("blue"))),
            View("a peek", todo, peek = SHAPES),
            View("a revealed section", todo, peek = DONE),
            View("a revealed section with a note", todo, peek = MID),
            View("one category", Filter(categories = setOf(INBOX))),
            View("folded categories", folds = setOf(HOME, SHAPES)),
        ),
    )

    private fun shape(lines: List<Line>) = lines.filter { it is Header || it is Movable }.map { it.key }

    private fun names(lines: List<Line>) = lines.map {
        when (it) {
            is Header -> "[${it.category.name}]"
            is Block -> it.row.task.title
            is Section -> "<${it.row.task.title} ${it.count}${if (it.revealed) " shown" else ""}>"
            else -> it.key.take(1)
        }
    }

    @Test
    fun everyDropLandsWhereItWasShown() {
        val failures = ArrayList<String>()
        var drops = 0
        for ((file, views) in fixtures) {
            val raw = File(System.getProperty("ordo.repo")!!, "testdata/native/$file").readText()
            val project = Project.parse(raw)
            for (v in views) {
                val lines = Board.lines(project, v.filter, v.peek, folds = v.folds)
                for (block in lines.filterIsInstance<Movable>()) {
                    for ((t, over) in lines.withIndex()) {
                        if (over.key == block.key || !swappable(over, t)) continue
                        val what = "$file ${v.name}: ${block.row.task.title} onto ${over.key}"
                        val order = moved(lines, block.key, over.key)!!
                        val drop = dropPlace(order, block, lines)
                        val op = drop?.op
                        if (shape(order) == shape(lines)) {
                            if (op != null) failures += "$what: sends $op for an order it didn't change"
                            continue
                        }
                        if (drop == null || op == null) {
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
                        val folds = if (drop.unfold) v.folds - drop.cat else v.folds
                        val held = if (v.filter.active && block is Block) setOf(block.row.task.id) else emptySet()
                        val editedProject = Project.parse(core.project(PID))
                        val edited = Board.lines(editedProject, v.filter, v.peek, held, folds)
                        val laid = Board.lines(project.placed(drop), v.filter, v.peek, held, folds)
                        if (names(laid) != names(edited)) failures += "$what: the release lays out ${names(laid)}, the edit leaves ${names(edited)}"
                        // A separator dragged off hands its rows to the section above, whose lines show with them.
                        val left = project.categories.first { c -> c.tasks.any { it.id == block.row.task.id } }.tasks
                        val k = left.indexOfFirst { it.id == block.row.task.id }
                        val joined = if (!block.row.task.separator) emptySet() else left.subList(maxOf(0, left.subList(0, k).indexOfLast { it.separator }), k).filter { it.line }.map { "t:${it.id}" }.toSet()
                        // A note shows with the section the row lands in, wherever the row goes in it.
                        val into = Project.parse(core.project(PID)).categories.first { it.id == drop.cat }.tasks
                        val x = into.indexOfFirst { it.id == block.row.task.id }
                        val start = maxOf(0, into.subList(0, x).indexOfLast { it.separator })
                        val end = into.subList(x + 1, into.size).indexOfFirst { it.separator }.let { if (it < 0) into.size else x + 1 + it }
                        val notes = into.subList(start, end).filter { it.note }.map { "t:${it.id}" }.toSet()
                        // A peek reveals the rows its line counts: a separator the drop turns plain, or into a line, takes rows out of its reach or into it.
                        val peeked = shape(lines).toSet() - shape(Board.lines(project, v.filter, folds = v.folds)).toSet() - block.key
                        val revealed = shape(edited).toSet() - shape(Board.lines(editedProject, v.filter, null, held, folds)).toSet()
                        val after = shape(edited).filter { (it !in joined && it !in notes && it !in revealed) || it in shape(order) }
                        // A separator or a note whose section the drop emptied hides with it.
                        val emptied = order.filterIsInstance<Movable>().filter { it.row.task.line && it.key != block.key && it.key !in after }.map { it.key }
                        val shown = shape(order) - emptied.toSet() - (peeked - after.toSet())
                        val landed = if (drop.unfold) after.filter { it in shown } else after
                        val above = { s: List<String> -> s.getOrNull(s.indexOf(block.key) - 1) }
                        val title = { s: List<String> -> s.map { k -> lines.first { it.key == k }.let { names(listOf(it)).single() } } }
                        if (landed != shown || above(after) != above(shown)) failures += "$what: shown ${title(shown)}, landed ${names(edited.filter { it is Header || it is Movable })}"
                    }
                }
            }
        }
        assertEquals("${failures.size} of $drops drops:\n" + failures.take(20).joinToString("\n"), 0, failures.size)
        assertTrue("only $drops drops", drops > 1_000)
    }
}
