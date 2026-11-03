package io.github.ilyasturki.ordo.compose.ui.home

import android.os.SystemClock
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.gestures.ScrollableState
import androidx.compose.foundation.gestures.detectDragGesturesAfterLongPress
import androidx.compose.foundation.interaction.DragInteraction
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.input.pointer.PointerInputChange
import androidx.compose.ui.input.pointer.PointerInputScope
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import io.github.ilyasturki.ordo.compose.model.Block
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Hidden
import io.github.ilyasturki.ordo.compose.model.Line
import io.github.ilyasturki.ordo.compose.model.NoTasks
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Seen
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.serialization.json.JsonObject
import sh.calvin.reorderable.DragGestureDetector
import sh.calvin.reorderable.ReorderableItem
import sh.calvin.reorderable.rememberReorderableLazyListState
import sh.calvin.reorderable.rememberScroller

// A peek reveals one category's hidden rows; held rows stay where an edit from the list left
// them though the filter no longer matches. Both end when the project or the filter changes.
data class ListUi(val peek: String? = null, val held: Set<String> = emptySet())

@Composable
fun ProjectList(
    entry: Entry,
    filter: Filter,
    folds: Set<String>,
    ui: ListUi,
    onUi: (ListUi) -> Unit,
    onOpen: (String) -> Unit,
    onHeader: (Category) -> Unit,
    onCategories: () -> Unit,
    onNewCategory: () -> Unit,
    onFocus: (String?) -> Unit,
    onScrolled: (Boolean) -> Unit = {},
    query: String = "",
    jump: String? = null,
    onJumped: () -> Unit = {},
) {
    val engine = LocalEngine.current
    val appScope = LocalAppScope.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    val pid = entry.id
    val searching = query.isNotBlank()
    val lines = remember(entry.project, filter, folds, ui, query) {
        if (searching) Board.search(Board.lines(entry.project, filter, held = ui.held), query) else Board.lines(entry.project, filter, ui.peek, ui.held, folds)
    }
    // Search looks through folds without changing them, so nothing there has a fold to toggle.
    val fold = { id: String, folded: Boolean -> scope.launch { engine.setFolded(pid, id, !folded) }; Unit }
    var dragOrder by remember { mutableStateOf<List<Line>?>(null) }
    // A drop's order stays on screen until its edit lands, and the next lift builds on it.
    var landing by remember { mutableStateOf<List<Line>?>(null) }
    val shown = dragOrder ?: landing ?: lines
    // A filter arriving from another device ends them too.
    LaunchedEffect(filter) { if (ui != ListUi()) onUi(ListUi()) }
    // A lifted row's release also reaches the row as a tap: that one neither opens it nor steps its status.
    var lifting by remember { mutableStateOf(false) }
    var droppedAt by remember { mutableLongStateOf(0L) }
    val tapping = { !lifting && SystemClock.uptimeMillis() - droppedAt > 400 }
    val listState = rememberLazyListState()
    val pull = remember(listState) { Pull(listState) }
    val navBar = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val reorder = rememberReorderableLazyListState(
        listState,
        scrollThresholdPadding = remember(navBar) { PaddingValues(bottom = navBar) },
        // The library's own pace: a twentieth of the list each tenth of a second.
        scroller = rememberScroller(pull, { listState.layoutInfo.viewportSize.height * 0.05f }),
    ) { from, to ->
        dragOrder = moved(dragOrder ?: landing ?: lines, from.key.toString(), to.key.toString()) ?: return@rememberReorderableLazyListState
        feedback.slot()
    }
    val onScreen by rememberUpdatedState(shown)
    var lifted by remember { mutableStateOf(emptyList<Line>()) }
    val drop: (Block) -> Unit = drop@{ block ->
        lifting = false
        droppedAt = SystemClock.uptimeMillis()
        val order = dragOrder
        dragOrder = null
        val op = order?.let { dropOp(it, block, lifted) } ?: return@drop
        val into = headerAbove(order, block)
        if (filter.active) onUi(ui.copy(held = ui.held + block.row.task.id))
        landing = order
        // The edit lands even if the page goes on the drop; only what follows it dies with the page.
        val answer = appScope.async { engine.edit(pid, op, quiet = true) }
        scope.launch {
            val a = answer.await()
            if (a.applied) feedback.commit() else if (a.status != "unchanged") feedback.reject()
            // As adding to it does: the row stays in sight where it landed.
            if (a.applied && into?.folded == true) engine.setFolded(pid, into.category.id, false)
            if (landing === order) landing = null
        }
    }
    // A handle's gesture keeps the callbacks it started with: these reach the current ones.
    val lift by rememberUpdatedState {
        lifted = onScreen
        lifting = true
        feedback.hold()
    }
    val dropped by rememberUpdatedState(drop)
    val report by rememberUpdatedState(onFocus)
    val scrolledReport by rememberUpdatedState(onScrolled)
    LaunchedEffect(listState) { snapshotFlow { listState.canScrollBackward }.collect { scrolledReport(it) } }
    // A category gone to stays the focus until the user scrolls, short or last as it may be.
    var held by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(listState) {
        snapshotFlow {
            val info = listState.layoutInfo
            held ?: Board.focused(onScreen, info.visibleItemsInfo.map { Seen(it.index, it.offset, it.size) }, info.viewportStartOffset, info.viewportEndOffset)
        }.collect { report(it) }
    }
    LaunchedEffect(listState) {
        listState.interactionSource.interactions.collect { if (it is DragInteraction.Start) held = null }
    }
    val jumped by rememberUpdatedState(onJumped)
    LaunchedEffect(jump) {
        val cat = jump ?: return@LaunchedEffect
        held = cat
        val at = withTimeoutOrNull(5_000) {
            snapshotFlow { onScreen.indexOfFirst { it is Header && it.cat == cat && !it.folded } }.first { it >= 0 }
        }
        if (at != null) listState.animateScrollToItem(at)
        jumped()
    }

    if (entry.project.categories.isEmpty()) {
        EmptyState("No categories yet", action = "New category", onAction = onNewCategory)
        return
    }

    if (searching && shown.isEmpty()) {
        EmptyState("Nothing matches")
        return
    }

    LaunchedEffect(query) { if (searching) listState.scrollToItem(0) }
    val bottom = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding() + 96.dp
    LazyColumn(state = listState, modifier = Modifier.fillMaxSize().testTag("list"), contentPadding = PaddingValues(bottom = bottom)) {
        itemsIndexed(shown, key = { _, it -> it.key }) { i, line ->
            ReorderableItem(reorder, key = line.key, enabled = swappable(line, i)) { dragging ->
                when (line) {
                    is Header -> CategoryHeader(
                        line,
                        onLongClick = onCategories,
                        onFold = if (searching) null else ({ fold(line.category.id, line.folded) }),
                    ) { onHeader(line.category) }
                    is Hidden -> HiddenLine(line) {
                        onUi(if (line.peeking) ui.copy(peek = null) else ui.copy(peek = line.cat))
                    }
                    is NoTasks -> NoTasksLine()
                    is Block -> {
                        val elevation by animateDpAsState(if (dragging) 6.dp else 0.dp, label = "lift")
                        // Each row is a handle of the block, so the autoscroll's edge test follows the row under the finger.
                        val handle = Modifier.draggableHandle(
                            enabled = !searching,
                            onDragStarted = { lift() },
                            onDragStopped = { dropped(line) },
                            dragGestureDetector = pull,
                        )
                        Surface(
                            tonalElevation = elevation,
                            shadowElevation = elevation,
                            color = if (dragging) MaterialTheme.colorScheme.surfaceContainerHigh else MaterialTheme.colorScheme.surface,
                        ) {
                            Column(Modifier.fillMaxWidth()) {
                                val cycle = { id: String, status: String ->
                                    if (tapping()) {
                                        if (filter.active) onUi(ui.copy(held = ui.held + id))
                                        // The step shows in place; done keeps its Undo, since the row may leave the view.
                                        val done = Status.next(status) == Status.COMPLETED
                                        scope.launch {
                                            if (engine.edit(pid, Ops.cycle(id), quiet = !done).applied && done) feedback.commit()
                                        }
                                    }
                                }
                                val open = { id: String -> if (tapping()) onOpen(id) }
                                TaskRow(
                                    line.row,
                                    child = false,
                                    onCycle = { cycle(line.row.task.id, line.row.task.status) },
                                    onOpen = { open(line.row.task.id) },
                                    modifier = handle,
                                    onFold = if (searching) null else ({ fold(line.row.task.id, line.row.folded) }),
                                )
                                for (c in line.children) {
                                    TaskRow(c, child = true, onCycle = { cycle(c.task.id, c.task.status) }, onOpen = { open(c.task.id) }, modifier = handle)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

// The long press that lifts a row, and the list as the autoscroll sees it: an edge scrolls the list
// only once the finger has moved toward it, so a row lifted near an edge doesn't carry the list off.
private class Pull(private val list: LazyListState) : ScrollableState by list, DragGestureDetector {
    private var y = 0f
    private var low = 0f
    private var high = 0f
    private var slop = 0f

    override val canScrollForward: Boolean get() = list.canScrollForward && y > low + slop
    override val canScrollBackward: Boolean get() = list.canScrollBackward && y < high - slop

    override suspend fun PointerInputScope.detect(
        onDragStart: (Offset) -> Unit,
        onDragEnd: () -> Unit,
        onDragCancel: () -> Unit,
        onDrag: (change: PointerInputChange, dragAmount: Offset) -> Unit,
    ) {
        slop = viewConfiguration.touchSlop
        detectDragGesturesAfterLongPress(
            onDragStart = {
                y = 0f
                low = 0f
                high = 0f
                onDragStart(it)
            },
            onDragEnd = onDragEnd,
            onDragCancel = onDragCancel,
            onDrag = { change, amount ->
                y += amount.y
                low = minOf(low, y)
                high = maxOf(high, y)
                onDrag(change, amount)
            },
        )
    }
}

internal fun headerAbove(order: List<Line>, block: Block): Header? {
    val i = order.indexOfFirst { it.key == block.key }
    return (i - 1 downTo 0).firstNotNullOfOrNull { order[it] as? Header }
}

// The places a lifted block can take: no row lands above the first header, nor beside a line that only counts rows.
internal fun swappable(line: Line, index: Int): Boolean = line is Block || (line is Header && index > 0)

// The order once the lifted block takes the place of the line keyed to, laid out as Board.lines would.
internal fun moved(order: List<Line>, from: String, to: String): List<Line>? {
    val f = order.indexOfFirst { it.key == from }
    val t = order.indexOfFirst { it.key == to }
    if (f < 0 || t < 0) return null
    return settled(order.toMutableList().apply { add(t, removeAt(f)) })
}

// Each category's rows above the line counting what the filter hides; "No tasks" goes once a row arrives.
private fun settled(order: List<Line>): List<Line> {
    val out = ArrayList<Line>(order.size)
    val tail = ArrayList<Line>()
    var rows = false
    val flush = {
        out += if (rows) tail.filter { it !is NoTasks } else tail
        tail.clear()
        rows = false
    }
    for (line in order) {
        when (line) {
            is Header -> {
                flush()
                out += line
            }
            is Block -> {
                out += line
                rows = true
            }
            else -> tail += line
        }
    }
    flush()
    return out
}

private fun rows(order: List<Line>) = order.filter { it is Header || it is Block }.map { it.key }

// Where a dropped block lands: under the header above it, above the next block of its category, first
// under a folded one. Nothing when the rows stand as they did at the lift.
internal fun dropOp(order: List<Line>, block: Block, lifted: List<Line>): JsonObject? {
    if (rows(order) == rows(lifted)) return null
    val i = order.indexOfFirst { it.key == block.key }
    if (i < 0) return null
    val header = headerAbove(order, block) ?: return null
    val id = block.row.task.id
    val before = if (header.folded) {
        header.category.tasks.firstOrNull { it.parentId == "" && it.id != id }?.id
    } else {
        order.drop(i + 1).takeWhile { it !is Header }.firstNotNullOfOrNull { (it as? Block)?.row?.task?.id }
    }
    return Ops.place(id, header.category.id, before.orEmpty())
}
