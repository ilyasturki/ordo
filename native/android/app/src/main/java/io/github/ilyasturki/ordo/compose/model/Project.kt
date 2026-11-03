package io.github.ilyasturki.ordo.compose.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonTransformingSerializer
import kotlinx.serialization.json.jsonPrimitive

// Read only: the core's JSON stays the copy that is stored and synced, so a field this app
// doesn't know survives.
val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
}

@Serializable
data class Task(
    val id: String,
    @SerialName("parent_id") val parentId: String = "",
    val title: String = "",
    val status: String = Status.TODO,
    @Serializable(with = Priority.Legacy::class) val priority: String = "",
    @SerialName("estimate_minutes") val estimateMinutes: Int = 0,
    val description: String = "",
    val kind: String = "",
    @SerialName("tag_color") val tagColor: String = "",
) {
    val separator: Boolean get() = kind == "separator"
    val finished: Boolean get() = status == Status.COMPLETED || status == Status.CANCELLED
}

@Serializable
data class Category(
    val id: String,
    val name: String = "",
    @SerialName("parent_id") val parentId: String = "",
    @SerialName("estimate_minutes") val estimateMinutes: Int = 0,
    val pinned: Boolean = false,
    val tasks: List<Task> = emptyList(),
)

@Serializable
data class Project(
    val id: String,
    val name: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
    val categories: List<Category> = emptyList(),
) {
    fun category(id: String): Category? = categories.firstOrNull { it.id == id }

    fun find(taskId: String): Pair<Category, Task>? {
        for (cat in categories) {
            val task = cat.tasks.firstOrNull { it.id == taskId } ?: continue
            return cat to task
        }
        return null
    }

    companion object {
        fun parse(raw: String): Project = json.decodeFromString(serializer(), raw)
    }
}

object Status {
    const val TODO = "todo"
    const val IN_PROGRESS = "in_progress"
    const val COMPLETED = "completed"
    const val CANCELLED = "cancelled"
    val all = listOf(TODO, IN_PROGRESS, COMPLETED, CANCELLED)

    fun label(status: String): String = when (status) {
        TODO -> "Todo"
        IN_PROGRESS -> "In Progress"
        COMPLETED -> "Completed"
        CANCELLED -> "Cancelled"
        else -> status
    }

    // What status.cycle steps to.
    fun next(status: String): String = when (status) {
        TODO -> IN_PROGRESS
        IN_PROGRESS -> COMPLETED
        COMPLETED -> CANCELLED
        else -> TODO
    }
}

// Medium is the missing field (docs/file-format.md); the cache an older build wrote can still hold "medium".
object Priority {
    val all = listOf("critical", "high", "", "low", "trivial")

    fun label(priority: String): String =
        if (priority == "" || priority == "medium") "Medium" else priority.replaceFirstChar { it.uppercase() }

    internal object Legacy : JsonTransformingSerializer<String>(String.serializer()) {
        override fun transformDeserialize(element: JsonElement) =
            if (element.jsonPrimitive.content == "medium") JsonPrimitive("") else element
    }
}

object Tags {
    val all = listOf("green", "blue", "magenta", "cyan")

    // A diamond mark from a file written before the four dots reads as its colour's dot.
    fun dot(color: String): String = color.removeSuffix("-diamond")

    fun label(color: String): String =
        if (color == "") "None" else dot(color).replaceFirstChar { it.uppercase() }
}

object Estimates {
    val presets = listOf(15, 30, 60, 120, 240, 480, 960, 1440, 2400)

    fun format(minutes: Int): String = when {
        minutes <= 0 -> ""
        minutes < 60 -> "${minutes}m"
        minutes / 60 < 8 -> "${minutes / 60}h"
        else -> "${minutes / 60 / 8}d"
    }
}
