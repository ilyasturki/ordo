package io.github.ilyasturki.ordo.compose.model

import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.SetSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonTransformingSerializer
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive

// "none" on disk and on the wire, "" here, where it matches a task's empty field. Older prefs.json files hold "".
internal object Buckets : JsonTransformingSerializer<Set<String>>(SetSerializer(String.serializer())) {
    private fun swap(element: JsonElement, from: String, to: String) = JsonArray(
        element.jsonArray.map { if (it.jsonPrimitive.content == from) JsonPrimitive(to) else it },
    )

    override fun transformDeserialize(element: JsonElement) = swap(element, "none", "")

    override fun transformSerialize(element: JsonElement) = swap(element, "", "none")
}

// The TUI's filter (internal/app/state.go): "" is the no-priority and the untagged bucket, and a
// selected category also shows its sub-categories.
@Serializable
data class Filter(
    val statuses: Set<String> = emptySet(),
    @Serializable(with = Buckets::class) val priorities: Set<String> = emptySet(),
    val categories: Set<String> = emptySet(),
    @Serializable(with = Buckets::class) val tags: Set<String> = emptySet(),
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
