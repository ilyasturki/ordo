package io.github.ilyasturki.ordo.compose.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class BoardTest {
    private fun task(id: String, status: String = Status.TODO, parent: String = "", priority: String = "", tag: String = "", kind: String = "") =
        Task(id = id, parentId = parent, title = id, status = status, priority = priority, tagColor = tag, kind = kind)

    private val project = Project(
        id = "p",
        name = "P",
        categories = listOf(
            Category(
                "feat", "Feature",
                tasks = listOf(
                    task("a", Status.IN_PROGRESS),
                    task("a1", Status.COMPLETED, parent = "a"),
                    task("a2", parent = "a", priority = "high"),
                    task("sep", kind = "separator"),
                    task("b", Status.COMPLETED),
                    task("c", tag = "green-diamond"),
                ),
            ),
            Category("ui", "UI", parentId = "feat", tasks = listOf(task("d", Status.CANCELLED))),
            Category("fix", "Fix", tasks = listOf(task("e", priority = "high"))),
        ),
    )

    private fun shape(lines: List<Line>): List<String> = lines.map {
        when (it) {
            is Header -> "# ${it.category.name}" + if (it.parentName != "") " < ${it.parentName}" else ""
            is Block -> listOf(it.row.task.id + if (it.row.muted) "~" else "")
                .plus(it.children.map { c -> "  " + c.task.id + if (c.muted) "~" else "" })
                .joinToString(",")
            is Hidden -> "(${it.count} hidden${if (it.peeking) ", peeking" else ""})"
        }
    }

    @Test
    fun finishedRowsStayInPlaceWithNoFilter() {
        assertEquals(
            listOf("# Feature", "a,  a1,  a2", "sep", "b", "c", "# UI < Feature", "d", "# Fix", "e"),
            shape(Board.lines(project, Filter.NONE)),
        )
    }

    @Test
    fun aParentShowsWhenAChildMatchesAndSeparatorsHide() {
        val f = Filter(priorities = setOf("high"))
        assertEquals(
            listOf("# Feature", "a,  a2", "(3 hidden)", "# UI < Feature", "(1 hidden)", "# Fix", "e"),
            shape(Board.lines(project, f)),
        )
    }

    @Test
    fun aSelectedCategoryKeepsItsSubCategories() {
        val f = Filter(categories = setOf("feat"))
        assertTrue(f.categoryVisible(project.categories[1]))
        assertFalse(f.categoryVisible(project.categories[2]))
        assertEquals(
            listOf("# Feature", "a,  a1,  a2", "b", "c", "# UI < Feature", "d", "# Fix", "(1 hidden)"),
            shape(Board.lines(project, f)),
        )
    }

    @Test
    fun theNoPriorityAndUntaggedBucketsAreEmpty() {
        val f = Filter(priorities = setOf(""), tags = setOf(""))
        assertEquals(listOf("a", "a1", "b", "d"), Board.lines(project, f).filterIsInstance<Block>().flatMap { listOf(it.row) + it.children }.map { it.task.id })
    }

    @Test
    fun aDiamondTagMatchesItsColoursDot() {
        val f = Filter(tags = setOf("green"))
        assertEquals(listOf("c"), Board.lines(project, f).filterIsInstance<Block>().map { it.row.task.id })
    }

    @Test
    fun aPeekRevealsTheCategoryMuted() {
        val f = Filter(statuses = setOf(Status.TODO))
        assertEquals(
            listOf("# Feature", "a~,  a1~,  a2", "sep", "b~", "c", "(2 hidden, peeking)", "# UI < Feature", "(1 hidden)", "# Fix", "e"),
            shape(Board.lines(project, f, peek = "feat")),
        )
    }

    @Test
    fun aHeldRowStaysMutedAndIsNotCountedHidden() {
        val f = Filter(statuses = setOf(Status.TODO))
        assertEquals(
            listOf("# Feature", "a,  a2", "c", "(2 hidden)"),
            shape(Board.lines(project, f)).take(4),
        )
        assertEquals(
            listOf("# Feature", "a,  a2", "b~", "c", "(1 hidden)"),
            shape(Board.lines(project, f, held = setOf("b"))).take(5),
        )
    }

    @Test
    fun aFilterDropsCategoriesThatAreGone() {
        val f = Filter(categories = setOf("feat", "gone"))
        assertEquals(setOf("feat"), f.pruned(project).categories)
    }
}
