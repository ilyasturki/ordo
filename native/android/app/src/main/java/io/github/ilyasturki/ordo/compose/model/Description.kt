package io.github.ilyasturki.ordo.compose.model

private val stored = Regex("""^attachments/([0-9a-f]{32}\.(?:png|jpg|gif|webp))$""")

data class ImageRef(val name: String, val target: String) {
    // The name the server stores it under; null for a path only the machine that wrote it can open.
    val file: String? get() = stored.matchEntire(target)?.groupValues?.get(1)
}

// The image lines of a description, read as app/src/lib/model.ts reads them.
object Description {
    private val ref = Regex("""!\[([^\[\]\n]*)]\(([^()\n]+)\)""")
    private val space = Regex("\\s+")

    private fun isImage(target: String): Boolean {
        val t = target.trim()
        return stored.matches(t) || t.startsWith("/")
    }

    fun refs(description: String): List<ImageRef> {
        if (!description.contains("![")) return emptyList()
        return ref.findAll(description)
            .map { ImageRef(it.groupValues[1].trim(), it.groupValues[2].trim()) }
            .filter { isImage(it.target) }
            .distinctBy { it.target }
            .toList()
    }

    fun images(description: String): Int = refs(description).size

    // As the TUI's search reads a description (attachments.Display): each reference turned into its label.
    fun display(description: String): String {
        if (!description.contains("![")) return description
        return ref.replace(description) { m ->
            if (isImage(m.groupValues[2])) "▣ " + m.groupValues[1].trim().ifEmpty { "image" } else m.value
        }
    }

    fun preview(description: String): String {
        val stripped = ref.replace(description) { m -> if (isImage(m.groupValues[2])) " " else m.value }
        return stripped.lineSequence()
            .map { it.split(space).filter(String::isNotEmpty).joinToString(" ") }
            .filter(String::isNotEmpty)
            .take(2)
            .joinToString("\n")
    }

    private fun imageOnly(line: String): Boolean {
        val m = ref.findAll(line).toList()
        return m.size == 1 && m[0].value == line && isImage(m[0].groupValues[2])
    }

    fun join(text: String, images: List<String>): String {
        val body = text.replace("\r\n", "\n").trimEnd('\n')
        val parts = if (body.isBlank()) emptyList() else listOf(body)
        return (parts + images).joinToString("\n")
    }

    // The details field and the image lines under it, or the whole description when splitting would lose something.
    fun fields(description: String): Pair<String, List<String>> {
        val lines = description.split("\n")
        var cut = lines.size
        while (cut > 0) {
            val line = lines[cut - 1].trim()
            if (line.isEmpty()) {
                cut--
                continue
            }
            if (!imageOnly(line)) break
            cut--
        }
        val images = lines.drop(cut).map(String::trim).filter(String::isNotEmpty)
        val text = lines.take(cut).joinToString("\n").trimEnd('\n')
        if (join(text, images) != description.trimEnd('\n')) return description to emptyList()
        return text to images
    }
}
