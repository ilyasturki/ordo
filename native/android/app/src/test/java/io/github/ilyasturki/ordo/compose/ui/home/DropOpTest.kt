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
        assertEquals(listOf(header("fix"), "h:fix", header("empty"), row("e"), header("docs")), down.slice(down.indexOf(down.first { it.key == header("fix") }) until down.indexOfFirst { it.key == header("docs") } + 1).map { it.key })
        val up = down.drag("e", header("empty"))
        assertEquals("the row back above the line counting what the filter hides", listOf(header("fix"), row("e"), "h:fix", header("empty")), up.slice(up.indexOfFirst { it.key == header("fix") } until up.indexOfFirst { it.key == header("empty") } + 1).map { it.key })
        assertNull(dropOp(up, up.block("e"), lines))
        val last = lines.drag("h", header("docs"))
        assertEquals(Ops.place("h", "empty", ""), dropOp(last, last.block("h"), lines))
        assertEquals("h:docs", last.last().key)
    }

    @Test
    fun belowAFoldedHeaderARowLandsFirstThere() {
        val lines = lines(folds = setOf("fix"))
        val down = lines.drag("c", header("fix"))
        assertEquals(Ops.place("c", "fix", "e"), dropOp(down, down.block("c"), lines))
        assertEquals(true, headerAbove(down, down.block("c"))?.folded)
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
