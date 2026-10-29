package io.github.ilyasturki.ordo.compose.model

// The image lines of a description, read as app/src/lib/model.ts reads them.
object Description {
    private val ref = Regex("""!\[([^\[\]\n]*)]\(([^()\n]+)\)""")
    private val space = Regex("\\s+")
    private val stored = Regex("""^attachments/[0-9a-f]{32}\.(?:png|jpg|gif|webp)$""")

    private fun isImage(target: String): Boolean {
        val t = target.trim()
        return stored.matches(t) || t.startsWith("/")
    }

    fun images(description: String): Int {
        if (!description.contains("![")) return 0
        return ref.findAll(description)
            .map { it.groupValues[2].trim() }
            .filter(::isImage)
            .distinct()
            .count()
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
