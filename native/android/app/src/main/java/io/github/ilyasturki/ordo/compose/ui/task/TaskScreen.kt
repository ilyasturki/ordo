package io.github.ilyasturki.ordo.compose.ui.task

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.isImeVisible
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AssistChip
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.InputChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation3.runtime.NavKey
import io.github.ilyasturki.ordo.compose.model.Description
import io.github.ilyasturki.ordo.compose.model.Estimates
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Priority
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.model.Tags
import io.github.ilyasturki.ordo.compose.model.Task
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.LocalSnackbar
import io.github.ilyasturki.ordo.compose.ui.TaskKey
import io.github.ilyasturki.ordo.compose.ui.home.Picker
import io.github.ilyasturki.ordo.compose.ui.home.StatusIcon
import io.github.ilyasturki.ordo.compose.ui.home.TagDot
import io.github.ilyasturki.ordo.compose.ui.home.plainFieldColors
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private val space = Regex("\\s+")

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun TaskScreen(pid: String, tid: String, open: (NavKey) -> Unit, back: () -> Unit) {
    val engine = LocalEngine.current
    val appScope = LocalAppScope.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    val state by engine.state.collectAsStateWithLifecycle()
    val project = state.project(pid)?.project
    val found = project?.find(tid)
    if (project == null || found == null) {
        LaunchedEffect(Unit) { back() }
        return
    }
    val (cat, task) = found

    var title by remember(tid) { mutableStateOf(task.title) }
    var titleBase by remember(tid) { mutableStateOf(task.title) }
    var editingTitle by remember { mutableStateOf(false) }
    val fields = Description.fields(task.description)
    var details by remember(tid) { mutableStateOf(fields.first) }
    var detailsBase by remember(tid) { mutableStateOf(fields.first) }
    var descBase by remember(tid) { mutableStateOf(task.description) }
    var editingDetails by remember { mutableStateOf(false) }
    var menu by remember { mutableStateOf("") }

    fun adoptTitle() {
        if (task.title != titleBase) {
            title = task.title
            titleBase = task.title
        }
    }
    fun adoptDetails() {
        if (task.description != descBase) {
            val text = Description.fields(task.description).first
            details = text
            detailsBase = text
            descBase = task.description
        }
    }
    LaunchedEffect(task.title) { if (!editingTitle) adoptTitle() }
    LaunchedEffect(task.description) { if (!editingDetails) adoptDetails() }

    suspend fun saveTitle() {
        val next = title.replace(space, " ").trim()
        if (next.isEmpty() || next == titleBase) return
        val base = titleBase
        titleBase = next
        val a = engine.edit(pid, Ops.title(tid, next, base), quiet = true)
        if (!a.applied && a.status != "unchanged") titleBase = base
    }

    suspend fun saveDetails() {
        if (details == detailsBase) return
        val current = engine.state.value.project(pid)?.project?.find(tid)?.second ?: return
        val text = Description.join(details, Description.fields(current.description).second)
        val base = descBase
        detailsBase = details
        descBase = text
        val a = engine.edit(pid, Ops.description(tid, text, base), quiet = true)
        if (!a.applied && a.status != "unchanged") descBase = base
    }

    LaunchedEffect(title) {
        if (title.replace(space, " ").trim() != titleBase) {
            delay(800)
            saveTitle()
        }
    }
    LaunchedEffect(details) {
        if (details != detailsBase) {
            delay(1000)
            saveDetails()
        }
    }
    val pendingTitle by rememberUpdatedState(suspend { saveTitle() })
    val pendingDetails by rememberUpdatedState(suspend { saveDetails() })
    DisposableEffect(tid) {
        onDispose {
            appScope.launch {
                pendingTitle()
                pendingDetails()
            }
        }
    }

    fun edit(op: kotlinx.serialization.json.JsonObject, quiet: Boolean = true) {
        scope.launch { engine.edit(pid, op, quiet) }
    }

    val parent = if (task.parentId != "") cat.tasks.firstOrNull { it.id == task.parentId } else null
    val children = if (task.parentId == "" && !task.separator) cat.tasks.filter { it.parentId == task.id } else emptyList()
    val done = task.finished

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = { IconButton(onClick = back) { Icon(Icons.Back, "Back") } },
                title = {
                    Box {
                        TextButton(onClick = { menu = "move" }) {
                            Text(cat.name, style = MaterialTheme.typography.titleMedium)
                            Icon(Icons.ExpandMore, null, Modifier.size(20.dp))
                        }
                        Picker(menu == "move", { menu = "" }, project.categories.filter { it.id != cat.id }, { c ->
                            project.categories.firstOrNull { it.id == c.parentId }?.let { "${it.name} › ${c.name}" } ?: c.name
                        }, { c -> edit(Ops.move(tid, c.id), quiet = false) })
                    }
                },
                actions = {
                    Box {
                        IconButton(onClick = { menu = "more" }) { Icon(Icons.More, "More") }
                        DropdownMenu(expanded = menu == "more", onDismissRequest = { menu = "" }) {
                            DropdownMenuItem(
                                text = { Text("Delete") },
                                leadingIcon = { Icon(Icons.Delete, null) },
                                onClick = {
                                    menu = ""
                                    scope.launch { engine.edit(pid, Ops.delete(tid)) }
                                },
                            )
                        }
                    }
                },
            )
        },
        floatingActionButton = {
            AnimatedVisibility(!task.separator && !WindowInsets.isImeVisible, enter = scaleIn() + fadeIn(), exit = scaleOut() + fadeOut()) {
                val label = if (done) "Mark open" else "Mark done"
                ExtendedFloatingActionButton(
                    text = { Text(label) },
                    icon = { Icon(if (done) Icons.Todo else Icons.Check, null) },
                    modifier = Modifier.semantics { contentDescription = label },
                    onClick = {
                        scope.launch {
                            val a = engine.edit(pid, Ops.status(tid, if (done) Status.TODO else Status.COMPLETED))
                            if (a.applied && !done) {
                                feedback.commit()
                                back()
                            }
                        }
                    },
                )
            }
        },
        snackbarHost = { SnackbarHost(LocalSnackbar.current) },
    ) { pad ->
        Column(
            Modifier
                .fillMaxSize()
                .padding(pad)
                .imePadding()
                .verticalScroll(rememberScrollState()),
        ) {
            if (parent != null) {
                Text(
                    "Subtask of ${parent.title}",
                    style = MaterialTheme.typography.labelLarge,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier
                        .clickable { open(TaskKey(pid, parent.id)) }
                        .padding(horizontal = 20.dp, vertical = 8.dp),
                )
            }
            TextField(
                title,
                { title = it.replace("\n", " ") },
                modifier = Modifier
                    .fillMaxWidth()
                    .onFocusChanged {
                        editingTitle = it.isFocused
                        if (it.isFocused) return@onFocusChanged
                        if (title.replace(space, " ").trim() == titleBase) adoptTitle() else scope.launch { saveTitle() }
                    },
                placeholder = { Text(if (task.separator) "Separator label" else "Title") },
                textStyle = MaterialTheme.typography.headlineSmall.copy(textDecoration = if (done) TextDecoration.LineThrough else null),
                colors = plainFieldColors(),
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Done),
            )
            if (!task.separator) {
                TextField(
                    details,
                    { details = it },
                    modifier = Modifier
                        .fillMaxWidth()
                        .onFocusChanged {
                            editingDetails = it.isFocused
                            if (it.isFocused) return@onFocusChanged
                            if (details == detailsBase) adoptDetails() else scope.launch { saveDetails() }
                        },
                    placeholder = { Text("Add details") },
                    textStyle = MaterialTheme.typography.bodyLarge,
                    colors = plainFieldColors(),
                    keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences),
                )
                Properties(task, Description.images(task.description), menu, { menu = it }, ::edit)
                if (task.parentId == "") {
                    Subtasks(pid, cat.id, task, children, open)
                }
            }
            Spacer(Modifier.height(120.dp))
        }
    }
}

