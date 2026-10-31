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

    private val lines = listOf(
        Header(project.categories[0], ""),
        Block("feat", Row(task("a")), emptyList()),
        Header(project.categories[1], "Feature"),
        Block("ui", Row(task("d")), emptyList()),
        Header(project.categories[2], ""),
        Block("fix", Row(task("e")), emptyList()),
    )

    @Test
    fun theFocusIsTheCategoryWithTheMostPixelsAcrossItsRows() {
        assertEquals("fix", Board.focused(lines, listOf(Seen(1, 0, 100), Seen(4, 100, 60), Seen(5, 160, 60)), 0, 400))
        assertEquals("feat", Board.focused(lines, listOf(Seen(1, 0, 100), Seen(4, 100, 40), Seen(5, 140, 50)), 0, 400))
    }

    @Test
    fun onlyTheVisiblePartOfARowCounts() {
        val seen = listOf(Seen(1, -300, 400), Seen(5, 100, 200))
        assertEquals("fix", Board.focused(lines, seen, 0, 300))
        assertEquals("fix", Board.focused(lines, listOf(Seen(1, -90, 100), Seen(5, 200, 200)), 0, 250))
    }

    @Test
    fun aSubCategoryIsItsOwnFocus() {
        assertEquals("ui", Board.focused(lines, listOf(Seen(1, 0, 50), Seen(2, 50, 40), Seen(3, 90, 60)), 0, 300))
    }

    @Test
    fun aTieGoesToTheUpperCategoryAndNothingOnScreenIsNoFocus() {
        assertEquals("feat", Board.focused(lines, listOf(Seen(1, 0, 50), Seen(5, 50, 50)), 0, 300))
        assertEquals(null, Board.focused(lines, emptyList(), 0, 300))
        assertEquals(null, Board.focused(lines, listOf(Seen(99, 0, 50)), 0, 300))
    }

    @Test
    fun theTargetIsTheFocusUnlessTheFilterHidesIt() {
        assertEquals("fix", Board.target(project, Filter.NONE, "fix")?.id)
        assertEquals("feat", Board.target(project, Filter.NONE, null)?.id)
        assertEquals("feat", Board.target(project, Filter.NONE, "gone")?.id)
        assertEquals("fix", Board.target(project, Filter(categories = setOf("fix")), "feat")?.id)
        assertEquals("ui", Board.target(project, Filter(categories = setOf("feat")), "ui")?.id)
        assertEquals(null, Board.target(Project(id = "p", name = "P"), Filter.NONE, null))
    }

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
            Category("idle", "Idle"),
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
        assertEquals(listOf("# Idle", "(no tasks)"), found("idle"))
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

    @Test
    fun theFilterHoldsOneBucketValueOnDiskAndReadsBoth() {
        val filter = Filter(priorities = setOf("", "high"), tags = setOf(""))
        val raw = json.encodeToString(Filter.serializer(), filter)
        assertTrue(raw, raw.contains("\"none\"") && !raw.contains("\"\""))
        assertEquals(filter, json.decodeFromString(Filter.serializer(), raw))
        assertEquals(filter, json.decodeFromString(Filter.serializer(), """{"priorities":["","high"],"tags":[""]}"""))
    }

    @Test
    fun aViewEditKeepsWhatItLeavesOut() {
        val view = View(setOf("c1"), Filter(statuses = setOf("todo")))
        assertEquals(View(emptySet(), view.filter), ViewEdit(folds = emptySet()).onto(view))
        assertEquals(View(view.folds), ViewEdit(filter = Filter()).onto(view))
        assertEquals(ViewEdit(setOf("c2"), Filter()), ViewEdit(setOf("c1"), Filter()).then(ViewEdit(folds = setOf("c2"))))
    }

    @Test
    fun aViewEditSaysWhatItClears() {
        assertEquals("""{"folds":[]}""", json.encodeToString(ViewEdit.serializer(), ViewEdit(folds = emptySet())))
        assertEquals("""{"filter":{}}""", json.encodeToString(ViewEdit.serializer(), ViewEdit(filter = Filter())))
    }

    @Test
    fun aFoldedCategoryShowsItsHeaderAlone() {
        val lines = Board.lines(project, Filter.NONE, folds = setOf("fix"))
        assertEquals(
            listOf("# Feature", "a,  a1,  a2", "sep", "b", "c", "# UI < Feature", "d", "# Fix"),
            shape(lines),
        )
    }

    @Test
    fun aFoldedParentHidesItsSubCategoriesAndCountsThem() {
        val lines = Board.lines(project, Filter.NONE, folds = setOf("feat"))
        assertEquals(listOf("# Feature", "# Fix", "e"), shape(lines))
        val header = lines[0] as Header
        assertTrue(header.folded)
        assertEquals(Status.IN_PROGRESS, header.status)
        assertEquals(3, header.done)
        assertEquals(6, header.total)
    }

    @Test
    fun aFoldedSubCategoryHidesItsRowsAndCountsNothing() {
        val lines = Board.lines(project, Filter.NONE, folds = setOf("ui"))
        val header = lines.filterIsInstance<Header>().first { it.category.id == "ui" }
        assertTrue(header.folded)
        assertEquals(Status.COMPLETED, header.status)
        assertEquals(0, header.total)
        assertEquals(listOf("# UI < Feature", "# Fix"), shape(lines).drop(5).take(2))
    }

    @Test
    fun aFoldedRowHidesItsSubtasksAndKeepsTheirCount() {
        val lines = Board.lines(project, Filter.NONE, folds = setOf("a"))
        val a = lines.filterIsInstance<Block>().first { it.row.task.id == "a" }
        assertTrue(a.children.isEmpty())
        assertTrue(a.row.folded)
        assertEquals(1, a.row.childDone)
        assertEquals(2, a.row.childTotal)
    }

    @Test
    fun aFoldOnARowWithNoSubtasksMarksNothing() {
        val b = Board.lines(project, Filter.NONE, folds = setOf("b")).filterIsInstance<Block>().first { it.row.task.id == "b" }
        assertFalse(b.row.folded)
    }

    @Test
    fun aFoldedRowStaysForASubtaskTheFilterKeeps() {
        val lines = Board.lines(project, Filter(priorities = setOf("high")), folds = setOf("a"))
        val a = lines.filterIsInstance<Block>().first { it.row.task.id == "a" }
        assertTrue("its parent shows while the filter keeps a subtask", a.row.folded)
        assertTrue(a.children.isEmpty())
    }

    @Test
    fun aFoldedCategoryShowsNoHiddenLine() {
        val lines = Board.lines(project, Filter(statuses = setOf("todo")), folds = setOf("feat", "fix"))
        assertEquals(listOf("# Feature", "# Fix"), shape(lines))
    }

    @Test
    fun aPeekDoesNotOpenAFoldedRow() {
        val lines = Board.lines(project, Filter(statuses = setOf("completed")), peek = "feat", folds = setOf("a"))
        val a = lines.filterIsInstance<Block>().first { it.row.task.id == "a" }
        assertTrue(a.children.isEmpty())
    }

    @Test
    fun searchLooksThroughFolds() {
        val lines = Board.search(Board.lines(project, Filter.NONE), "a1")
        assertEquals(listOf("# Feature", "a,  a1"), shape(lines))
    }

    @Test
    fun aFoldNamingNothingChangesNothing() {
        assertEquals(shape(Board.lines(project, Filter.NONE)), shape(Board.lines(project, Filter.NONE, folds = setOf("gone"))))
    }
}
