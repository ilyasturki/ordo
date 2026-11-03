package io.github.ilyasturki.ordo.compose.ui.settings

enum class ChangelogKind { Release, Section, Bullet, Paragraph }

data class ChangelogLine(val kind: ChangelogKind, val text: String, val date: String = "")

private val link = Regex("""\[([^\]]*)]\([^)]*\)""")
private val reference = Regex("""^\[[^\]]+]: """)

fun parseChangelog(src: String): List<ChangelogLine> {
    val out = mutableListOf<ChangelogLine>()
    var release: ChangelogLine? = null
    val body = mutableListOf<ChangelogLine>()
    fun flush() {
        if (release != null && body.isNotEmpty()) {
            out += release!!
            out += body
        }
        body.clear()
    }
    for (raw in src.lines()) {
        val line = raw.trim()
        if (line.startsWith("## ")) {
            flush()
            val (name, date) = line.removePrefix("## ").split(" - ", limit = 2).let { it[0] to it.getOrElse(1) { "" } }
            release = ChangelogLine(ChangelogKind.Release, name.trim('[', ']'), date)
        } else if (release == null || line.isEmpty() || reference.containsMatchIn(line)) {
            continue
        } else {
            val text = link.replace(line, "$1")
            body += when {
                text.startsWith("### ") -> ChangelogLine(ChangelogKind.Section, text.removePrefix("### "))
                text.startsWith("- ") -> ChangelogLine(ChangelogKind.Bullet, text.removePrefix("- "))
                else -> ChangelogLine(ChangelogKind.Paragraph, text)
            }
        }
    }
    flush()
    return out
}
