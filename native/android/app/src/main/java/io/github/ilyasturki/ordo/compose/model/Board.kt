package io.github.ilyasturki.ordo.compose.model

data class Row(
    val task: Task,
    val childDone: Int = 0,
    val childTotal: Int = 0,
    val images: Int = 0,
    val preview: String = "",
    val muted: Boolean = false,
)

sealed interface Line {
    val key: String
    val cat: String
}

data class Header(val category: Category, val parentName: String) : Line {
    override val key get() = "c:${category.id}"
    override val cat get() = category.id
}

// A top-level row with the subtasks the filter shows: the unit a drag moves.
data class Block(override val cat: String, val row: Row, val children: List<Row>) : Line {
    override val key get() = "t:${row.task.id}"
}

data class Hidden(override val cat: String, val count: Int, val peeking: Boolean) : Line {
    override val key get() = "h:$cat"
}

data class NoTasks(override val cat: String) : Line {
    override val key get() = "n:$cat"
}

// The list the TUI draws (internal/app/sync.go rebuildPositions): every category header, rows in
// file order with the finished ones in place, and a line counting what the filter hides.
object Board {
    fun lines(project: Project, filter: Filter, peek: String? = null, held: Set<String> = emptySet()): List<Line> {
        val names = project.categories.associate { it.id to it.name }
        val out = ArrayList<Line>()
        for (cat in project.categories) {
            out += Header(cat, names[cat.parentId].orEmpty())
            val first = out.size
            val shown = shownRows(cat, filter, held)
            val peeking = peek == cat.id
            var i = 0
            while (i < cat.tasks.size) {
                val end = blockEnd(cat.tasks, i)
                if (shown[i] || peeking) {
                    val children = (i + 1 until end)
                        .filter { shown[it] || peeking }
                        .map { row(cat.tasks[it], cat, filter, held, hidden = !shown[it]) }
                    val kids = cat.tasks.subList(i + 1, end)
                    out += Block(
                        cat.id,
                        row(cat.tasks[i], cat, filter, held, hidden = !shown[i]).copy(
                            childDone = kids.count { it.finished },
                            childTotal = kids.size,
                        ),
                        children,
                    )
                }
                i = end
            }
            val hidden = cat.tasks.indices.count { !shown[it] && !cat.tasks[it].separator }
            if (hidden > 0) out += Hidden(cat.id, hidden, peeking)
            if (out.size == first && project.categories.none { it.parentId == cat.id }) out += NoTasks(cat.id)
        }
        return out
    }

    // Muted: a row only a peek shows, or one held where an edit left it though the filter no longer matches.
    private fun row(task: Task, cat: Category, filter: Filter, held: Set<String>, hidden: Boolean) = Row(
        task = task,
        images = Description.images(task.description),
        preview = Description.preview(task.description),
        muted = !task.separator && (hidden || (task.id in held && !filter.taskVisible(task, cat))),
    )

    // The TUI's search (internal/app/search.go rowMatches) as a narrower list: a row stays when it or a subtask matches, every row under a matching category.
    fun search(lines: List<Line>, query: String): List<Line> {
        val q = query.trim()
        if (q == "") return lines
        val out = ArrayList<Line>()
        var header: Header? = null
        var named = false
        for (line in lines) {
            when (line) {
                is Header -> {
                    header = line
                    named = contains(line.category.name, q)
                    if (named) out += line
                }
                is Block -> {
                    val children = if (named) line.children else line.children.filter { matches(it.task, q) }
                    if (named || children.isNotEmpty() || matches(line.row.task, q)) {
                        if (!named) header?.let { out += it }
                        header = null
                        out += line.copy(children = children)
                    }
                }
                is Hidden -> {}
            }
        }
        return out
    }

    private fun matches(task: Task, query: String): Boolean =
        contains(task.title, query) || contains(Description.display(task.description), query)

    // The TUI's smartcase: an uppercase letter in the query makes it case-sensitive.
    private fun contains(text: String, query: String): Boolean =
        text.contains(query, ignoreCase = query.none(Char::isUpperCase))

    fun shownRows(cat: Category, filter: Filter, held: Set<String> = emptySet()): BooleanArray {
        val tasks = cat.tasks
        val shown = BooleanArray(tasks.size) { i ->
            val t = tasks[i]
            if (t.separator) !filter.active else t.id in held || filter.taskVisible(t, cat)
        }
        var i = 0
        while (i < tasks.size) {
            val end = blockEnd(tasks, i)
            for (j in i + 1 until end) shown[i] = shown[i] || shown[j]
            i = end
        }
        return shown
    }

    fun blockEnd(tasks: List<Task>, i: Int): Int {
        var end = i + 1
        if (tasks[i].parentId != "") return end
        while (end < tasks.size && tasks[end].parentId == tasks[i].id) end++
        return end
    }
}
