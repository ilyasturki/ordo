package io.github.ilyasturki.ordo.compose.ui.home

import io.github.ilyasturki.ordo.compose.model.Block
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Hidden
import io.github.ilyasturki.ordo.compose.model.Line
import io.github.ilyasturki.ordo.compose.model.NoTasks
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Project
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.model.Task
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class DropOpTest {
    private fun dropOp(order: List<Line>, block: Block, lifted: List<Line>) = dropPlace(order, block, lifted)?.op

    private fun task(id: String, status: String = Status.TODO, parent: String = "", kind: String = "") =
        Task(id = id, parentId = parent, title = id, status = status, kind = kind)

    // Feature [a, b (b1), separator, c] with the sub-category UI [d], Fix [e, f done], an empty category, Docs [g done, h].
    private val project = Project(
        id = "p",
        categories = listOf(
            Category("feat", "Feature", tasks = listOf(task("a"), task("b"), task("b1", parent = "b"), task("sep", kind = "separator"), task("c"))),
            Category("ui", "UI", parentId = "feat", tasks = listOf(task("d"))),
            Category("fix", "Fix", tasks = listOf(task("e"), task("f", Status.COMPLETED))),
            Category("empty", "Empty"),
            Category("docs", "Docs", tasks = listOf(task("g", Status.COMPLETED), task("h"))),
        ),
    )

    private fun lines(filter: Filter = Filter.NONE, folds: Set<String> = emptySet()) = Board.lines(project, filter, folds = folds)

    private fun List<Line>.block(id: String) = first { it.key == "t:$id" } as Block

    private fun List<Line>.drag(id: String, over: String): List<Line> = moved(this, "t:$id", over)!!

    private fun drop(lifted: List<Line>, id: String, over: String) = dropOp(lifted.drag(id, over), lifted.block(id), lifted)

    private fun row(id: String) = "t:$id"

    private fun header(id: String) = "c:$id"

    @Test
    fun aDropLandsAboveTheNextRowUnderTheHeaderAboveIt() {
        val lines = lines()
        for ((name, drag, want) in listOf(
            Triple("down by one", Pair("a", row("b")), Ops.place("a", "feat", "sep")),
            Triple("up by one", Pair("f", row("e")), Ops.place("f", "fix", "e")),
            Triple("first under its header", Pair("c", row("a")), Ops.place("c", "feat", "a")),
            Triple("last above the next header", Pair("a", row("c")), Ops.place("a", "feat", "")),
            Triple("above a separator", Pair("c", row("sep")), Ops.place("c", "feat", "sep")),
            Triple("a dragged separator", Pair("sep", row("a")), Ops.place("sep", "feat", "a")),
            Triple("a row with its subtasks", Pair("b", row("a")), Ops.place("b", "feat", "a")),
            Triple("past a header going down: first of the next category", Pair("c", header("ui")), Ops.place("c", "ui", "d")),
            Triple("past a header going up: last of the previous one", Pair("e", header("fix")), Ops.place("e", "ui", "")),
            Triple("a parent category, above its first sub-category", Pair("d", header("ui")), Ops.place("d", "feat", "")),
            Triple("into an empty category from above", Pair("f", header("empty")), Ops.place("f", "empty", "")),
            Triple("into an empty category from below", Pair("g", header("docs")), Ops.place("g", "empty", "")),
        )) {
            assertEquals(name, want, drop(lines, drag.first, drag.second))
        }
    }

    @Test
    fun onlyRowsAndTheHeadersAfterTheFirstAreSlots() {
        val lines = lines(Filter(statuses = setOf(Status.TODO)))
        assertTrue(lines.any { it is Hidden } && lines.any { it is NoTasks })
        for ((i, line) in lines.withIndex()) {
            assertEquals(line.key, line is Block || (line is Header && i > 0), swappable(line, i))
        }
    }

    @Test
    fun aDropWhereItWasLiftedSendsNothing() {
        val lines = lines()
        assertNull(dropOp(lines, lines.block("a"), lines))
        val there = lines.drag("a", row("b"))
        val back = there.drag("a", row("b"))
        assertEquals(lines.map { it.key }, back.map { it.key })
        assertNull(dropOp(back, back.block("a"), lines))
        assertNull("measured against what the lift showed", dropOp(there, there.block("a"), there))
    }

    @Test
    fun underAFilterTheHiddenLineStaysLastOfItsCategory() {
        val lines = lines(Filter(statuses = setOf(Status.TODO)))
        val down = lines.drag("e", header("empty"))
        assertEquals(Ops.place("e", "empty", ""), dropOp(down, down.block("e"), lines))
        assertEquals("No tasks keeps its height under the row", listOf(header("fix"), "h:fix", header("empty"), row("e"), "n:empty", header("docs")), down.slice(down.indexOf(down.first { it.key == header("fix") }) until down.indexOfFirst { it.key == header("docs") } + 1).map { it.key })
        assertEquals(setOf("n:empty"), entered(down))
        val up = down.drag("e", header("empty"))
        assertEquals("the row back above the line counting what the filter hides", listOf(header("fix"), row("e"), "h:fix", header("empty")), up.slice(up.indexOfFirst { it.key == header("fix") } until up.indexOfFirst { it.key == header("empty") } + 1).map { it.key })
        assertNull(dropOp(up, up.block("e"), lines))
        assertEquals(emptySet<String>(), entered(up))
        val last = lines.drag("h", header("docs"))
        assertEquals(Ops.place("h", "empty", ""), dropOp(last, last.block("h"), lines))
        assertEquals(listOf(header("empty"), row("h"), "n:empty", header("docs"), "h:docs"), last.takeLast(5).map { it.key })
    }

    @Test
    fun theSameAskFromALayoutAFrameBehindLeavesTheOrder() {
        val lines = lines(Filter(statuses = setOf(Status.TODO)))
        val up = moved(lines, row("e"), header("fix"), up = true)!!
        assertEquals(up.map { it.key }, moved(up, row("e"), header("fix"), up = true)!!.map { it.key })
        val down = moved(lines, row("a"), row("b"), up = false)!!
        assertEquals(listOf(row("b"), row("a")), down.filter { it.key == row("a") || it.key == row("b") }.map { it.key })
        assertEquals(down.map { it.key }, moved(down, row("a"), row("b"), up = false)!!.map { it.key })
    }

    @Test
    fun aLiftedBlockNeverAddsNorDropsALine() {
        for (filter in listOf(Filter.NONE, Filter(statuses = setOf(Status.TODO)))) {
            val lines = lines(filter)
            for ((t, over) in lines.withIndex()) {
                if (!swappable(over, t)) continue
                for (block in lines.filterIsInstance<Block>()) {
                    if (block.key == over.key) continue
                    val order = lines.drag(block.row.task.id, over.key)
                    assertEquals("${block.key} onto ${over.key}", lines.map { it.key }.sorted(), order.map { it.key }.sorted())
                }
            }
        }
    }

    // Inbox [p, —done, b done, k done, —next, c, r], Work [w, v, —later, o done], Home [f], Loose [a, x done, y, z].
    private val sections = Project(
        id = "s",
        categories = listOf(
            Category("inbox", "Inbox", tasks = listOf(task("p"), task("done", kind = "separator"), task("b", Status.COMPLETED), task("k", Status.COMPLETED), task("next", kind = "separator"), task("c"), task("r"))),
            Category("work", "Work", tasks = listOf(task("w"), task("v"), task("later", kind = "separator"), task("o", Status.COMPLETED))),
            Category("home", "Home", tasks = listOf(task("f"))),
            Category("loose", "Loose", tasks = listOf(task("a"), task("x", Status.COMPLETED), task("y"), task("z"))),
        ),
    )

    @Test
    fun underAFilterADropStaysInTheSectionItShowsIn() {
        val lines = Board.lines(sections, Filter(statuses = setOf(Status.TODO)))
        assertTrue(lines.none { it.key == row("done") || it.key == row("later") })
        for ((name, drag, want) in listOf(
            Triple("up onto a separator: above the section hidden whole", Pair("c", row("next")), Ops.place("c", "inbox", "done")),
            Triple("down to the last row: above the hidden tail", Pair("w", row("v")), Ops.place("w", "work", "later")),
            Triple("up past a header: above the hidden tail", Pair("f", header("home")), Ops.place("f", "work", "later")),
            Triple("up past hidden rows: as K, right above the next row", Pair("z", row("y")), Ops.place("z", "loose", "y")),
            Triple("down past hidden rows: as J, right below the last row", Pair("a", row("y")), Ops.place("a", "loose", "z")),
            Triple("first under its header", Pair("r", row("p")), Ops.place("r", "inbox", "p")),
        )) {
            assertEquals(name, want, drop(lines, drag.first, drag.second))
        }
    }

    @Test
    fun theProjectAsADropLeavesIt() {
        val placed = project.placed(Drop("b", "fix", "f", false))
        assertEquals(listOf("a", "sep", "c"), placed.categories.first { it.id == "feat" }.tasks.map { it.id })
        assertEquals(listOf("e", "b", "b1", "f"), placed.categories.first { it.id == "fix" }.tasks.map { it.id })
        assertEquals(listOf("g", "h", "a"), project.placed(Drop("a", "docs", "", false)).categories.first { it.id == "docs" }.tasks.map { it.id })
    }

    @Test
    fun belowAFoldedHeaderARowLandsFirstThere() {
        val lines = lines(folds = setOf("fix"))
        val down = lines.drag("c", header("fix"))
        assertEquals(Drop("c", "fix", "e", unfold = true), dropPlace(down, down.block("c"), lines))
        val up = lines.drag("h", header("empty"))
        assertEquals(Ops.place("h", "fix", "e"), dropOp(up, up.block("h"), lines))
    }

    @Test
    fun aFoldedParentTakesTheDropItself() {
        val lines = lines(folds = setOf("feat"))
        assertEquals(listOf(header("feat"), header("fix")), lines.take(2).map { it.key })
        val up = lines.drag("e", header("fix"))
        assertEquals(Ops.place("e", "feat", "a"), dropOp(up, up.block("e"), lines))
    }
}
