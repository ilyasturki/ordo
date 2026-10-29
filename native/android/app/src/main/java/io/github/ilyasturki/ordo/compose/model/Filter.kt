package io.github.ilyasturki.ordo.compose.model

import kotlinx.serialization.Serializable

// The TUI's filter (internal/app/state.go): "" is the no-priority and the untagged bucket, and a
// selected category also shows its sub-categories.
@Serializable
data class Filter(
    val statuses: Set<String> = emptySet(),
    val priorities: Set<String> = emptySet(),
    val categories: Set<String> = emptySet(),
    val tags: Set<String> = emptySet(),
) {
    val active: Boolean
        get() = statuses.isNotEmpty() || priorities.isNotEmpty() || categories.isNotEmpty() || tags.isNotEmpty()

    fun categoryVisible(cat: Category): Boolean =
        categories.isEmpty() || cat.id in categories || (cat.parentId != "" && cat.parentId in categories)

    fun taskVisible(task: Task, cat: Category): Boolean {
        if (!active) return true
        if (statuses.isNotEmpty() && task.status !in statuses) return false
        if (priorities.isNotEmpty() && task.priority !in priorities) return false
        if (!categoryVisible(cat)) return false
        if (tags.isNotEmpty() && Tags.dot(task.tagColor) !in tags) return false
        return true
    }

    fun pruned(project: Project): Filter {
        val ids = project.categories.mapTo(HashSet()) { it.id }
        return if (categories.all { it in ids }) this else copy(categories = categories.filterTo(HashSet()) { it in ids })
    }

    companion object {
        val NONE = Filter()
    }
}
