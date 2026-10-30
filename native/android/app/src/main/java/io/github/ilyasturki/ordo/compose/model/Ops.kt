package io.github.ilyasturki.ordo.compose.model

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import java.security.SecureRandom

// What internal/core answers an edit; queue and undo stay JSON, sent back as the core wrote them.
@Serializable
data class Answer(
    val status: String,
    val msg: String = "",
    val row: String = "",
    val saved: Boolean = false,
    val theirs: String? = null,
    val queue: JsonObject? = null,
    val undo: List<JsonObject> = emptyList(),
) {
    val applied: Boolean get() = status == "applied"
}

// The ops of docs/sync.md the app makes.
object Ops {
    private fun op(action: String, vararg fields: Pair<String, Any?>): JsonObject = buildJsonObject {
        put("action", action)
        for ((k, v) in fields) {
            when (v) {
                null -> {}
                is String -> if (v != "") put(k, v)
                is Int -> put(k, v)
                is Boolean -> if (v) put(k, true)
                is List<*> -> put(k, JsonArray(v.map { JsonPrimitive(it as String) }))
                else -> error("unsupported field $k")
            }
        }
    }

    fun cycle(id: String) = op("status.cycle", "ids" to listOf(id))
    fun status(id: String, value: String) = op("status.set", "ids" to listOf(id), "value" to value)
    fun priority(id: String, value: String) = op("priority.set", "ids" to listOf(id), "value" to value)
    fun tag(id: String, value: String) = op("tag.set", "ids" to listOf(id), "value" to value)
    fun estimate(id: String, minutes: Int) = op("estimate.set", "ids" to listOf(id), "value" to minutes.toString())
    fun title(id: String, title: String, base: String) =
        op("task.title", "ids" to listOf(id), "title" to title, "base" to base)
    fun description(id: String, text: String, base: String) =
        op("task.description", "ids" to listOf(id), "text" to text, "base" to base)
    fun attach(id: String, lines: List<String>) = op("task.attach", "ids" to listOf(id), "lines" to lines)
    fun add(cat: String, title: String, parent: String = "") =
        op("task.add", "cat" to cat, "title" to title, "parent" to parent)
    fun insert(cat: String, after: String, title: String) =
        op("task.insert", "cat" to cat, "ids" to (if (after == "") emptyList() else listOf(after)), "title" to title)
    fun separator(cat: String, after: String, label: String) =
        op("separator.add", "cat" to cat, "ids" to (if (after == "") emptyList() else listOf(after)), "title" to label)
    fun delete(id: String) = op("task.delete", "ids" to listOf(id))
    fun move(id: String, cat: String) = op("task.move", "ids" to listOf(id), "cat" to cat)
    fun place(id: String, cat: String, before: String) =
        op("task.place", "ids" to listOf(id), "cat" to cat, "before" to before)
    fun categoryAdd(title: String, parent: String = "") =
        op("category.add", "title" to title, "parent" to parent)
    fun categoryRename(cat: String, title: String) = op("category.rename", "cat" to cat, "title" to title)
    fun categoryDelete(cat: String) = op("category.delete", "cat" to cat)
    fun projectCreate(title: String) = op("project.create", "title" to title)
    fun projectRename(title: String) = op("project.rename", "title" to title)
    fun projectDelete() = op("project.delete")

    private val random = SecureRandom()

    // The ID ordo mints: 16 random bytes as lowercase hex in 8-4-4-4-12 groups.
    fun newId(): String {
        val b = ByteArray(16).also(random::nextBytes)
        val h = b.joinToString("") { "%02x".format(it) }
        return "${h.substring(0, 8)}-${h.substring(8, 12)}-${h.substring(12, 16)}-${h.substring(16, 20)}-${h.substring(20)}"
    }
}
