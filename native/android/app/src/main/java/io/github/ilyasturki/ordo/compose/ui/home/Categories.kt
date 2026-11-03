package io.github.ilyasturki.ordo.compose.ui.home

import android.os.SystemClock
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.model.Clash
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.LocalSnackbar
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.async
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonObject
import sh.calvin.reorderable.ReorderableColumn
import sh.calvin.reorderable.ReorderableItem
import sh.calvin.reorderable.rememberReorderableLazyListState

sealed interface CategorySheet {
    data class Menu(val cat: String) : CategorySheet
    data class New(val parent: String) : CategorySheet
    data class NewSeparator(val cat: String) : CategorySheet
    data class Delete(val cat: String) : CategorySheet
}

@Composable
fun CategorySheets(entry: Entry, filter: Filter, sheet: CategorySheet, onSheet: (CategorySheet?) -> Unit) {
    val engine = LocalEngine.current
    val scope = LocalAppScope.current
    val snackbar = LocalSnackbar.current
    val feedback = rememberFeedback()
    val pid = entry.id
    val dismiss = { onSheet(null) }
    var clash by remember { mutableStateOf<Pair<Clash, CompletableDeferred<ClashPick?>>?>(null) }
    // A taken name asks what frees it, and the sheet stays open until a pick lands or is cancelled.
    suspend fun named(name: String, edit: (String) -> JsonObject): Boolean {
        val a = engine.edit(pid, edit(name), quiet = true)
        val taken = a.clash ?: return a.applied
        val pick = CompletableDeferred<ClashPick?>()
        clash = taken to pick
        return when (pick.await()) {
            ClashPick.MINE -> engine.edit(pid, edit(taken.mine), quiet = true).applied
            ClashPick.OTHER -> engine.edit(pid, Ops.categoryRename(taken.taken, taken.other), quiet = true).applied &&
                engine.edit(pid, edit(name), quiet = true).applied
            null -> false
        }
    }
    when (sheet) {
        is CategorySheet.Menu -> {
            val cat = entry.project.category(sheet.cat) ?: return
            NameSheet(cat.name, "Category name", "Rename", onDismiss = dismiss, onDone = { name ->
                name == cat.name || named(name) { Ops.categoryRename(cat.id, it) }
            }) {
                SheetAction(Icons.NewFolder, "New sub-category") { onSheet(CategorySheet.New(cat.parentId.ifEmpty { cat.id })) }
                SheetAction(Icons.Separator, "Add a separator") {
                    if (filter.active) {
                        onSheet(null)
                        feedback.reject()
                        scope.launch { snackbar.showSnackbar("Clear the filter to add a separator") }
                    } else {
                        onSheet(CategorySheet.NewSeparator(cat.id))
                    }
                }
                SheetAction(Icons.Pin, if (cat.pinned) "Unpin" else "Pin") {
                    onSheet(null)
                    feedback.commit()
                    scope.launch { engine.edit(pid, Ops.categoryPin(cat.id, !cat.pinned)) }
                }
                SheetAction(Icons.Delete, "Delete category", danger = true) { onSheet(CategorySheet.Delete(cat.id)) }
            }
        }
        is CategorySheet.New -> NameSheet("", if (sheet.parent == "") "New category" else "New sub-category", "Add", onDismiss = dismiss, onDone = { name ->
            named(name) { Ops.categoryAdd(it, sheet.parent) }.also {
                if (it) feedback.commit()
                if (it && sheet.parent != "") scope.launch { engine.setFolded(pid, sheet.parent, false) }
            }
        })
        is CategorySheet.NewSeparator -> NameSheet("", "Separator label", "Add", blankAllowed = true, onDismiss = dismiss, onDone = { label ->
            engine.edit(pid, Ops.separator(sheet.cat, "", label), quiet = true).applied.also {
                if (it) scope.launch { engine.setFolded(pid, sheet.cat, false) }
            }
        })
        is CategorySheet.Delete -> {
            val cat = entry.project.category(sheet.cat) ?: return
            ConfirmDialog("Delete category “${cat.name}”?", deletedWith(entry.project, cat), "Delete", dismiss) {
                scope.launch { engine.edit(pid, Ops.categoryDelete(cat.id)) }
            }
        }
    }
    clash?.let { (taken, pick) ->
        ClashDialog(entry.project.category(taken.taken)?.name.orEmpty(), taken) {
            clash = null
            pick.complete(it)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CategoriesScreen(pid: String, back: () -> Unit) {
    val engine = LocalEngine.current
    val state by engine.state.collectAsStateWithLifecycle()
    val entry = state.project(pid)
    if (entry == null) {
        LaunchedEffect(Unit) { back() }
        return
    }
    var sheet by remember { mutableStateOf<CategorySheet?>(null) }

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = { IconButton(onClick = back) { Icon(Icons.Back, "Back") } },
                title = { Text("${entry.name} categories", maxLines = 1, overflow = TextOverflow.Ellipsis) },
            )
        },
        floatingActionButton = {
            FloatingActionButton(onClick = { sheet = CategorySheet.New("") }) { Icon(Icons.Add, "New category") }
        },
        snackbarHost = { SnackbarHost(LocalSnackbar.current) },
    ) { pad ->
        if (entry.project.categories.isEmpty()) {
            Text(
                "No categories yet",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth().padding(pad).padding(top = 64.dp),
            )
        } else {
            CategoryTree(entry, pad, onMenu = { sheet = CategorySheet.Menu(it) })
        }
    }

    sheet?.let { s ->
        val filter = (state.views[pid]?.filter ?: Filter.NONE).pruned(entry.project)
        CategorySheets(entry, filter, s) { sheet = it }
    }
}

