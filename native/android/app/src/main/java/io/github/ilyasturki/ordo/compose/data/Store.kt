package io.github.ilyasturki.ordo.compose.data

import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.View
import io.github.ilyasturki.ordo.compose.model.ViewEdit
import io.github.ilyasturki.ordo.compose.model.json
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.JsonObject
import java.io.File
import java.io.FileOutputStream
import java.io.IOException
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicReference

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
    // Where older builds kept each project's filter: boot moves them to the views.
    val filters: Map<String, Filter> = emptyMap(),
    // When each project was last the page shown or edited here, in epoch milliseconds.
    val opened: Map<String, Long> = emptyMap(),
)

@Serializable
data class Views(
    val held: Map<String, View> = emptyMap(),
    // Changes the server hasn't had yet.
    val unsent: Map<String, ViewEdit> = emptyMap(),
)

// The phone's copy, as files: one per project (its meta line, then the core's JSON), the queue,
// the session, the prefs and the views. Each write lands whole or not at all. A save returns once it
// is queued: the writes run in the order made, and a read waits for those before it.
class Store(
    private val dir: File,
    imageCache: File = File(dir, "image-cache"),
    private val writer: ExecutorService = Disk,
) {
    private val projects = File(dir, "projects")
    val images = Images(File(dir, "uploads"), imageCache)
    private val failure = AtomicReference<Throwable?>()

    init {
        projects.mkdirs()
    }

    fun session(): Session? = read("session.json")?.let { json.decodeFromString(Session.serializer(), it) }

    fun saveSession(s: Session?) {
        if (s == null) queue { File(dir, "session.json").delete() } else write(File(dir, "session.json")) { json.encodeToString(Session.serializer(), s) }
    }

    fun prefs(): Prefs = read("prefs.json")?.let { runCatching { json.decodeFromString(Prefs.serializer(), it) }.getOrNull() } ?: Prefs()

    // No fsync: prefs() starts from the defaults when a crash leaves the file torn.
    fun savePrefs(p: Prefs) = write(File(dir, "prefs.json"), durable = false) { json.encodeToString(Prefs.serializer(), p) }

    fun views(): Views = read("views.json")?.let { runCatching { json.decodeFromString(Views.serializer(), it) }.getOrNull() } ?: Views()

    fun saveViews(v: Views) = write(File(dir, "views.json")) { json.encodeToString(Views.serializer(), v) }

    fun queue(): List<Queued> =
        read("queue.json")?.let { json.decodeFromString(ListSerializer(Queued.serializer()), it) } ?: emptyList()

    fun saveQueue(q: List<Queued>) = write(File(dir, "queue.json")) { json.encodeToString(ListSerializer(Queued.serializer()), q) }

    fun projects(): List<Pair<StoredProject, String>> {
        settle()
        return projects.listFiles { f -> f.name.endsWith(".json") }.orEmpty().mapNotNull { f ->
            runCatching {
                val text = f.readText()
                val cut = text.indexOf('\n')
                json.decodeFromString(StoredProject.serializer(), text.substring(0, cut)) to text.substring(cut + 1)
            }.getOrNull()
        }
    }

    fun saveProject(meta: StoredProject, raw: String) =
        write(File(projects, "${meta.id}.json")) { json.encodeToString(StoredProject.serializer(), meta) + "\n" + raw }

    fun deleteProject(id: String) = queue { File(projects, "$id.json").delete() }

    fun wipe() {
        queue {
            projects.listFiles().orEmpty().forEach(File::delete)
            File(dir, "queue.json").delete()
            File(dir, "prefs.json").delete()
            File(dir, "views.json").delete()
        }
        images.wipe()
    }

    private fun read(name: String): String? {
        settle()
        return File(dir, name).takeIf(File::exists)?.readText()
    }

    private fun write(file: File, durable: Boolean = true, text: () -> String) = queue {
        val tmp = File(file.parentFile, "${file.name}.tmp")
        FileOutputStream(tmp).use { out ->
            out.write(text().toByteArray())
            if (durable) out.fd.sync()
        }
        Files.move(tmp.toPath(), file.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING)
    }

    // A write that failed surfaces at the next save or read, never silently.
    private fun queue(job: () -> Unit) {
        val earlier = failure.getAndSet(null)
        writer.execute {
            try {
                job()
            } catch (t: Throwable) {
                failure.compareAndSet(null, t)
            }
        }
        earlier?.let { throw IOException("The phone's copy didn't save", it) }
    }

    private fun settle() {
        writer.submit {}.get()
        raise()
    }

    private fun raise() {
        failure.getAndSet(null)?.let { throw IOException("The phone's copy didn't save", it) }
    }

    companion object {
        // One lane for every store: a read through any of them waits for the writes queued before it.
        private val Disk: ExecutorService = Executors.newSingleThreadExecutor { r -> Thread(r, "ordo-store").apply { isDaemon = true } }
    }
}
