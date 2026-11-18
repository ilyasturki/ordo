package io.github.ilyasturki.ordo.compose.ui.home

import android.os.SystemClock
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.gestures.ScrollableState
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.awaitLongPressOrCancellation
import androidx.compose.foundation.gestures.drag
import androidx.compose.foundation.interaction.DragInteraction
import androidx.compose.foundation.layout.Box
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
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.runtime.withFrameNanos
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.input.pointer.PointerInputChange
import androidx.compose.ui.input.pointer.PointerInputScope
import androidx.compose.ui.input.pointer.changedToUp
import androidx.compose.ui.input.pointer.positionChange
import androidx.compose.ui.input.pointer.positionChangeIgnoreConsumed
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
import io.github.ilyasturki.ordo.compose.model.Project
import io.github.ilyasturki.ordo.compose.model.Seen
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
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

// One instance for every page without a state: a fresh one on each recomposition would never compare equal.
val NoUi = ListUi()

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
    var lifted by remember { mutableStateOf(emptyList<Line>()) }
    var carried by remember { mutableStateOf<Block?>(null) }
    var landed by remember { mutableStateOf<Drop?>(null) }
    // Before the library lets the row go: the list as the drop leaves it, so the row settles in its last slot.
    val finish by rememberUpdatedState(suspend {
        val order = dragOrder
        val block = carried
        val drop = if (order != null && block != null) dropPlace(order, block, lifted) else null
        if (drop != null) {
            landed = drop
            val held = if (filter.active) ui.held + drop.id else ui.held
            dragOrder = Board.lines(entry.project.placed(drop), filter, ui.peek, held, if (drop.unfold) folds - drop.cat else folds)
            withFrameNanos {}
            withFrameNanos {}
        }
    })
    val swaps = remember { mutableIntStateOf(0) }
    val pull = remember(listState) { Pull(listState, { swaps.intValue }) { finish() } }
    val navBar = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding()
    val reorder = rememberReorderableLazyListState(
        listState,
        scrollThresholdPadding = remember(navBar) { PaddingValues(bottom = navBar) },
        // The library's own pace: a twentieth of the list each tenth of a second.
        scroller = rememberScroller(pull, { listState.layoutInfo.viewportSize.height * 0.05f }),
    ) { from, to ->
        val order = dragOrder ?: landing ?: lines
        val next = moved(order, from.key.toString(), to.key.toString(), up = to.index < from.index) ?: return@rememberReorderableLazyListState
        if (next.map { it.key } == order.map { it.key }) return@rememberReorderableLazyListState
        dragOrder = next
        swaps.intValue++
        feedback.slot()
    }
    val onScreen by rememberUpdatedState(shown)
    val drop: (Block) -> Unit = drop@{ block ->
        lifting = false
        droppedAt = SystemClock.uptimeMillis()
        val order = dragOrder
        dragOrder = null
        val drop = landed ?: order?.let { dropPlace(it, block, lifted) }
        landed = null
        carried = null
        if (order == null || drop == null) return@drop
        if (filter.active) onUi(ui.copy(held = ui.held + drop.id))
        landing = order
        // The edit lands even if the page goes on the drop; only what follows it dies with the page.
        val answer = appScope.async { engine.edit(pid, drop.op, quiet = true) }
        scope.launch {
            val a = answer.await()
            if (a.applied) feedback.commit() else if (a.status != "unchanged") feedback.reject()
            // As adding to it does: the row stays in sight where it landed.
            if (a.applied && drop.unfold) engine.setFolded(pid, drop.cat, false)
            if (landing === order) landing = null
        }
    }
    // A handle's gesture keeps the callbacks it started with: these reach the current ones.
    val lift by rememberUpdatedState { block: Block ->
        lifted = onScreen
        carried = block
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
    val blank = remember(shown) { entered(shown) }
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
                    is NoTasks -> Box(Modifier.alpha(if (line.key in blank) 0f else 1f)) { NoTasksLine() }
                    is Block -> {
                        val elevation by animateDpAsState(if (dragging) 6.dp else 0.dp, label = "lift")
                        // Each row is a handle of the block, so the autoscroll's edge test follows the row under the finger.
                        val handle = Modifier.draggableHandle(
                            enabled = !searching,
                            onDragStarted = { lift(line) },
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
// The library swaps only on a move of the finger, never while the last swap waits for its layout: each
// layout asks again, and the release waits for the order to catch up with where the finger left the row.
private class Pull(
    private val list: LazyListState,
    private val swaps: () -> Int,
    private val finish: suspend () -> Unit,
) : ScrollableState by list, DragGestureDetector {
    private var y = 0f
    private var low = 0f
    private var high = 0f
    private var slop = 0f
    private var letGo = false

    override val canScrollForward: Boolean get() = !letGo && list.canScrollForward && y > low + slop
    override val canScrollBackward: Boolean get() = !letGo && list.canScrollBackward && y < high - slop

    override suspend fun PointerInputScope.detect(
        onDragStart: (Offset) -> Unit,
        onDragEnd: () -> Unit,
        onDragCancel: () -> Unit,
        onDrag: (change: PointerInputChange, dragAmount: Offset) -> Unit,
    ) {
        slop = viewConfiguration.touchSlop
        coroutineScope {
            var last: PointerInputChange? = null
            var asking: Job? = null
            val move = { change: PointerInputChange, amount: Offset ->
                last = change
                y += amount.y
                low = minOf(low, y)
                high = maxOf(high, y)
                onDrag(change, amount)
            }
            // detectDragGesturesAfterLongPress, but for the release's own move: the system resamples a moving
            // finger to the frame, so the last move a flick sends falls short of where the finger lifts.
            awaitEachGesture {
                val down = awaitFirstDown(requireUnconsumed = false)
                val press = awaitLongPressOrCancellation(down.id) ?: return@awaitEachGesture
                y = 0f
                low = 0f
                high = 0f
                letGo = false
                onDragStart(press.position)
                // A frame after the layout, once the library has let go of the swap that waited for it.
                asking = launch {
                    snapshotFlow { list.layoutInfo }.collect {
                        withFrameNanos {}
                        last?.let { change -> onDrag(change, Offset.Zero) }
                    }
                }
                val ended = try {
                    drag(press.id) { move(it, it.positionChange()); it.consume() }
                } catch (e: CancellationException) {
                    asking?.cancel()
                    onDragCancel()
                    throw e
                }
                if (ended) {
                    currentEvent.changes.firstOrNull { it.changedToUp() }?.let { up ->
                        val rest = up.positionChangeIgnoreConsumed()
                        if (rest != Offset.Zero) move(up, rest)
                        up.consume()
                    }
                    letGo = true
                    launch {
                        settle { last?.let { change -> onDrag(change, Offset.Zero) } }
                        asking?.cancel()
                        last = null
                        finish()
                        onDragEnd()
                    }
                } else {
                    asking?.cancel()
                    last = null
                    onDragCancel()
                }
            }
        }
    }

    // A swap asked for lands within two frames.
    private suspend fun settle(ask: () -> Unit) {
        var seen = swaps()
        var still = 0
        var frames = 0
        while (still < 3 && frames++ < 30) {
            withFrameNanos {}
            ask()
            val now = swaps()
            if (now == seen) still++ else still = 0
            seen = now
        }
    }
}

// The places a lifted block can take: no row lands above the first header, nor beside a line that only counts rows.
internal fun swappable(line: Line, index: Int): Boolean = line is Block || (line is Header && index > 0)

// Every line keeps its place and height until the release, so each move is the swap the library expects.
// The library asks from a layout that can trail the order by a frame: the same ask again changes nothing.
internal fun moved(order: List<Line>, from: String, to: String, up: Boolean = order.indexOfFirst { it.key == to } < order.indexOfFirst { it.key == from }): List<Line>? {
    val f = order.indexOfFirst { it.key == from }
    if (f < 0 || from == to) return null
    val rest = order.toMutableList().apply { removeAt(f) }
    var t = rest.indexOfFirst { it.key == to }
    if (t < 0) return null
    if (!up) t++ else if (rest[t] is Header) while (t > 0 && (rest[t - 1] is Hidden || rest[t - 1] is NoTasks)) t--
    return rest.apply { add(t, order[f]) }
}

// The "No tasks" lines of categories a lifted block has entered: blank, so the list holds still under the finger.
internal fun entered(order: List<Line>): Set<String> {
    val out = HashSet<String>()
    var rows = false
    for (line in order) {
        when (line) {
            is Header -> rows = false
            is Block -> rows = true
            is NoTasks -> if (rows) out += line.key
            is Hidden -> {}
        }
    }
    return out
}

private fun rows(order: List<Line>) = order.filter { it is Header || it is Block }.map { it.key }

internal data class Drop(val id: String, val cat: String, val before: String, val unfold: Boolean) {
    val op: JsonObject get() = Ops.place(id, cat, before)
}

// Rows the filter hides between the rows shown around the drop stay below a block moved down, as J leaves
// them, and above one moved up, as K does, but for a separator among them: the block stops above it, in the
// section shown, so the separator stays hidden.
internal fun dropPlace(order: List<Line>, block: Block, lifted: List<Line>): Drop? {
    val shape = rows(order)
    if (shape == rows(lifted)) return null
    val i = order.indexOfFirst { it.key == block.key }
    if (i < 0) return null
    val h = (i - 1 downTo 0).firstOrNull { order[it] is Header } ?: return null
    val header = order[h] as Header
    val id = block.row.task.id
    val all = header.category.tasks
    val at = all.indexOfFirst { it.id == id }
    val tasks = if (at < 0) all else all.subList(0, at) + all.subList(Board.blockEnd(all, at), all.size)
    if (header.folded) return Drop(id, header.category.id, tasks.firstOrNull { it.parentId == "" }?.id.orEmpty(), true)
    val above = order.subList(h + 1, i).lastOrNull { it is Block } as Block?
    val below = order.subList(i + 1, order.size).takeWhile { it !is Header }.firstOrNull { it is Block } as Block?
    val to = below?.let { b -> tasks.indexOfFirst { it.id == b.row.task.id }.takeIf { it >= 0 } } ?: tasks.size
    val from = if (above == null) 0 else tasks.indexOfFirst { it.id == above.row.task.id }.let { if (it < 0) to else minOf(Board.blockEnd(tasks, it), to) }
    val up = shape.indexOf(block.key) < rows(lifted).indexOf(block.key)
    val slot = if (up) (from until to).firstOrNull { tasks[it].separator } ?: to else from
    return Drop(id, header.category.id, tasks.getOrNull(slot)?.id.orEmpty(), false)
}

// The project once the drop lands, as the core's task.place leaves it.
internal fun Project.placed(drop: Drop): Project {
    val src = categories.firstOrNull { c -> c.tasks.any { it.id == drop.id } } ?: return this
    val from = src.tasks.indexOfFirst { it.id == drop.id }
    val block = src.tasks.subList(from, Board.blockEnd(src.tasks, from))
    val ids = block.mapTo(HashSet()) { it.id }
    return copy(categories = categories.map { c ->
        val kept = c.tasks.filter { it.id !in ids }
        if (c.id != drop.cat) {
            if (kept.size == c.tasks.size) c else c.copy(tasks = kept)
        } else {
            val at = kept.indexOfFirst { it.id == drop.before }.takeIf { drop.before != "" && it >= 0 } ?: kept.size
            c.copy(tasks = kept.subList(0, at) + block + kept.subList(at, kept.size))
        }
    })
}
