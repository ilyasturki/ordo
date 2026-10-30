package io.github.ilyasturki.ordo.compose.data

import java.io.File
import java.io.FileOutputStream
import java.io.InputStream
import java.nio.file.Files
import java.nio.file.StandardCopyOption
import java.security.MessageDigest

class ImageRefused(message: String) : Exception(message)

// A file waiting under the name the server gives its bytes, and the line naming it in a description.
data class Kept(val name: String, val alt: String) {
    val line: String get() = "![$alt](attachments/$name)"
}

// Files under the name the server gives the bytes (docs/sync.md, Images): those it hasn't stored yet, kept like the queue, and a bounded cache of the others.
class Images(private val waiting: File, private val cache: File, private val bound: Long = 200L shl 20) {
    init {
        waiting.mkdirs()
        cache.mkdirs()
    }

    fun waiting(): List<String> =
        waiting.listFiles { f -> !f.name.endsWith(TMP) }.orEmpty().sortedBy(File::lastModified).map(File::getName)

    fun file(name: String): File? {
        File(waiting, name).takeIf(File::exists)?.let { return it }
        return File(cache, name).takeIf(File::exists)?.also { it.setLastModified(System.currentTimeMillis()) }
    }

    fun newTemp(): File = File.createTempFile("image", TMP, waiting)

    // Copies the stream to a temp file on disk for good: the caller keeps it, or deletes it.
    fun copy(input: InputStream): File {
        val temp = newTemp()
        try {
            FileOutputStream(temp).use { out ->
                val buffer = ByteArray(1 shl 16)
                var total = 0L
                while (true) {
                    val n = input.read(buffer)
                    if (n < 0) break
                    total += n
                    if (total > MAX_BYTES) throw ImageRefused(TOO_LARGE)
                    out.write(buffer, 0, n)
                }
                out.fd.sync()
            }
        } catch (e: Exception) {
            temp.delete()
            throw e
        }
        return temp
    }

    // Moves a temp file holding one of the formats the server stores into the waiting ones.
    fun keep(temp: File, alt: String): Kept {
        try {
            if (temp.length() > MAX_BYTES) throw ImageRefused(TOO_LARGE)
            val ext = extensionOf(temp) ?: throw ImageRefused(NOT_AN_IMAGE)
            val name = "${hash(temp)}.$ext"
            move(temp, File(waiting, name))
            return Kept(name, altOf(alt))
        } finally {
            temp.delete()
        }
    }

    // The server has it now: it stays readable from the cache, without a download.
    fun stored(name: String) {
        val file = File(waiting, name)
        if (!file.exists()) return
        move(file, File(cache, name))
        trim(name)
    }

    // Fetches the file into the cache, with fetch writing to the temp file it is given.
    suspend fun cached(name: String, fetch: suspend (File) -> Unit): File {
        val temp = File.createTempFile("image", TMP, cache)
        try {
            fetch(temp)
            val file = File(cache, name)
            move(temp, file)
            trim(name)
            return file
        } finally {
            temp.delete()
        }
    }

    fun wipe() {
        waiting.listFiles().orEmpty().forEach(File::delete)
        cache.listFiles().orEmpty().forEach(File::delete)
    }

    // Drops the temp files a crash left: they were never kept.
    fun sweep() {
        waiting.listFiles { f -> f.name.endsWith(TMP) }.orEmpty().forEach(File::delete)
        cache.listFiles { f -> f.name.endsWith(TMP) }.orEmpty().forEach(File::delete)
    }

    private fun trim(keep: String) {
        val files = cache.listFiles { f -> !f.name.endsWith(TMP) }.orEmpty().sortedBy(File::lastModified)
        var total = files.sumOf(File::length)
        for (f in files) {
            if (total <= bound) return
            if (f.name == keep) continue
            total -= f.length()
            f.delete()
        }
    }

    private fun move(from: File, to: File) {
        Files.move(from.toPath(), to.toPath(), StandardCopyOption.ATOMIC_MOVE, StandardCopyOption.REPLACE_EXISTING)
    }

    private fun hash(file: File): String {
        val digest = MessageDigest.getInstance("SHA-256")
        file.inputStream().use { input ->
            val buffer = ByteArray(1 shl 16)
            while (true) {
                val n = input.read(buffer)
                if (n < 0) break
                digest.update(buffer, 0, n)
            }
        }
        return digest.digest().take(16).joinToString("") { "%02x".format(it) }
    }

    companion object {
        const val MAX_BYTES = 25L shl 20
        const val TOO_LARGE = "Images over 25 MB can't be attached"
        const val NOT_AN_IMAGE = "Only png, jpeg, gif and webp images can be attached"
        private const val TMP = ".tmp"

        fun extensionOf(file: File): String? {
            val head = ByteArray(14)
            val n = file.inputStream().use { it.read(head) }
            return extension(head.copyOf(n.coerceAtLeast(0)))
        }

        // As the server sniffs them (http.DetectContentType): anything else, it refuses.
        fun extension(head: ByteArray): String? {
            fun starts(vararg b: Int, at: Int = 0) = head.size >= at + b.size && b.indices.all { head[at + it].toInt() and 0xff == b[it] }
            fun ascii(s: String, at: Int = 0) = starts(*s.map(Char::code).toIntArray(), at = at)
            return when {
                starts(0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a) -> "png"
                starts(0xff, 0xd8, 0xff) -> "jpg"
                ascii("GIF87a") || ascii("GIF89a") -> "gif"
                ascii("RIFF") && ascii("WEBPVP", at = 8) -> "webp"
                else -> null
            }
        }

        // The alt text of a file's line, as attachments.Line cleans it. The photo picker names a file by its number, which says nothing.
        fun altOf(fileName: String): String {
            val dot = fileName.lastIndexOf('.')
            val stem = if (dot > 0) fileName.substring(0, dot) else fileName
            val flat = stem.map { if (it == '[' || it == ']' || it.isISOControl()) ' ' else it }.joinToString("")
                .trim().replace(Regex("\\s+"), " ")
            return if (flat == "image" || flat.all(Char::isDigit)) "photo" else flat
        }
    }
}
