package io.github.ilyasturki.ordo.compose.ui.task

import androidx.activity.compose.LocalActivity
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.dp
import androidx.core.view.WindowCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.model.Description
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import kotlin.math.min

private const val MAX_ZOOM = 5f
private const val DOUBLE_TAP_ZOOM = 2.5f

// Full screen on black, whatever the theme: pinch to zoom, swipe between images, back closes.
@Composable
fun ViewerScreen(pid: String, tid: String, start: Int, back: () -> Unit) {
    val engine = LocalEngine.current
    val state by engine.state.collectAsStateWithLifecycle()
    val task = state.project(pid)?.project?.find(tid)?.second
    val images = task?.let { t -> Description.refs(t.description).mapNotNull { it.file } }.orEmpty()
    if (images.isEmpty()) {
        LaunchedEffect(Unit) { back() }
        return
    }

    val activity = LocalActivity.current
    val view = LocalView.current
    DisposableEffect(Unit) {
        val bars = activity?.window?.let { WindowCompat.getInsetsController(it, view) }
        val light = bars?.isAppearanceLightStatusBars
        bars?.isAppearanceLightStatusBars = false
        onDispose { light?.let { bars.isAppearanceLightStatusBars = it } }
    }

    val pager = rememberPagerState(initialPage = start.coerceIn(0, images.lastIndex)) { images.size }
    var zoomed by remember { mutableStateOf(false) }
    Box(Modifier.fillMaxSize().background(Color.Black)) {
        HorizontalPager(
            state = pager,
            modifier = Modifier.fillMaxSize().testTag("viewer"),
            beyondViewportPageCount = 1,
            userScrollEnabled = !zoomed,
        ) { page ->
            ViewerPage(images[page], onZoomed = { zoomed = it })
        }
        Row(Modifier.statusBarsPadding(), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onClick = back) { Icon(Icons.Back, "Back", tint = Color.White) }
            if (images.size > 1) {
                Text("${pager.currentPage + 1} of ${images.size}", style = MaterialTheme.typography.titleSmall, color = Color.White)
            }
        }
    }
}

// Twice the screen, so a zoom stays sharp a while without holding the whole photo.
@Composable
private fun ViewerPage(name: String, onZoomed: (Boolean) -> Unit) {
    BoxWithConstraints(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        val density = LocalDensity.current
        val w = with(density) { maxWidth.roundToPx() }
        val h = with(density) { maxHeight.roundToPx() }
        val bitmap = rememberImage(name, w * 2, h * 2, cover = false)
        if (bitmap == null) {
            CircularProgressIndicator(color = Color.White)
        } else {
            Zoomable(bitmap, Size(w.toFloat(), h.toFloat()), onZoomed)
        }
    }
}

@Composable
private fun Zoomable(bitmap: ImageBitmap, box: Size, onZoomed: (Boolean) -> Unit) {
    var zoom by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }
    val fit = min(box.width / bitmap.width, box.height / bitmap.height)
    val shown = Size(bitmap.width * fit, bitmap.height * fit)
    fun clamped(o: Offset, z: Float): Offset {
        val x = ((shown.width * z - box.width) / 2).coerceAtLeast(0f)
        val y = ((shown.height * z - box.height) / 2).coerceAtLeast(0f)
        return Offset(o.x.coerceIn(-x, x), o.y.coerceIn(-y, y))
    }
    val transform = rememberTransformableState { centroid, zoomChange, panChange, _ ->
        val next = (zoom * zoomChange).coerceIn(1f, MAX_ZOOM)
        val at = centroid - Offset(box.width / 2, box.height / 2)
        offset = clamped(at * (1 - next / zoom) + offset * (next / zoom) + panChange, next)
        zoom = next
    }
    LaunchedEffect(zoom > 1f) { onZoomed(zoom > 1f) }
    Image(
        bitmap,
        null,
        Modifier
            .fillMaxSize()
            .pointerInput(Unit) {
                detectTapGestures(onDoubleTap = { at ->
                    val next = if (zoom > 1f) 1f else DOUBLE_TAP_ZOOM
                    val from = at - Offset(box.width / 2, box.height / 2)
                    offset = clamped(from * (1 - next), next)
                    zoom = next
                })
            }
            .transformable(transform, canPan = { zoom > 1f })
            .graphicsLayer {
                scaleX = zoom
                scaleY = zoom
                translationX = offset.x
                translationY = offset.y
            },
        contentScale = ContentScale.Fit,
    )
}
