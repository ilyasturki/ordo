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
            is NoTasks -> "(no tasks)"
        }
    }

    @Test
    fun anEmptyCategoryWithoutSubCategoriesSaysSo() {
        val empty = Project(
            id = "p",
            categories = listOf(
                Category("a", "A"),
                Category("b", "B"),
                Category("b1", "B1", parentId = "b"),
                Category("c", "C", tasks = listOf(task("s", kind = "separator"))),
            ),
        )
        assertEquals(
            listOf("# A", "(no tasks)", "# B", "# B1 < B", "(no tasks)", "# C", "s"),
            shape(Board.lines(empty, Filter.NONE)),
        )
        assertEquals(
            listOf("# A", "(no tasks)", "# B", "# B1 < B", "(no tasks)", "# C", "(no tasks)"),
            shape(Board.lines(empty, Filter(statuses = setOf(Status.TODO)))),
        )
    }

    @Test
    fun aCategoryWhoseRowsTheFilterHidesCountsThemInstead() {
        assertEquals(
            listOf("# Feature", "(5 hidden)", "# UI < Feature", "(1 hidden)", "# Fix", "(1 hidden)"),
            shape(Board.lines(project, Filter(tags = setOf("blue")))),
        )
    }

    @Test
    fun finishedRowsStayInPlaceWithNoFilter() {
        assertEquals(
            listOf("# Feature", "a,  a1,  a2", "sep", "b", "c", "# UI < Feature", "d", "# Fix", "e"),
            shape(Board.lines(project, Filter.NONE)),
        )
    }

    private fun titled(id: String, title: String, description: String = "", parent: String = "", kind: String = "") =
        Task(id = id, parentId = parent, title = title, description = description, kind = kind)

    private val searched = Project(
        id = "p",
        name = "P",
        categories = listOf(
            Category(
                "feat", "Feature",
                tasks = listOf(
                    titled("a", "Parent"),
                    titled("a1", "Pair a phone", parent = "a"),
                    titled("a2", "Other child", parent = "a"),
                    titled("sep", "Later", kind = "separator"),
                    titled("b", "Done", "Reach ordo\n![holiday](attachments/00000000000000000000000000000000.png)"),
                ),
            ),
            Category("docs", "Docs", tasks = listOf(titled("c", "Guide"), titled("d", "Notes"))),
        ),
    )

    private fun found(query: String, filter: Filter = Filter.NONE) = shape(Board.search(Board.lines(searched, filter), query))

    @Test
    fun aSearchKeepsTheRowsThatMatchUnderTheirCategory() {
        assertEquals(listOf("# Feature", "a,  a1"), found("pair"))
        assertEquals(listOf("# Feature", "sep"), found("later"))
        assertEquals(listOf("# Feature", "b"), found("reach"))
        assertEquals(emptyList<String>(), found("nothing like this"))
    }

    @Test
    fun aSearchIsSmartcase() {
        assertEquals(listOf("# Feature", "a,  a1"), found("pair"))
        assertEquals(listOf("# Feature", "a,  a1"), found("Pair"))
        assertEquals(emptyList<String>(), found("PAIR"))
    }

    @Test
    fun aSearchMatchesACategoryNameWithAllItsRows() {
        assertEquals(listOf("# Docs", "c", "d"), found("docs"))
    }

    @Test
    fun aSearchReadsAnImageByItsLabelNotItsFileName() {
        assertEquals(listOf("# Feature", "b"), found("▣ holiday"))
        assertEquals(emptyList<String>(), found("attachments/"))
    }

    @Test
    fun aSearchStaysInsideTheFilterAndCountsNothingHidden() {
        val f = Filter(statuses = setOf(Status.COMPLETED))
        assertEquals(emptyList<String>(), found("pair", f))
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
            listOf("# Feature", "a,  a1~,  a2", "sep", "b~", "c", "(2 hidden, peeking)", "# UI < Feature", "(1 hidden)", "# Fix", "e"),
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
