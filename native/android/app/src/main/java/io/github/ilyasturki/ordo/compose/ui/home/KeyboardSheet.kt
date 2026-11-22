package io.github.ilyasturki.ordo.compose.ui.home

import android.content.res.Configuration
import android.os.Build
import android.os.CancellationSignal
import android.os.SystemClock
import android.view.View
import android.view.Window
import androidx.activity.compose.LocalActivity
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.AnimationSpec
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.animate
import androidx.compose.animation.core.tween
import androidx.compose.foundation.gestures.Orientation
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.draggable
import androidx.compose.foundation.gestures.rememberDraggableState
import androidx.compose.foundation.gestures.waitForUpOrCancellation
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.WindowInsetsSides
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.imeAnimationSource
import androidx.compose.foundation.layout.imeAnimationTarget
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.only
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.BottomSheetDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
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
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.LocalWindowInfo
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.onClick
import androidx.compose.ui.semantics.paneTitle
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.core.graphics.Insets
import androidx.core.view.ViewCompat
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsAnimationControlListenerCompat
import androidx.core.view.WindowInsetsAnimationControllerCompat
import androidx.core.view.WindowInsetsCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.navigationevent.NavigationEvent
import androidx.navigationevent.NavigationEventDispatcher
import androidx.navigationevent.NavigationEventHandler
import androidx.navigationevent.NavigationEventInfo
import androidx.navigationevent.compose.LocalNavigationEventDispatcherOwner
import io.github.ilyasturki.ordo.compose.ui.LocalAppScope
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import kotlin.math.max
import kotlin.math.min
import kotlin.math.roundToInt

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
// A swipe down or a back gesture moves the keyboard by hand, the sheet riding it; only the release or the completed back dismisses.
// Every dismissal (back, a swipe down, a tap outside, the keyboard's own hide) first runs onLeave; a refusal keeps the sheet up, resting on the bottom edge, and the next dismissal drops it.
@OptIn(ExperimentalLayoutApi::class, ExperimentalMaterial3Api::class)
@Composable
fun KeyboardSheet(onLeave: suspend () -> Boolean, onGone: () -> Unit, content: @Composable ColumnScope.(Finish) -> Unit) {
    val density = LocalDensity.current
    val appScope = LocalAppScope.current
    val scope = rememberCoroutineScope()
    val keyboard = LocalSoftwareKeyboardController.current
    val window = LocalWindowInfo.current
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    val view = LocalView.current
    val activity = LocalActivity.current
    val ime = WindowInsets.ime
    val source = WindowInsets.imeAnimationSource
    val target = WindowInsets.imeAnimationTarget
    val bars = WindowInsets.navigationBars
    val spatial = MaterialTheme.motionScheme.defaultSpatialSpec<Float>()
    // The sheet's presence while the keyboard isn't carrying it: none, or a hardware keyboard, or a refused close.
    val rest = remember { Animatable(0f) }
    // The sheet's own offset when no keyboard carries it: a drag or a back peek beside a hardware keyboard.
    val drag = remember { Animatable(0f) }
    val hand = remember { ImeHand(view, activity?.window) }
    var height by remember { mutableIntStateOf(0) }
    var closing by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf(false) }
    var refused by remember { mutableStateOf(false) }
    // A fall the sheet itself started before a late refusal: it isn't the next dismissal.
    var ownFall by remember { mutableStateOf(false) }
    var pulling by remember { mutableStateOf(false) }
    var pullFrom by remember { mutableFloatStateOf(0f) }
    var pull by remember { mutableFloatStateOf(0f) }
    var peek by remember { mutableFloatStateOf(0f) }
    val menus = remember { SheetMenus() }
    val leaving by rememberUpdatedState(onLeave)
    val gone by rememberUpdatedState(onGone)

    var tall by remember { mutableIntStateOf(0) }
    // A keyboard turned back mid-rise starts its fall from a source of 0: its height is the one it was rising to.
    val full by remember {
        derivedStateOf {
            val now = max(source.getBottom(density), target.getBottom(density))
            if (now > 0 || ime.getBottom(density) == 0) now else tall
        }
    }
    val raised by remember {
        derivedStateOf { if (full <= 0) 0f else (ime.getBottom(density).toFloat() / full).coerceIn(0f, 1f) }
    }
    // How far the sheet's top stands above the window's bottom: on the keyboard, its nav bar padding behind it, sinking into it as it falls.
    val lifted by remember {
        derivedStateOf {
            val kb = ime.getBottom(density).toFloat()
            val bar = bars.getBottom(density).toFloat()
            max(kb + raised * (height - bar), rest.value * (max(kb - bar, 0f) + height))
        }
    }
    val lowest = remember { floatArrayOf(Float.MAX_VALUE) }
    // Once the sheet is leaving it only goes down: a keyboard that shows late doesn't lift it again.
    val top by remember {
        derivedStateOf {
            val now = if (pulling) pullFrom - pull else lifted - drag.value
            if (closing) min(now, lowest[0]).also { lowest[0] = it } else now.also { lowest[0] = Float.MAX_VALUE }
        }
    }
    val presence by remember {
        derivedStateOf { if (lifted <= 0f) 0f else max(raised, rest.value) * (top / lifted).coerceIn(0f, 1f) }
    }

    // The keyboard's bottom inset that puts the sheet's top where the finger has it.
    fun underFinger(shown: Int): Float {
        val bar = bars.getBottom(density)
        return (pullFrom - pull) * shown / (shown + height - bar).coerceAtLeast(1)
    }

    // The keyboard's own way down, the sheet riding it. The field keeps focus until the sheet is gone: losing it detaches the keyboard, which then leaves on a motion of its own.
    fun drop() {
        keyboard?.hide()
        scope.launch { rest.animateTo(0f, spatial) }
    }

    fun carry(spec: AnimationSpec<Float>, velocity: Float = 0f) {
        scope.launch {
            if (hand.await()) {
                launch { drag.animateTo(0f, spec) }
                launch { rest.animateTo(0f, spec) }
                hand.settle(down = true, spec, velocity)
            } else {
                drop()
            }
        }
    }

    fun putBack(spec: AnimationSpec<Float>, velocity: Float = 0f) {
        scope.launch { drag.animateTo(0f, spec) }
        scope.launch { if (hand.await()) hand.settle(down = false, spec, velocity) }
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
            if (done && !closing) {
                closing = true
                drop()
            }
        }
    }

    // Saves and lowers at once; a refusal answered before the save suspends keeps the keyboard up, a later one rests the sheet.
    // False when it did nothing, so a gesture that ended on it is still the caller's to settle.
    fun leave(lower: () -> Unit): Boolean {
        if (closing || busy) return false
        if (refused) {
            closing = true
            lower()
            return true
        }
        busy = true
        var answered = false
        appScope.launch(Dispatchers.Main.immediate, CoroutineStart.UNDISPATCHED) {
            val done = leaving()
            answered = true
            busy = false
            when {
                done && !closing -> {
                    closing = true
                    lower()
                }
                done -> {}
                closing -> {
                    closing = false
                    refused = true
                    ownFall = hand.active || ime.getBottom(density) > 0
                    stayUp()
                }
                else -> {
                    refused = true
                    putBack(spatial)
                    stayUp()
                }
            }
        }
        if (!answered) {
            closing = true
            lower()
        }
        return true
    }

    fun release(spec: AnimationSpec<Float>, velocity: Float = 0f) = if (closing) carry(spec, velocity) else putBack(spec, velocity)

    // A sheet that takes over from another finds the keyboard up: it shows in place.
    LaunchedEffect(Unit) {
        if (target.getBottom(density) > 0) rest.snapTo(1f)
    }
    LaunchedEffect(Unit) {
        snapshotFlow { max(source.getBottom(density), target.getBottom(density)) }.collect { if (it > 0) tall = it }
    }
    // The keyboard heading down on its own is a dismissal only when this window let it go: a dialog over it or the app leaving takes it too.
    // A busy main thread can merge the keyboard's rise into its fall, so a keyboard first seen on its way down counts as well.
    LaunchedEffect(Unit) {
        var was: Keyboard? = null
        snapshotFlow {
            when {
                target.getBottom(density) > 0 -> Keyboard.UP
                ime.getBottom(density) > 0 -> Keyboard.FALLING
                else -> Keyboard.DOWN
            }
        }.distinctUntilChanged().collect { now ->
            val dismissed = was == Keyboard.UP || (was == Keyboard.DOWN && now == Keyboard.FALLING)
            if (dismissed && now != Keyboard.UP && !closing && !hand.active && !ownFall) {
                val ours = window.isWindowFocused && lifecycle.currentState.isAtLeast(Lifecycle.State.RESUMED)
                if (ours && !busy) leave(::drop) else stayUp()
            }
            if (now == Keyboard.DOWN) ownFall = false
            was = now
        }
    }
    // A keyboard brought back up after it went down since the refusal takes the refusal away; one still rising or put back by a cancelled gesture doesn't.
    LaunchedEffect(Unit) {
        var fromDown = false
        snapshotFlow {
            when {
                raised >= 1f && target.getBottom(density) > 0 -> 1
                ime.getBottom(density) == 0 -> -1
                else -> 0
            }
        }.distinctUntilChanged().collect { at ->
            if (at < 0 && refused) fromDown = true
            if (at > 0 && !closing) {
                rest.snapTo(0f)
                if (fromDown) refused = false
                fromDown = false
            }
        }
    }
    // Without a keyboard on its way, the sheet comes up on its own: at once beside a hardware keyboard, which keeps the software one down.
    val hardware = LocalConfiguration.current.hardKeyboardHidden == Configuration.HARDKEYBOARDHIDDEN_NO
    LaunchedEffect(Unit) {
        val coming = withTimeoutOrNull(if (hardware) 0L else NO_KEYBOARD_MS) { snapshotFlow { target.getBottom(density) > 0 }.first { it } }
        if (coming == null && !closing) rest.animateTo(1f, spatial)
    }
    // A keyboard asked for before the dismissal can still show after it: it goes straight back down.
    LaunchedEffect(closing) {
        if (!closing) return@LaunchedEffect
        snapshotFlow { target.getBottom(density) > 0 && !hand.active }.collect { if (it) keyboard?.hide() }
    }
    LaunchedEffect(closing) {
        if (!closing) return@LaunchedEffect
        withTimeoutOrNull(GONE_WAIT_MS) { snapshotFlow { ime.getBottom(density) == 0 && rest.value == 0f && !hand.active }.first { it } }
        snapshotFlow { busy }.first { !it }
        if (closing) gone()
    }
    DisposableEffect(hand) { onDispose { hand.end() } }

    // Above the keyboard's own back, for the sheet's whole life: a key back or a completed gesture saves, a cancelled one leaves everything as it was.
    val owner = LocalNavigationEventDispatcherOwner.current
    DisposableEffect(owner) {
        val back = SheetBack(
            progressed = { p ->
                if (!closing) {
                    peek = PEEK * BACK_EASING.transform(p)
                    if (hand.take { shown, from -> min(from.toFloat(), shown * (1f - peek)) }) {
                        hand.follow()
                    } else {
                        scope.launch { drag.snapTo(peek * height) }
                    }
                }
            },
            completed = {
                val left = if (hand.active) leave { carry(tween(POST_COMMIT_MS, easing = COMMIT_EASING)) } else leave(::drop)
                if (!left) release(tween(POST_COMMIT_MS, easing = if (closing) COMMIT_EASING else CANCEL_EASING))
            },
            cancelled = { release(tween(POST_COMMIT_MS, easing = CANCEL_EASING)) },
        )
        owner?.navigationEventDispatcher?.addHandler(back, NavigationEventDispatcher.PRIORITY_OVERLAY)
        onDispose { back.remove() }
    }

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
                        if (waitForUpOrCancellation() != null && !menu && menus.closedAt < down.uptimeMillis) leave(::drop)
                    }
                }
                .semantics {
                    contentDescription = "Close sheet"
                    onClick {
                        leave(::drop)
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
                .graphicsLayer { translationY = size.height - top }
                .draggable(
                    rememberDraggableState { d ->
                        pull = max(0f, pull + d)
                        hand.follow()
                    },
                    Orientation.Vertical,
                    enabled = !closing,
                    onDragStarted = {
                        pullFrom = top
                        pull = 0f
                        pulling = true
                        hand.take { shown, _ -> underFinger(shown) }
                    },
                    onDragStopped = { velocity ->
                        drag.snapTo(lifted - (pullFrom - pull))
                        pulling = false
                        val shown = full.coerceAtLeast(1)
                        val carried = -velocity * shown / (shown + height - bars.getBottom(density)).coerceAtLeast(1)
                        val far = pull > height / 2f || velocity > with(density) { FLING.toPx() }
                        if (!far || !leave { carry(spatial, carried) }) release(spatial, carried)
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

private class SheetBack(
    private val progressed: (Float) -> Unit,
    private val completed: () -> Unit,
    private val cancelled: () -> Unit,
) : NavigationEventHandler<NavigationEventInfo>(NavigationEventInfo.None, true) {
    override fun onBackProgressed(event: NavigationEvent) = progressed(event.progress)

    override fun onBackCompleted() = completed()

    override fun onBackCancelled() = cancelled()
}

// The keyboard moved by hand through the platform's insets controller: everything riding its insets follows.
private class ImeHand(private val view: View, private val window: Window?) : WindowInsetsAnimationControlListenerCompat {
    var active by mutableStateOf(false)
        private set
    private var controller: WindowInsetsAnimationControllerCompat? = null
    private var signal: CancellationSignal? = null
    private var ready = CompletableDeferred<Boolean>()
    private var goal: (shown: Int, from: Int) -> Float = { shown, _ -> shown.toFloat() }
    private var from = 0
    private var turn = 0

    // False when there is no software keyboard up to take: the caller moves the sheet itself.
    fun take(goal: (shown: Int, from: Int) -> Float): Boolean {
        this.goal = goal
        if (active) {
            turn++
            return true
        }
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.R) return false
        val window = window ?: return false
        if (ViewCompat.getRootWindowInsets(view)?.isVisible(WindowInsetsCompat.Type.ime()) != true) return false
        active = true
        ready = CompletableDeferred()
        val cancel = CancellationSignal()
        signal = cancel
        WindowCompat.getInsetsController(window, view).controlWindowInsetsAnimation(WindowInsetsCompat.Type.ime(), -1, null, cancel, this)
        return true
    }

    fun follow() {
        val c = controller ?: return
        place(c, goal(c.shownStateInsets.bottom, from))
    }

    // False when the platform never handed the keyboard over.
    suspend fun await(): Boolean {
        if (!active) return false
        if (withTimeoutOrNull(READY_MS) { ready.await() } == true) return true
        end()
        return false
    }

    suspend fun settle(down: Boolean, spec: AnimationSpec<Float>, velocity: Float = 0f) {
        val c = controller ?: return
        val mine = ++turn
        val end = if (down) c.hiddenStateInsets.bottom else c.shownStateInsets.bottom
        animate(c.currentInsets.bottom.toFloat(), end.toFloat(), velocity, spec) { value, _ ->
            if (turn == mine) controller?.let { place(it, value) }
        }
        if (turn == mine) controller?.finish(!down)
    }

    // The sheet gone mid-gesture: the keyboard goes with it.
    fun end() {
        val c = controller
        if (c != null) c.finish(false) else signal?.cancel()
        reset()
    }

    private fun place(c: WindowInsetsAnimationControllerCompat, bottom: Float) {
        if (!c.isReady) return
        val shown = c.shownStateInsets.bottom
        val hidden = c.hiddenStateInsets.bottom
        val at = bottom.roundToInt().coerceIn(hidden, shown)
        c.setInsetsAndAlpha(Insets.of(0, 0, 0, at), 1f, if (shown == hidden) 1f else (shown - at).toFloat() / (shown - hidden))
    }

    override fun onReady(controller: WindowInsetsAnimationControllerCompat, types: Int) {
        this.controller = controller
        from = controller.currentInsets.bottom
        ready.complete(true)
        follow()
    }

    override fun onFinished(controller: WindowInsetsAnimationControllerCompat) = reset()

    override fun onCancelled(controller: WindowInsetsAnimationControllerCompat?) = reset()

    private fun reset() {
        controller = null
        signal = null
        active = false
        ready.complete(false)
    }
}

private enum class Keyboard { UP, FALLING, DOWN }

// A software keyboard can take a second to start rising on a loaded phone.
private const val NO_KEYBOARD_MS = 2000L

// A keyboard that never goes down doesn't keep a closed sheet on screen.
private const val GONE_WAIT_MS = 1500L

// The platform hands over a keyboard that is up within a frame or two.
private const val READY_MS = 500L

// M3's bottom sheet velocity threshold.
private val FLING = 125.dp

// The platform's IME back animation (ImeBackAnimationController): a peek of a tenth while the finger moves, 200 ms to either end.
private const val PEEK = 0.1f
private const val POST_COMMIT_MS = 200
private val BACK_EASING = CubicBezierEasing(0.1f, 0.1f, 0f, 1f)
private val COMMIT_EASING = CubicBezierEasing(0.3f, 0f, 1f, 1f)
private val CANCEL_EASING = CubicBezierEasing(0.05f, 0.7f, 0.1f, 1f)
