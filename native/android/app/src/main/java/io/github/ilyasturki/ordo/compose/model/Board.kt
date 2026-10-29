package io.github.ilyasturki.ordo.compose.model

data class Row(
    val task: Task,
    val childDone: Int = 0,
    val childTotal: Int = 0,
    val images: Int = 0,
    val preview: String = "",
    // Shown by a peek or held after an edit, though the filter hides it.
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

// The list the TUI draws (internal/app/sync.go rebuildPositions): every category header, rows in
// file order with the finished ones in place, and a line counting what the filter hides.
object Board {
    fun lines(project: Project, filter: Filter, peek: String? = null, held: Set<String> = emptySet()): List<Line> {
        val names = project.categories.associate { it.id to it.name }
        val out = ArrayList<Line>()
        for (cat in project.categories) {
            out += Header(cat, names[cat.parentId].orEmpty())
            val shown = shownRows(cat, filter, held)
            val peeking = peek == cat.id
            var i = 0
            while (i < cat.tasks.size) {
                val end = blockEnd(cat.tasks, i)
                if (shown[i] || peeking) {
                    val children = (i + 1 until end)
                        .filter { shown[it] || peeking }
                        .map { row(cat.tasks[it], cat, filter, peeking || cat.tasks[it].id in held) }
                    val kids = cat.tasks.subList(i + 1, end)
                    out += Block(
                        cat.id,
                        row(cat.tasks[i], cat, filter, peeking || cat.tasks[i].id in held).copy(
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
        }
        return out
    }

    private fun row(task: Task, cat: Category, filter: Filter, revealed: Boolean) = Row(
        task = task,
        images = Description.images(task.description),
        preview = Description.preview(task.description),
        muted = revealed && !task.separator && !filter.taskVisible(task, cat),
    )

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
