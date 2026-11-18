package io.github.ilyasturki.ordo.compose.ui.home

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.FloatingActionButtonMenu
import androidx.compose.material3.FloatingActionButtonMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.material3.ToggleFloatingActionButton
import androidx.compose.material3.ToggleFloatingActionButtonDefaults.animateIcon
import androidx.compose.runtime.Composable
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChange
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.onClick
import androidx.compose.ui.semantics.onLongClick
import androidx.compose.ui.semantics.role
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback

enum class AddKind(val label: String, val icon: ImageVector) {
    PROJECT("Project", Icons.Tab),
    CATEGORY("Category", Icons.NewFolder),
    SEPARATOR("Separator", Icons.Separator),
    NOTE("Note", Icons.Note),
    TASK("Task", Icons.Todo),
}

// The + adds a task in one tap, as in Google Tasks; held, it opens the kinds, the first nearest the thumb.
@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
fun AddButton(name: String, kinds: List<AddKind>, open: Boolean, onOpen: (Boolean) -> Unit, onTap: () -> Unit, onPick: (AddKind) -> Unit) {
    val feedback = rememberFeedback()
    BackHandler(open) { onOpen(false) }
    val tap = { if (open) onOpen(false) else onTap() }
    val hold = {
        feedback.hold()
        onOpen(true)
    }
    FloatingActionButtonMenu(
        expanded = open,
        horizontalAlignment = Alignment.End,
        button = {
            ToggleFloatingActionButton(
                checked = open,
                onCheckedChange = { tap() },
                modifier = Modifier
                    .held(hold)
                    .clearAndSetSemantics {
                        contentDescription = if (open) "Close" else name
                        role = Role.Button
                        onClick {
                            tap()
                            true
                        }
                        onLongClick("More to add") {
                            hold()
                            true
                        }
                    },
            ) {
                val icon by remember { derivedStateOf { if (checkedProgress > 0.5f) Icons.Close else Icons.Add } }
                Icon(icon, null, Modifier.animateIcon({ checkedProgress }))
            }
        },
    ) {
        for (k in kinds.reversed()) {
            FloatingActionButtonMenuItem(
                onClick = {
                    onOpen(false)
                    onPick(k)
                },
                text = { Text(k.label) },
                icon = { Icon(k.icon, null) },
            )
        }
    }
}

// A long press seen before the toggle sees it: the release that follows is swallowed, so it never reaches the toggle as a tap.
private fun Modifier.held(onHold: () -> Unit): Modifier = pointerInput(Unit) {
    awaitEachGesture {
        awaitFirstDown(requireUnconsumed = false, pass = PointerEventPass.Initial)
        val released = withTimeoutOrNull(viewConfiguration.longPressTimeoutMillis) {
            var moved = 0f
            do {
                val e = awaitPointerEvent(PointerEventPass.Initial)
                moved += e.changes.sumOf { it.positionChange().getDistance().toDouble() }.toFloat()
            } while (e.changes.any { it.pressed } && moved <= viewConfiguration.touchSlop)
            true
        }
        if (released != null) return@awaitEachGesture
        onHold()
        do {
            val e = awaitPointerEvent(PointerEventPass.Initial)
            e.changes.forEach { it.consume() }
        } while (e.changes.any { it.pressed })
    }
}
