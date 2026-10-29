package io.github.ilyasturki.ordo.compose.ui.home

import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.asPaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import io.github.ilyasturki.ordo.compose.model.Block
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Hidden
import io.github.ilyasturki.ordo.compose.model.Line
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonObject
import sh.calvin.reorderable.ReorderableItem
import sh.calvin.reorderable.rememberReorderableLazyListState

// A peek reveals one category's hidden rows; held rows stay where an edit from the list left
// them though the filter no longer matches. Both end when the project or the filter changes.
data class ListUi(val peek: String? = null, val held: Set<String> = emptySet())

@Composable
fun ProjectList(
    entry: Entry,
    filter: Filter,
    ui: ListUi,
    onUi: (ListUi) -> Unit,
    onOpen: (String) -> Unit,
    onHeader: (Category) -> Unit,
    onNewCategory: () -> Unit,
) {
    val engine = LocalEngine.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    val pid = entry.id
    val lines = remember(entry.project, filter, ui) { Board.lines(entry.project, filter, ui.peek, ui.held) }
    var dragOrder by remember { mutableStateOf<List<Line>?>(null) }
    val shown = dragOrder ?: lines
    val listState = rememberLazyListState()
    val reorder = rememberReorderableLazyListState(listState) { from, to ->
        if (to.index == 0) return@rememberReorderableLazyListState
        dragOrder = (dragOrder ?: lines).toMutableList().apply { add(to.index, removeAt(from.index)) }
        feedback.slot()
    }

    if (entry.project.categories.isEmpty()) {
        Column(
            Modifier.fillMaxSize().padding(32.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text("No categories yet", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
            FilledTonalButton(onClick = onNewCategory, modifier = Modifier.padding(top = 16.dp)) { Text("New category") }
        }
        return
    }

    val bottom = WindowInsets.navigationBars.asPaddingValues().calculateBottomPadding() + 96.dp
    LazyColumn(state = listState, modifier = Modifier.fillMaxSize().testTag("list"), contentPadding = PaddingValues(bottom = bottom)) {
        items(shown, key = { it.key }) { line ->
            ReorderableItem(reorder, key = line.key) { dragging ->
                when (line) {
                    is Header -> CategoryHeader(line) { onHeader(line.category) }
                    is Hidden -> HiddenLine(line) {
                        onUi(if (line.peeking) ui.copy(peek = null) else ui.copy(peek = line.cat))
                    }
                    is Block -> {
                        val lift by animateDpAsState(if (dragging) 6.dp else 0.dp, label = "lift")
                        Surface(
                            tonalElevation = lift,
                            shadowElevation = lift,
                            color = if (dragging) MaterialTheme.colorScheme.surfaceContainerHigh else MaterialTheme.colorScheme.surface,
                            modifier = Modifier.longPressDraggableHandle(
                                onDragStarted = { feedback.hold() },
                                onDragStopped = {
                                    val order = dragOrder
                                    val op = order?.let { dropOp(it, line) }
                                    if (op == null) {
                                        dragOrder = null
                                    } else {
                                        feedback.commit()
                                        scope.launch {
                                            engine.edit(pid, op, quiet = true)
                                            dragOrder = null
                                        }
                                    }
                                },
                            ),
                        ) {
                            Column(Modifier.fillMaxWidth()) {
                                val cycle = { id: String, status: String ->
                                    if (filter.active) onUi(ui.copy(held = ui.held + id))
                                    scope.launch {
                                        if (engine.edit(pid, Ops.cycle(id)).applied && next(status) == Status.COMPLETED) feedback.commit()
                                    }
                                    Unit
                                }
                                TaskRow(line.row, child = false, onCycle = { cycle(line.row.task.id, line.row.task.status) }, onOpen = { onOpen(line.row.task.id) })
                                for (c in line.children) {
                                    TaskRow(c, child = true, onCycle = { cycle(c.task.id, c.task.status) }, onOpen = { onOpen(c.task.id) })
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

private fun next(status: String): String = when (status) {
    Status.TODO -> Status.IN_PROGRESS
    Status.IN_PROGRESS -> Status.COMPLETED
    Status.COMPLETED -> Status.CANCELLED
    else -> Status.TODO
}

// Where a dropped block lands: under the header above it, above the next block of its category.
internal fun dropOp(order: List<Line>, block: Block): JsonObject? {
    val i = order.indexOfFirst { it.key == block.key }
    if (i < 0) return null
    val header = (i - 1 downTo 0).firstNotNullOfOrNull { order[it] as? Header } ?: return null
    var before = ""
    for (j in i + 1 until order.size) {
        val l = order[j]
        if (l is Header) break
        if (l is Block) {
            before = l.row.task.id
            break
        }
    }
    return Ops.place(block.row.task.id, header.category.id, before)
}
