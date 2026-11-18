package io.github.ilyasturki.ordo.compose.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.Switch
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.PopupProperties
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.model.Clash
import io.github.ilyasturki.ordo.compose.model.Estimates
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Priority
import io.github.ilyasturki.ordo.compose.model.Project
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.model.Tags
import io.github.ilyasturki.ordo.compose.model.Task
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.sync.SyncState
import io.github.ilyasturki.ordo.compose.sync.Trouble
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.marks
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

private val space = Regex("\\s+")

@Composable
fun plainFieldColors() = TextFieldDefaults.colors(
    focusedContainerColor = Color.Transparent,
    unfocusedContainerColor = Color.Transparent,
    disabledContainerColor = Color.Transparent,
    focusedIndicatorColor = Color.Transparent,
    unfocusedIndicatorColor = Color.Transparent,
)

// A sheet around one name, docked to the keyboard. onDone's quiet is false for a save the user didn't ask for by name (a dismissal, another action), which then offers its Undo.
// A dismissal saves a changed name; an emptied one keeps the old, and a new row left empty adds nothing.
@Composable
fun NameSheet(
    initial: String,
    placeholder: String,
    action: String,
    onDismiss: () -> Unit,
    onDone: suspend (name: String, quiet: Boolean) -> Boolean,
    blankAllowed: Boolean = false,
    extra: @Composable ColumnScope.(leave: (then: () -> Unit) -> Unit) -> Unit = {},
) {
    val appScope = LocalAppScope.current
    val focus = remember { FocusRequester() }
    var text by rememberSaveable(stateSaver = TextFieldValue.Saver) { mutableStateOf(TextFieldValue(initial, TextRange(0, initial.length))) }
    suspend fun saveOnLeave(): Boolean {
        val name = text.text.trim()
        if (name == initial.trim() || (name.isEmpty() && (!blankAllowed || initial.isBlank()))) return true
        return onDone(name, false)
    }
    val leave: (() -> Unit) -> Unit = { then -> appScope.launch(Dispatchers.Main) { if (saveOnLeave()) then() } }
    KeyboardSheet(onLeave = ::saveOnLeave, onGone = onDismiss) { finish ->
        val done = { finish { (blankAllowed || text.text.isNotBlank()) && onDone(text.text.trim(), true) } }
        Column(Modifier.padding(top = 12.dp, bottom = 8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                TextField(
                    text,
                    { text = it },
                    modifier = Modifier.weight(1f).focusRequester(focus),
                    placeholder = { Text(placeholder) },
                    singleLine = true,
                    textStyle = MaterialTheme.typography.titleMedium,
                    colors = plainFieldColors(),
                    keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { done() }),
                )
                TextButton(onClick = done, enabled = blankAllowed || text.text.isNotBlank(), modifier = Modifier.padding(end = 8.dp)) { Text(action) }
            }
            extra(leave)
        }
    }
    LaunchedEffect(Unit) { focus.requestFocus() }
}

@Composable
fun SheetAction(icon: ImageVector, label: String, danger: Boolean = false, onClick: () -> Unit) {
    val colors = MaterialTheme.colorScheme
    ListItem(
        headlineContent = { Text(label) },
        modifier = Modifier.clickable(onClick = onClick, role = Role.Button),
        leadingContent = { Icon(icon, null) },
        colors = ListItemDefaults.colors(
            containerColor = Color.Transparent,
            headlineColor = if (danger) colors.error else colors.onSurface,
            leadingIconColor = if (danger) colors.error else colors.onSurfaceVariant,
        ),
    )
}

