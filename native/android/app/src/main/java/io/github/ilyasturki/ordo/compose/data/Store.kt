package io.github.ilyasturki.ordo.compose.data

import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.json
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.JsonObject
import java.io.File
import java.io.FileOutputStream
import java.nio.file.Files
import java.nio.file.StandardCopyOption

@Serializable
data class Session(
    val server: String,
    val token: String,
    // Set once the server stopped taking the token; the data stays until the next sign-in.
    val expired: Boolean = false,
)

@Serializable
data class StoredProject(
    val id: String,
    val name: String,
    // The stamp of the server copy this one derives from, empty until the server saw it.
    val stamp: String,
    val error: String? = null,
)

@Serializable
data class Queued(
    val seq: Long,
    // Names the op in the sync answer.
    val qid: String,
    val project: String,
    val op: JsonObject,
)

@Serializable
data class Prefs(
    val current: String = "",
    val filters: Map<String, Filter> = emptyMap(),
)

// The phone's copy, as files: one per project (its meta line, then the core's JSON), the queue,
// the session and the prefs. Each write lands whole or not at all.
class Store(private val dir: File, imageCache: File = File(dir, "image-cache")) {
    private val projects = File(dir, "projects")
    val images = Images(File(dir, "uploads"), imageCache)

    init {
        projects.mkdirs()
    }

    fun session(): Session? = read("session.json")?.let { json.decodeFromString(Session.serializer(), it) }

    fun saveSession(s: Session?) {
        if (s == null) File(dir, "session.json").delete() else write(File(dir, "session.json"), json.encodeToString(Session.serializer(), s))
    }

    fun prefs(): Prefs = read("prefs.json")?.let { runCatching { json.decodeFromString(Prefs.serializer(), it) }.getOrNull() } ?: Prefs()

    fun savePrefs(p: Prefs) = write(File(dir, "prefs.json"), json.encodeToString(Prefs.serializer(), p))

    fun queue(): List<Queued> =
        read("queue.json")?.let { json.decodeFromString(ListSerializer(Queued.serializer()), it) } ?: emptyList()

    fun saveQueue(q: List<Queued>) = write(File(dir, "queue.json"), json.encodeToString(ListSerializer(Queued.serializer()), q))

    fun projects(): List<Pair<StoredProject, String>> =
        projects.listFiles { f -> f.name.endsWith(".json") }.orEmpty().mapNotNull { f ->
            runCatching {
                val text = f.readText()
                val cut = text.indexOf('\n')
                json.decodeFromString(StoredProject.serializer(), text.substring(0, cut)) to text.substring(cut + 1)
            }.getOrNull()
        }

    fun saveProject(meta: StoredProject, raw: String) =
        write(File(projects, "${meta.id}.json"), json.encodeToString(StoredProject.serializer(), meta) + "\n" + raw)

    fun deleteProject(id: String) {
        File(projects, "$id.json").delete()
    }

    fun wipe() {
        projects.listFiles().orEmpty().forEach(File::delete)
        File(dir, "queue.json").delete()
        File(dir, "prefs.json").delete()
        images.wipe()
    }

    private fun read(name: String): String? = File(dir, name).takeIf(File::exists)?.readText()

    private fun write(file: File, text: String) {
        val tmp = File(file.parentFile, "${file.name}.tmp")
        FileOutputStream(tmp).use { out ->
            out.write(text.toByteArray())
            out.fd.sync()
        }
        Files.move(tmp.toPath(), file.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING)
    }
}
