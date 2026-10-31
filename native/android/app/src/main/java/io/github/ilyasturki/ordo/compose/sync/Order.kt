package io.github.ilyasturki.ordo.compose.sync

import java.text.Collator
import java.time.OffsetDateTime

val projectsByName: Comparator<Entry> = Collator.getInstance().apply { strength = Collator.PRIMARY }
    .let { c -> Comparator<Entry> { a, b -> c.compare(a.name, b.name) }.thenBy { it.id } }

// As recent as the later of its last open here and its last edit anywhere: the TUI picker's rule.
fun recency(e: Entry, opened: Map<String, Long>): Long {
    val edited = runCatching { OffsetDateTime.parse(e.project.updatedAt).toInstant().toEpochMilli() }.getOrDefault(0)
    return maxOf(opened[e.id] ?: 0, edited)
}

// Most recently used first, a tie by name.
fun rankProjects(entries: List<Entry>, opened: Map<String, Long>): List<Entry> =
    entries.sortedWith(compareByDescending<Entry> { recency(it, opened) }.then(projectsByName))
