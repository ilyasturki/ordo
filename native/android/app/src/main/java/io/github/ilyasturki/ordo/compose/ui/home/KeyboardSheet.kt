package io.github.ilyasturki.ordo.compose.ui.home

import android.content.res.Configuration
import android.os.SystemClock
import androidx.activity.compose.BackHandler
import androidx.compose.animation.core.Animatable
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.waitForUpOrCancellation
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.exclude
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.imeAnimationSource
import androidx.compose.foundation.layout.imeAnimationTarget
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.BottomSheetDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.platform.LocalWindowInfo
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.onClick
import androidx.compose.ui.semantics.paneTitle
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import kotlin.math.max

// What a keyboard sheet's content runs to finish: the sheet closes once the commit lands, and holds every other close until it has answered.
typealias Finish = (commit: suspend () -> Boolean) -> Unit

// True once the sheet is on its way out: a menu it opened closes rather than outlive it.
val LocalSheetLeaving = compositionLocalOf { false }

// A menu that keeps the keyboard doesn't take the window's touches: a tap outside it also lands on the scrim, which lets it only close the menu.
class SheetMenus {
    var open = false
        private set
    var closedAt = 0L
        private set

    fun shown(open: Boolean) {
        if (this.open && !open) closedAt = SystemClock.uptimeMillis()
        this.open = open
    }
}

val LocalSheetMenus = compositionLocalOf<SheetMenus?> { null }

