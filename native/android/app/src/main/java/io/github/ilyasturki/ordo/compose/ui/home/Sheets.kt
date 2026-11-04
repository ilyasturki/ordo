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
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.displayCutout
import androidx.compose.foundation.layout.exclude
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.systemBars
import androidx.compose.foundation.layout.union
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
import androidx.compose.material3.SheetState
import androidx.compose.material3.SheetValue
import androidx.compose.material3.Switch
import androidx.compose.material3.rememberBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
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
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
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
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.marks
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
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

// A height that grew with the keyboard would move the Expanded anchor on each keyboard frame, and each move restarts the sheet's spring, which then only runs once the keyboard has stopped.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun KeyboardSheet(sheet: SheetState, onDismiss: () -> Unit, content: @Composable ColumnScope.() -> Unit) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        modifier = Modifier.risesWithKeyboard(),
        sheetState = sheet,
        dragHandle = null,
        contentWindowInsets = { WindowInsets.systemBars.union(WindowInsets.displayCutout).only(WindowInsetsSides.Vertical) },
        content = content,
    )
}

// Composed so that the insets are the sheet's window's, the only one the keyboard animates. The sheet keeps its navigation bar padding, which the IME inset already counts.
private fun Modifier.risesWithKeyboard(): Modifier = composed {
    val lift = WindowInsets.ime.exclude(WindowInsets.navigationBars)
    offset { IntOffset(0, -lift.getBottom(this)) }
}

// A sheet around one name, docked to the keyboard.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NameSheet(
    initial: String,
    placeholder: String,
    action: String,
    onDismiss: () -> Unit,
    onDone: suspend (String) -> Boolean,
    blankAllowed: Boolean = false,
    extra: @Composable ColumnScope.() -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    val focus = remember { FocusRequester() }
    var text by remember { mutableStateOf(TextFieldValue(initial, TextRange(0, initial.length))) }
    val sheet = rememberBottomSheetState(SheetValue.Hidden, enabledValues = setOf(SheetValue.Hidden, SheetValue.Expanded))
    val done: () -> Unit = {
        scope.launch {
            if ((blankAllowed || text.text.isNotBlank()) && onDone(text.text.trim())) {
                sheet.hide()
                onDismiss()
            }
        }
    }
    KeyboardSheet(sheet, onDismiss) {
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
            extra()
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
) {
    DropdownMenu(expanded = expanded, onDismissRequest = onDismiss) {
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddSheet(entry: Entry, filter: Filter, initialCat: String, onDismiss: () -> Unit, onAdded: (String) -> Unit) {
    val engine = LocalEngine.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    val focus = remember { FocusRequester() }
    val sheet = rememberBottomSheetState(SheetValue.Hidden, enabledValues = setOf(SheetValue.Hidden, SheetValue.Expanded))
    val categories = entry.project.categories
    var title by remember { mutableStateOf("") }
    var details by remember { mutableStateOf<String?>(null) }
    var cat by remember { mutableStateOf(initialCat) }
    var priority by remember { mutableStateOf("") }
    var tag by remember { mutableStateOf("") }
    var estimate by remember { mutableIntStateOf(0) }
    var after by remember { mutableStateOf("") }
    val pid = entry.id
    val hidden = categories.firstOrNull { it.id == cat }?.let { !filter.taskVisible(Task(id = "", priority = priority, tagColor = tag), it) } == true

    val save: () -> Unit = save@{
        val text = title.replace(space, " ").trim()
        if (text.isEmpty() || cat.isEmpty()) return@save
        if (hidden) {
            feedback.reject()
            return@save
        }
        title = ""
        scope.launch {
            val a = engine.edit(pid, Ops.insert(cat, after, text), quiet = true)
            if (!a.applied) {
                title = text
                feedback.reject()
                return@launch
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
        }
    }

    KeyboardSheet(sheet, onDismiss) {
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
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Row(Modifier.weight(1f), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.weight(1f, fill = false).padding(end = 4.dp)) {
                MarkChip(categories.firstOrNull { it.id == cat }?.name ?: "Category", Modifier.padding(start = 8.dp), trailing = Icons.ExpandMore) { menu = "cat" }
                Picker(menu == "cat", { menu = "" }, categories, { c ->
                    categories.firstOrNull { it.id == c.parentId }?.let { "${it.name} › ${c.name}" } ?: c.name
                }, { onCategory(it.id) })
            }
            if (!describing) BarIcon(Icons.Notes, "Description", onDescription)
            Box {
                if (priority == "") {
                    BarIcon(Icons.Flag, "Priority") { menu = "priority" }
                } else {
                    PriorityChip(priority, compact = true, onClick = { menu = "priority" }, modifier = Modifier.padding(horizontal = 4.dp))
                }
                Picker(menu == "priority", { menu = "" }, Priority.all, Priority::label, onPriority, leading = { Icon(Icons.priority(it), null) })
            }
            Box {
                if (tag == "") {
                    BarIcon(Icons.Palette, "Colour") { menu = "tag" }
                } else {
                    MarkChip("", Modifier.padding(horizontal = 4.dp), tag = tag, name = "Colour", value = Tags.label(tag)) { menu = "tag" }
                }
                Picker(menu == "tag", { menu = "" }, listOf("") + Tags.all, Tags::label, onTag, leading = { if (it != "") TagDot(it) })
            }
            Box {
                if (estimate == 0) {
                    BarIcon(Icons.Schedule, "Estimate") { menu = "estimate" }
                } else {
                    MarkChip(Estimates.format(estimate), Modifier.padding(horizontal = 4.dp), name = "Estimate") { menu = "estimate" }
                }
                Picker(menu == "estimate", { menu = "" }, listOf(0) + Estimates.presets, { if (it == 0) "None" else Estimates.format(it) }, onEstimate)
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