// The fields that are set as chips, then one small button for each of the others.
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Properties(task: Task, images: Int, menu: String, setMenu: (String) -> Unit, edit: (kotlinx.serialization.json.JsonObject, Boolean) -> Unit) {
    val id = task.id
    val statusSet = task.status == Status.IN_PROGRESS || task.status == Status.CANCELLED
    @Composable
    fun Field(key: String, set: Boolean, icon: androidx.compose.ui.graphics.vector.ImageVector, add: String, label: String, leading: @Composable () -> Unit, picker: @Composable () -> Unit) {
        Box {
            if (set) {
                InputChip(selected = true, onClick = { setMenu(key) }, label = { Text(label) }, leadingIcon = leading)
            } else {
                IconButton(onClick = { setMenu(key) }) { Icon(icon, add, tint = MaterialTheme.colorScheme.onSurfaceVariant) }
            }
            picker()
        }
    }
    FlowRow(
        Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        Field("status", statusSet, Icons.status(task.status), "Status: ${Status.label(task.status)}", Status.label(task.status), { Icon(Icons.status(task.status), null, Modifier.size(18.dp)) }) {
            Picker(menu == "status", { setMenu("") }, Status.all, Status::label, { edit(Ops.status(id, it), false) }, leading = { Icon(Icons.status(it), null) })
        }
        Field("priority", task.priority != "", Icons.Flag, "Add a priority", Priority.label(task.priority), { Icon(Icons.priority(task.priority), null, Modifier.size(18.dp)) }) {
            Picker(menu == "priority", { setMenu("") }, Priority.all + "", Priority::label, { edit(Ops.priority(id, it), true) })
        }
        Field("tag", task.tagColor != "", Icons.Label, "Add a tag", Tags.label(task.tagColor), { TagDot(task.tagColor) }) {
            Picker(menu == "tag", { setMenu("") }, Tags.all + "", Tags::label, { edit(Ops.tag(id, it), true) }, leading = { if (it != "") TagDot(it) })
        }
        Field("estimate", task.estimateMinutes > 0, Icons.Schedule, "Add an estimate", Estimates.format(task.estimateMinutes), { Icon(Icons.Schedule, null, Modifier.size(18.dp)) }) {
            Picker(menu == "estimate", { setMenu("") }, Estimates.presets + 0, { if (it == 0) "None" else Estimates.format(it) }, { edit(Ops.estimate(id, it), true) })
        }
        if (images > 0) {
            AssistChip(onClick = {}, enabled = false, label = { Text(if (images == 1) "1 image" else "$images images") }, leadingIcon = { Icon(Icons.Image, null, Modifier.size(18.dp)) })
        }
    }
}

