package io.github.ilyasturki.ordo.compose.sync

import java.text.Collator
import java.time.OffsetDateTime
import java.util.Locale

val projectsByName: Comparator<Entry> = Collator.getInstance().apply { strength = Collator.PRIMARY }
    .let { c -> Comparator<Entry> { a, b -> c.compare(a.name, b.name) }.thenBy { it.id } }

// As recent as the later of its last open here and its last edit anywhere: the TUI picker's rule.
fun recency(e: Entry, opened: Map<String, Long>): Long {
    val edited = runCatching { OffsetDateTime.parse(e.project.updatedAt).toInstant().toEpochMilli() }.getOrDefault(0)
    return maxOf(opened[e.id] ?: 0, edited)
}

// The TUI picker's grouping: groups match trimmed and case-insensitive, "" is no group.
fun groupKey(e: Entry): String = e.project.group.trim().lowercase(Locale.ROOT)

// Most recently used first, a tie by name, gathered into groups: each group where its most recent project
// stands, the projects without one last.
fun rankProjects(entries: List<Entry>, opened: Map<String, Long>): List<Entry> {
    val ranked = entries.sortedWith(compareByDescending<Entry> { recency(it, opened) }.then(projectsByName))
    val lead = HashMap<String, Int>()
    ranked.forEachIndexed { i, e -> lead.putIfAbsent(groupKey(e), i) }
    lead[""] = ranked.size
    return ranked.sortedBy { lead.getValue(groupKey(it)) }
}

// The groups in use, the most recent first, each spelt as its first project in entries has it.
fun projectGroups(entries: List<Entry>): List<String> =
    entries.filter { groupKey(it) != "" }.distinctBy(::groupKey).map { it.project.group.trim() }