// A top-level category is one list item with its sub-categories, so the block follows the finger whole.
@Composable
private fun CategoryTree(entry: Entry, pad: PaddingValues, onMenu: (String) -> Unit) {
    val engine = LocalEngine.current
    val appScope = LocalAppScope.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    val pid = entry.id
    val tops = remember(entry.project) { entry.project.categories.filter { it.parentId == "" } }
    var dragOrder by remember { mutableStateOf<List<Category>?>(null) }
    val shown = dragOrder ?: tops
    // A lifted row's release also reaches the row as a tap: that one opens no menu.
    var lifting by remember { mutableStateOf(false) }
    var droppedAt by remember { mutableLongStateOf(0L) }
    val tapping = { !lifting && SystemClock.uptimeMillis() - droppedAt > 400 }
    val lift = {
        lifting = true
        feedback.hold()
    }
    val letGo = {
        lifting = false
        droppedAt = SystemClock.uptimeMillis()
    }
    // The edit lands even if the page closes on the drop; only what follows it dies with the page.
    fun place(cat: String, before: String, done: (Boolean) -> Unit) {
        feedback.commit()
        val landing = appScope.async { engine.edit(pid, Ops.categoryPlace(cat, before)).applied }
        scope.launch { done(landing.await()) }
    }

    val listState = rememberLazyListState()
    val reorder = rememberReorderableLazyListState(listState) { from, to ->
        dragOrder = (dragOrder ?: tops).toMutableList().apply { add(to.index, removeAt(from.index)) }
        feedback.slot()
    }
    val ids = tops.map { it.id }
    var known by remember { mutableStateOf(ids.toSet()) }
    LaunchedEffect(ids) {
        if (ids.isNotEmpty() && ids.last() !in known) listState.animateScrollToItem(ids.lastIndex)
        known = ids.toSet()
    }

    val bottom = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding() + 96.dp
    LazyColumn(
        state = listState,
        modifier = Modifier.fillMaxSize().testTag("categories"),
        contentPadding = PaddingValues(top = pad.calculateTopPadding(), bottom = bottom),
    ) {
        items(shown, key = { it.id }) { top ->
            ReorderableItem(reorder, key = top.id) { dragging ->
                val stopped = {
                    letGo()
                    val order = dragOrder
                    if (order == null || order.map { it.id } == tops.map { it.id }) {
                        dragOrder = null
                    } else {
                        val at = order.indexOfFirst { it.id == top.id }
                        place(top.id, order.getOrNull(at + 1)?.id.orEmpty()) { dragOrder = null }
                    }
                }
                Lifted(dragging) {
                    Column(Modifier.fillMaxWidth()) {
                        CategoryLine(
                            top.name,
                            top.pinned,
                            child = false,
                            rowDrag = Modifier.longPressDraggableHandle(onDragStarted = { lift() }, onDragStopped = stopped),
                            handleDrag = Modifier.draggableHandle(onDragStarted = { lift() }, onDragStopped = stopped),
                            onClick = { if (tapping()) onMenu(top.id) },
                        )
                        SubCategories(
                            remember(entry.project, top.id) { entry.project.categories.filter { it.parentId == top.id } },
                            lift = lift,
                            letGo = letGo,
                            tapping = tapping,
                            place = ::place,
                            onMenu = onMenu,
                        )
                    }
                }
            }
        }
    }
}