@Composable
private fun Subtasks(pid: String, cat: String, task: Task, children: List<Task>, open: (NavKey) -> Unit) {
    val engine = LocalEngine.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    var text by remember { mutableStateOf("") }
    Column(Modifier.padding(top = 8.dp)) {
        for (c in children) {
            Row(
                Modifier
                    .fillMaxWidth()
                    .clickable { open(TaskKey(pid, c.id)) }
                    .padding(start = 8.dp, end = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                StatusIcon(c.status, { scope.launch { engine.edit(pid, Ops.cycle(c.id)) } })
                Text(
                    c.title,
                    style = MaterialTheme.typography.bodyLarge,
                    color = if (c.finished) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface,
                    textDecoration = if (c.finished) TextDecoration.LineThrough else null,
                )
            }
        }
        Row(Modifier.padding(start = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(48.dp), contentAlignment = Alignment.Center) {
                Icon(Icons.Subtask, null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            TextField(
                text,
                { text = it.replace("\n", " ") },
                modifier = Modifier.fillMaxWidth(),
                placeholder = { Text("Add a subtask") },
                singleLine = true,
                colors = plainFieldColors(),
                keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.Sentences, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = {
                    val t = text.replace(space, " ").trim()
                    if (t.isNotEmpty()) {
                        text = ""
                        scope.launch {
                            if (engine.edit(pid, Ops.add(cat, t, parent = task.id), quiet = true).applied) feedback.commit()
                        }
                    }
                }),
            )
        }
    }
}
