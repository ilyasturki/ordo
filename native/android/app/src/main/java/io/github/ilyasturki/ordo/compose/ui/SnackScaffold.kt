package io.github.ilyasturki.ordo.compose.ui

import androidx.compose.animation.core.animateIntAsState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp

// Scaffold's own slot floats the snackbar above the +: here it sits on the bottom edge and the + rises above it while it shows.
@Composable
fun SnackScaffold(
    modifier: Modifier = Modifier,
    topBar: @Composable () -> Unit,
    floatingActionButton: @Composable () -> Unit,
    content: @Composable (PaddingValues) -> Unit,
) {
    var snack by remember { mutableIntStateOf(0) }
    // M3's Snackbar pads itself by 12dp: the + keeps its own 16dp over the card, not over that margin.
    val margin = with(LocalDensity.current) { 12.dp.roundToPx() }
    val lift by animateIntAsState((snack - margin).coerceAtLeast(0), MaterialTheme.motionScheme.defaultSpatialSpec(), label = "lift")
    Box(modifier.fillMaxSize()) {
        Scaffold(
            topBar = topBar,
            floatingActionButton = { Box(Modifier.offset { IntOffset(0, -lift) }) { floatingActionButton() } },
            content = content,
        )
        SnackbarHost(
            LocalSnackbar.current,
            Modifier.align(Alignment.BottomCenter).navigationBarsPadding().onSizeChanged { snack = it.height }.testTag("snackbar"),
        )
    }
}