// A sheet in the page's own window, placed by the keyboard alone: it rises and leaves on the keyboard's own frames, as one motion.
// Every dismissal (back, a swipe down, a tap outside, the keyboard's own back) first runs onLeave; a refusal keeps the sheet up, resting on the bottom edge, and the next dismissal drops it.
@OptIn(ExperimentalLayoutApi::class, ExperimentalMaterial3Api::class)
@Composable
fun KeyboardSheet(onLeave: suspend () -> Boolean, onGone: () -> Unit, content: @Composable ColumnScope.(Finish) -> Unit) {
    val density = LocalDensity.current
    val appScope = LocalAppScope.current
    val scope = rememberCoroutineScope()
    val focusManager = LocalFocusManager.current
    val keyboard = LocalSoftwareKeyboardController.current
    val window = LocalWindowInfo.current
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    val ime = WindowInsets.ime
    val source = WindowInsets.imeAnimationSource
    val target = WindowInsets.imeAnimationTarget
    val lift = ime.exclude(WindowInsets.navigationBars)
    val spatial = MaterialTheme.motionScheme.defaultSpatialSpec<Float>()
    // The sheet's presence while the keyboard isn't carrying it: none, or a hardware keyboard, or a refused close.
    val rest = remember { Animatable(0f) }
    val drag = remember { Animatable(0f) }
    var height by remember { mutableStateOf(0) }
    var closing by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var refused by remember { mutableStateOf(false) }
    val menus = remember { SheetMenus() }
    val leaving by rememberUpdatedState(onLeave)
    val gone by rememberUpdatedState(onGone)

    val raised by remember {
        derivedStateOf {
            val full = max(source.getBottom(density), target.getBottom(density))
            if (full <= 0) 0f else (ime.getBottom(density).toFloat() / full).coerceIn(0f, 1f)
        }
    }
    val presence by remember { derivedStateOf { max(raised, rest.value) } }

    fun close() {
        if (closing) return
        closing = true
        focusManager.clearFocus(force = true)
        keyboard?.hide()
        scope.launch { rest.animateTo(0f, spatial) }
    }

    fun stayUp() {
        scope.launch {
            rest.snapTo(max(rest.value, raised))
            launch { drag.animateTo(0f, spatial) }
            rest.animateTo(1f, spatial)
        }
    }

    val finish: Finish = finish@{ commit ->
        if (closing || busy) return@finish
        busy = true
        appScope.launch(Dispatchers.Main) {
            val done = commit()
            busy = false
            if (done) close()
        }
    }

    fun leave() {
        if (closing || busy) return
        if (refused) return close()
        busy = true
        appScope.launch(Dispatchers.Main) {
            val done = leaving()
            busy = false
            if (done) {
                close()
            } else {
                refused = true
                stayUp()
            }
        }
    }

    // A sheet that takes over from another finds the keyboard up: it shows in place.
    LaunchedEffect(Unit) {
        if (target.getBottom(density) > 0) rest.snapTo(1f)
    }
    // The keyboard heading down is a dismissal only when this window let it go: a dialog over it or the app leaving takes it too.
    LaunchedEffect(Unit) {
        var wasUp = false
        snapshotFlow { target.getBottom(density) > 0 }.distinctUntilChanged().collect { up ->
            if (wasUp && !up && !closing) {
                val ours = window.isWindowFocused && lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)
                if (ours && !busy) leave() else stayUp()
            }
            wasUp = up
        }
    }
    LaunchedEffect(Unit) {
        snapshotFlow { raised >= 1f && target.getBottom(density) > 0 }.collect { up ->
            if (up && !closing) {
                rest.snapTo(0f)
                refused = false
            }
        }
    }
    // Without a keyboard on its way, the sheet comes up on its own: at once beside a hardware keyboard, which keeps the software one down.
    val hardware = LocalConfiguration.current.hardKeyboardHidden == Configuration.HARDKEYBOARDHIDDEN_NO
    LaunchedEffect(Unit) {
        val coming = withTimeoutOrNull(if (hardware) 0L else NO_KEYBOARD_MS) { snapshotFlow { target.getBottom(density) > 0 }.first { it } }
        if (coming == null && !closing) rest.animateTo(1f, spatial)
    }
    LaunchedEffect(closing) {
        if (!closing) return@LaunchedEffect
        withTimeoutOrNull(GONE_WAIT_MS) { snapshotFlow { ime.getBottom(density) == 0 && rest.value == 0f }.first { it } }
        gone()
    }
    BackHandler(!closing) { leave() }

    val scrim = BottomSheetDefaults.ScrimColor
    Box(Modifier.fillMaxSize()) {
        Box(
            Modifier
                .fillMaxSize()
                .drawBehind { drawRect(scrim, alpha = presence) }
                .pointerInput(Unit) {
                    awaitEachGesture {
                        val down = awaitFirstDown()
                        val menu = menus.open
                        if (waitForUpOrCancellation() != null && !menu && menus.closedAt < down.uptimeMillis) leave()
                    }
                }
                .semantics {
                    contentDescription = "Close sheet"
                    onClick {
                        leave()
                        true
                    }
                },
        )
        Surface(
            Modifier
                .align(Alignment.BottomCenter)
                .widthIn(max = BottomSheetDefaults.SheetMaxWidth)
                .fillMaxWidth()
                .onSizeChanged { height = it.height }
                .offset { IntOffset(0, -lift.getBottom(this)) }
                .graphicsLayer { translationY = (1f - presence) * size.height + drag.value }
                .draggable(
                    rememberDraggableState { d -> scope.launch { drag.snapTo((drag.value + d).coerceAtLeast(0f)) } },
                    Orientation.Vertical,
                    enabled = !closing,
                    onDragStopped = { velocity ->
                        if (drag.value > height / 2f || velocity > with(density) { FLING.toPx() }) leave() else drag.animateTo(0f, spatial)
                    },
                )
                .semantics { paneTitle = "Sheet" },
            shape = BottomSheetDefaults.ExpandedShape,
            color = BottomSheetDefaults.ContainerColor,
            tonalElevation = BottomSheetDefaults.Elevation,
        ) {
            CompositionLocalProvider(LocalSheetLeaving provides (closing || busy), LocalSheetMenus provides menus) {
                Column(Modifier.windowInsetsPadding(WindowInsets.navigationBars.only(WindowInsetsSides.Bottom))) { content(finish) }
            }
        }
    }
}

// A software keyboard can take a second to start rising on a loaded phone.
private const val NO_KEYBOARD_MS = 2000L

// A keyboard that never goes down doesn't keep a closed sheet on screen.
private const val GONE_WAIT_MS = 1500L

// M3's bottom sheet velocity threshold.
private val FLING = 125.dp