// ReorderableColumn doesn't autoscroll: a sub-category's drag stays on the screen.
@Composable
private fun SubCategories(
    subs: List<Category>,
    lift: () -> Unit,
    letGo: () -> Unit,
    tapping: () -> Boolean,
    place: (String, String, (Boolean) -> Unit) -> Unit,
    onMenu: (String) -> Unit,
) {
    if (subs.isEmpty()) return
    val feedback = rememberFeedback()
    var order by remember(subs) { mutableStateOf(subs) }
    ReorderableColumn(
        list = order,
        onSettle = { from, to ->
            val next = order.toMutableList().apply { add(to, removeAt(from)) }
            order = next
            place(next[to].id, next.getOrNull(to + 1)?.id.orEmpty()) { applied -> if (!applied) order = subs }
        },
        onMove = { feedback.slot() },
    ) { _, sub, dragging ->
        key(sub.id) {
            ReorderableItem {
                Lifted(dragging) {
                    CategoryLine(
                        sub.name,
                        sub.pinned,
                        child = true,
                        rowDrag = Modifier.longPressDraggableHandle(onDragStarted = { lift() }, onDragStopped = { letGo() }),
                        handleDrag = Modifier.draggableHandle(onDragStarted = { lift() }, onDragStopped = { letGo() }),
                        onClick = { if (tapping()) onMenu(sub.id) },
                    )
                }
            }
        }
    }
}

@Composable
private fun Lifted(dragging: Boolean, content: @Composable () -> Unit) {
    val lift by animateDpAsState(if (dragging) 6.dp else 0.dp, label = "lift")
    Surface(
        tonalElevation = lift,
        shadowElevation = lift,
        color = if (dragging) MaterialTheme.colorScheme.surfaceContainerHigh else MaterialTheme.colorScheme.surface,
        content = content,
    )
}

@Composable
private fun CategoryLine(name: String, pinned: Boolean, child: Boolean, rowDrag: Modifier, handleDrag: Modifier, onClick: () -> Unit) {
    Row(
        rowDrag
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(start = if (child) 48.dp else 24.dp, end = 4.dp)
            .defaultMinSize(minHeight = 56.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            buildAnnotatedString {
                append(name)
                if (pinned) pinMark()
            },
            inlineContent = pinContent(MaterialTheme.colorScheme.onSurfaceVariant),
            style = if (child) MaterialTheme.typography.bodyLarge else MaterialTheme.typography.titleMedium,
            modifier = Modifier.weight(1f).padding(vertical = 12.dp),
        )
        Box(handleDrag.size(48.dp), contentAlignment = Alignment.Center) {
            Icon(Icons.DragHandle, "Move $name", tint = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}