@Composable
fun ConfirmDialog(title: String, text: String, action: String, onDismiss: () -> Unit, onConfirm: () -> Unit) {
    val feedback = rememberFeedback()
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = { Text(text) },
        confirmButton = {
            TextButton(onClick = {
                feedback.commit()
                onDismiss()
                onConfirm()
            }) { Text(action, color = MaterialTheme.colorScheme.error) }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

enum class ClashPick { MINE, OTHER }

// The TUI's chooser for a taken category name (internal/app/category_clash.go), merge aside.
@Composable
fun ClashDialog(takenName: String, clash: Clash, onPick: (ClashPick?) -> Unit) {
    AlertDialog(
        onDismissRequest = { onPick(null) },
        title = { Text("“$takenName” is taken") },
        text = { Text("Another category here has that name.") },
        confirmButton = {
            Column(horizontalAlignment = Alignment.End) {
                TextButton(onClick = { onPick(ClashPick.MINE) }) { Text("Name it “${clash.mine}”") }
                TextButton(onClick = { onPick(ClashPick.OTHER) }) { Text("Rename the other to “${clash.other}”") }
                TextButton(onClick = { onPick(null) }) { Text("Cancel") }
            }
        },
    )
}

fun countTasks(project: Project, cat: Category): Int {
    val ids = project.categories.filter { it.id == cat.id || it.parentId == cat.id }
    return ids.sumOf { c -> c.tasks.count { !it.line } }
}

// What deleting a category takes with it, in the TUI's words (internal/app/view_confirm.go).
fun deletedWith(project: Project, cat: Category): String {
    val subs = project.categories.count { it.parentId == cat.id }
    val tasks = countTasks(project, cat)
    val taskWord = if (tasks == 1) "1 task" else "$tasks tasks"
    return when (subs) {
        0 -> "Its $taskWord go too."
        1 -> "Its sub-category and $taskWord go too."
        else -> "Its $subs sub-categories and $taskWord go too."
    }
}

// Picks one value from a short list, anchored to what opened it.
@Composable
fun <T> Picker(
    expanded: Boolean,
    onDismiss: () -> Unit,
    options: List<T>,
    label: (T) -> String,
    onPick: (T) -> Unit,
    leading: (@Composable (T) -> Unit)? = null,
    properties: PopupProperties = PopupProperties(focusable = true),
) {
    DropdownMenu(expanded = expanded, onDismissRequest = onDismiss, properties = properties) {
        for (o in options) {
            DropdownMenuItem(
                text = { Text(label(o)) },
                leadingIcon = leading?.let { { it(o) } },
                onClick = {
                    onDismiss()
                    onPick(o)
                },
            )
        }
    }
}

// Save keeps the sheet open for the next task; a dismissal saves what is typed, with its Undo, and a refused save keeps the sheet up once.
@Composable
fun AddSheet(entry: Entry, filter: Filter, initialCat: String, onDismiss: () -> Unit, onAdded: (String) -> Unit) {
    val engine = LocalEngine.current
    val appScope = LocalAppScope.current
    val feedback = rememberFeedback()
    val focus = remember { FocusRequester() }
    val categories = entry.project.categories
    var title by rememberSaveable { mutableStateOf("") }
    var details by rememberSaveable { mutableStateOf<String?>(null) }
    var cat by rememberSaveable { mutableStateOf(initialCat) }
    var priority by rememberSaveable { mutableStateOf("") }
    var tag by rememberSaveable { mutableStateOf("") }
    var estimate by rememberSaveable { mutableIntStateOf(0) }
    var after by rememberSaveable { mutableStateOf("") }
    val pid = entry.id
    val hidden = categories.firstOrNull { it.id == cat }?.let { !filter.taskVisible(Task(id = "", priority = priority, tagColor = tag), it) } == true

    // Runs on the app's scope: the edits after the insert land even when the sheet has gone.
    suspend fun commit(quiet: Boolean): Boolean {
        val text = title.replace(space, " ").trim()
        if (text.isEmpty() || cat.isEmpty()) {
            if (details.isNullOrBlank()) return true
            feedback.reject()
            return false
        }
        if (hidden) {
            feedback.reject()
            return false
        }
        title = ""
        val a = engine.edit(pid, Ops.insert(cat, after, text), quiet = quiet)
        if (!a.applied) {
            title = text
            feedback.reject()
            return false
        }
        feedback.commit()
        val row = a.row
        after = row
        details?.takeIf(String::isNotBlank)?.let { engine.edit(pid, Ops.description(row, it.trim(), ""), quiet = true) }
        if (priority != "") engine.edit(pid, Ops.priority(row, priority), quiet = true)
        if (tag != "") engine.edit(pid, Ops.tag(row, tag), quiet = true)
        if (estimate > 0) engine.edit(pid, Ops.estimate(row, estimate), quiet = true)
        details = null
        onAdded(cat)
        return true
    }
    val save: () -> Unit = { appScope.launch(Dispatchers.Main) { commit(quiet = true) } }

    KeyboardSheet(onLeave = { commit(quiet = false) }, onGone = onDismiss) {
        Column(Modifier.padding(top = 8.dp, bottom = 4.dp)) {
            TextField(
                title,
                { title = it.replace("\n", " ") },
                modifier = Modifier.fillMaxWidth().focusRequester(focus),
                placeholder = { Text("New task") },
                singleLine = true,
                textStyle = MaterialTheme.typography.titleMedium,
                colors = plainFieldColors(),
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { save() }),
            )
            if (hidden) {
                Text(
                    "Clear the filter to add a task here",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(horizontal = 16.dp),
                )
            }
            details?.let { d ->
                TextField(
                    d,
                    { details = it },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text("Add a description") },
                    textStyle = MaterialTheme.typography.bodyMedium,
                    colors = plainFieldColors(),
                    maxLines = 5,
                    keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                )
            }
            AddBar(
                categories, cat, details != null, priority, tag, estimate,
                canSave = title.isNotBlank() && !hidden,
                onCategory = { c ->
                    if (c != cat) after = ""
                    cat = c
                },
                onDescription = { details = "" },
                onPriority = { priority = it },
                onTag = { tag = it },
                onEstimate = { estimate = it },
                onSave = save,
            )
        }
    }
    LaunchedEffect(Unit) { focus.requestFocus() }
}

