package io.github.ilyasturki.ordo.compose.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SnackbarHostState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.OrdoApp
import io.github.ilyasturki.ordo.compose.sync.SyncEngine
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.serialization.Serializable

@Serializable
sealed interface NavKey

@Serializable
data class TaskKey(val pid: String, val tid: String) : NavKey

@Serializable
data class ViewerKey(val pid: String, val tid: String, val start: Int) : NavKey

@Serializable
data class CategoriesKey(val pid: String) : NavKey

@Serializable
data object SettingsKey : NavKey

@Serializable
data object DevicesKey : NavKey

@Serializable
data object LicenceKey : NavKey

@Serializable
data object SignInKey : NavKey

// A category the categories page sends the list to; the list clears it once there.
data class Jump(val pid: String, val cat: String)

val LocalEngine = staticCompositionLocalOf<SyncEngine> { error("no engine") }
val LocalJump = staticCompositionLocalOf<MutableStateFlow<Jump?>> { error("no jump") }
val LocalSnackbar = staticCompositionLocalOf<SnackbarHostState> { error("no snackbar host") }

// Outlives a screen: what a page saves as it closes still lands.
val LocalAppScope = staticCompositionLocalOf<CoroutineScope> { error("no app scope") }

@Composable
fun App(app: OrdoApp, signedOut: @Composable () -> Unit, signedIn: @Composable () -> Unit) {
    val state by app.engine.state.collectAsStateWithLifecycle()
    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.surface)) {
        if (!state.booted) return@Box
        CompositionLocalProvider(LocalEngine provides app.engine, LocalSnackbar provides app.snackbar, LocalAppScope provides app.scope, LocalJump provides app.jump) {
            if (state.session == null) signedOut() else signedIn()
        }
    }
}