// One line at 360dp whatever is set: an unset field is its icon, a set one its value in a compact chip (the priority's arrow, the colour's dot, the estimate).
@Composable
fun AddBar(
    categories: List<Category>,
    cat: String,
    describing: Boolean,
    priority: String,
    tag: String,
    estimate: Int,
    canSave: Boolean,
    onCategory: (String) -> Unit,
    onDescription: () -> Unit,
    onPriority: (String) -> Unit,
    onTag: (String) -> Unit,
    onEstimate: (Int) -> Unit,
    onSave: () -> Unit,
) {
    var menu by remember { mutableStateOf("") }
    // A focusable menu would take the window's focus, and the keyboard with it.
    val keepsKeyboard = PopupProperties(focusable = false)
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.weight(1f, fill = false).padding(end = 4.dp)) {
                MarkChip(categories.firstOrNull { it.id == cat }?.name ?: "Category", Modifier.padding(start = 8.dp), trailing = Icons.ExpandMore) { menu = "cat" }
                Picker(menu == "cat", { menu = "" }, categories, { c ->
                    categories.firstOrNull { it.id == c.parentId }?.let { "${it.name} › ${c.name}" } ?: c.name
                }, { onCategory(it.id) }, properties = keepsKeyboard)
            }
            if (!describing) BarIcon(Icons.Notes, "Description", onDescription)
            Box {
                if (priority == "") {
                    BarIcon(Icons.Flag, "Priority") { menu = "priority" }
                } else {
                    PriorityChip(priority, compact = true, onClick = { menu = "priority" }, modifier = Modifier.padding(horizontal = 4.dp))
                }
                Picker(menu == "priority", { menu = "" }, Priority.all, Priority::label, onPriority, leading = { Icon(Icons.priority(it), null) }, properties = keepsKeyboard)
            }
            Box {
                if (tag == "") {
                    BarIcon(Icons.Palette, "Colour") { menu = "tag" }
                } else {
                    MarkChip("", Modifier.padding(horizontal = 4.dp), tag = tag, name = "Colour", value = Tags.label(tag)) { menu = "tag" }
                }
                Picker(menu == "tag", { menu = "" }, listOf("") + Tags.all, Tags::label, onTag, leading = { if (it != "") TagDot(it) }, properties = keepsKeyboard)
            }
            Box {
                if (estimate == 0) {
                    BarIcon(Icons.Schedule, "Estimate") { menu = "estimate" }
                } else {
                    MarkChip(Estimates.format(estimate), Modifier.padding(horizontal = 4.dp), name = "Estimate") { menu = "estimate" }
                }
                Picker(menu == "estimate", { menu = "" }, listOf(0) + Estimates.presets, { if (it == 0) "None" else Estimates.format(it) }, onEstimate, properties = keepsKeyboard)
            }
        }
        TextButton(onClick = onSave, enabled = canSave) { Text("Save") }
    }
}

@Composable
private fun BarIcon(icon: ImageVector, name: String, onClick: () -> Unit) {
    IconButton(onClick = onClick) { Icon(icon, name, tint = MaterialTheme.colorScheme.onSurfaceVariant) }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun FilterSheet(project: Project, filter: Filter, onChange: (Filter) -> Unit, onDismiss: () -> Unit) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.verticalScroll(rememberScrollState()).padding(horizontal = 20.dp).padding(bottom = 16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Filter", style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
                Switch(
                    checked = filter.active,
                    onCheckedChange = { onChange(filter.copy(off = !it)) },
                    enabled = filter.hasValues,
                    modifier = Modifier.semantics { contentDescription = "Filter on" },
                )
                TextButton(onClick = { onChange(Filter.NONE) }, enabled = filter.hasValues) { Text("Clear") }
            }
            Section("Status") {
                for (s in Status.all) {
                    Toggle(Status.label(s), s in filter.statuses, leading = { Icon(Icons.status(s), null, Modifier.size(18.dp)) }) { onChange(filter.copy(off = false, statuses = filter.statuses.flip(s))) }
                }
            }
            Section("Priority") {
                for (p in Priority.all) {
                    Toggle(Priority.label(p), p in filter.priorities, leading = { PriorityMark(p) }) { onChange(filter.copy(off = false, priorities = filter.priorities.flip(p))) }
                }
            }
            if (project.categories.isNotEmpty()) {
                Section("Category") {
                    for (c in project.categories) {
                        val parent = project.categories.firstOrNull { it.id == c.parentId }
                        Toggle(parent?.let { "${it.name} › ${c.name}" } ?: c.name, c.id in filter.categories) {
                            onChange(filter.copy(off = false, categories = filter.categories.flip(c.id)))
                        }
                    }
                }
            }
            Section("Tag") {
                for (t in Tags.all + "") {
                    Toggle(if (t == "") "Untagged" else Tags.label(t), t in filter.tags, leading = if (t == "") null else ({ TagDot(t) })) {
                        onChange(filter.copy(off = false, tags = filter.tags.flip(t)))
                    }
                }
            }
        }
    }
}

private fun Set<String>.flip(v: String): Set<String> = if (v in this) this - v else this + v

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Text(title, style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 16.dp, bottom = 4.dp))
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) { content() }
}

@Composable
private fun Toggle(label: String, on: Boolean, leading: (@Composable () -> Unit)? = null, onClick: () -> Unit) {
    FilterChip(
        selected = on,
        onClick = onClick,
        label = { Text(label) },
        leadingIcon = leading ?: if (on) ({ Icon(Icons.Check, null, Modifier.size(18.dp)) }) else null,
    )
}

@Composable
private fun PriorityMark(priority: String) {
    val tint = when (priority) {
        "critical", "high" -> MaterialTheme.marks.up
        "low", "trivial" -> MaterialTheme.marks.down
        else -> LocalContentColor.current
    }
    Icon(Icons.priority(priority), null, Modifier.size(18.dp), tint = tint)
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SyncSheet(state: SyncState, onRetry: () -> Unit, onSignIn: () -> Unit, onDismiss: () -> Unit) {
    val trouble = state.trouble
    val (headline, body) = when (trouble) {
        is Trouble.Failing -> if (trouble.offline) {
            "Can't reach your server" to "The phone can't get to ordo serve right now: no signal, or the server is down. It keeps trying, in the background too."
        } else {
            "Your server isn't answering" to "ordo serve answered with an error: ${trouble.message}. The phone keeps trying, in the background too."
        }
        is Trouble.Refused -> "Your server refused the sync" to "ordo serve said: ${trouble.message}"
        is Trouble.Outdated -> "Update needed" to "This app and your server speak different versions of the sync: ${trouble.message}"
        Trouble.Expired -> "Sign in again" to "Your server no longer knows this phone: pair it again to keep syncing."
        null -> "Synced" to "Everything on this phone has reached your server."
    }
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.padding(horizontal = 24.dp).padding(bottom = 24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(if (trouble == null) Icons.Sync else Icons.CloudOff, null, tint = if (trouble == null) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error)
                Spacer(Modifier.width(12.dp))
                Text(headline, style = MaterialTheme.typography.titleLarge)
            }
            Text(body, style = MaterialTheme.typography.bodyMedium)
            val waiting = when {
                !state.waiting -> "No edit is waiting."
                state.pending + state.uploads + state.viewsWaiting == 1 -> "${state.waitingWords} waits safely on this phone."
                else -> "${state.waitingWords} wait safely on this phone."
            }
            Text(waiting, style = MaterialTheme.typography.bodyMedium)
            Text(
                "Last synced: ${if (state.lastSync == 0L) "not since the app started" else ago(state.lastSync)}\nServer: ${state.session?.server.orEmpty()}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(Modifier.fillMaxWidth().padding(top = 8.dp), horizontalArrangement = Arrangement.End) {
                if (trouble == Trouble.Expired) {
                    FilledTonalButton(onClick = onSignIn) { Text("Sign in again") }
                } else {
                    FilledTonalButton(onClick = onRetry) { Text("Retry") }
                }
            }
        }
    }
}

fun ago(at: Long): String {
    val s = (System.currentTimeMillis() - at) / 1000
    return when {
        s < 60 -> "just now"
        s < 3600 -> "${s / 60} min ago"
        s < 86_400 -> "${s / 3600} h ago"
        else -> "${s / 86_400} days ago"
    }
}
