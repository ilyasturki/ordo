package io.github.ilyasturki.ordo.compose.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation3.runtime.NavKey
import androidx.navigation3.runtime.entryProvider
import androidx.navigation3.runtime.rememberNavBackStack
import androidx.navigation3.ui.NavDisplay
import io.github.ilyasturki.ordo.compose.sync.SyncEngine
import io.github.ilyasturki.ordo.compose.ui.home.HomeScreen
import io.github.ilyasturki.ordo.compose.ui.settings.LicenceScreen
import io.github.ilyasturki.ordo.compose.ui.settings.SettingsScreen
import io.github.ilyasturki.ordo.compose.ui.signin.SignInScreen
import io.github.ilyasturki.ordo.compose.ui.task.TaskScreen
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable

@Serializable
data object HomeKey : NavKey

@Serializable
data class TaskKey(val pid: String, val tid: String) : NavKey

@Serializable
data object SettingsKey : NavKey

@Serializable
data object LicenceKey : NavKey

@Serializable
data object SignInKey : NavKey

val LocalEngine = staticCompositionLocalOf<SyncEngine> { error("no engine") }
val LocalSnackbar = staticCompositionLocalOf<SnackbarHostState> { error("no snackbar host") }

// Outlives a screen: what a page saves as it closes still lands.
val LocalAppScope = staticCompositionLocalOf<CoroutineScope> { error("no app scope") }

@Composable
fun App(engine: SyncEngine, appScope: CoroutineScope) {
    val state by engine.state.collectAsStateWithLifecycle()
    val snackbar = remember { SnackbarHostState() }
    LaunchedEffect(engine) {
        engine.snacks.collect { snack ->
            snackbar.currentSnackbarData?.dismiss()
            launch {
                val result = snackbar.showSnackbar(
                    snack.message,
                    actionLabel = snack.action,
                    duration = if (snack.action == null) SnackbarDuration.Short else SnackbarDuration.Long,
                )
                if (result == SnackbarResult.ActionPerformed) snack.run?.invoke()
            }
        }
    }
    Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.surface)) {
        if (!state.booted) return@Box
        CompositionLocalProvider(LocalEngine provides engine, LocalSnackbar provides snackbar, LocalAppScope provides appScope) {
            if (state.session == null) {
                SignInScreen(onBack = null)
            } else {
                Navigation()
            }
        }
    }
}

@Composable
private fun Navigation() {
    val stack = rememberNavBackStack(HomeKey)
    val go: (NavKey) -> Unit = { stack.add(it) }
    val back: () -> Unit = { if (stack.size > 1) stack.removeAt(stack.lastIndex) }
    NavDisplay(
        backStack = stack,
        onBack = back,
        entryProvider = entryProvider {
            entry<HomeKey> { HomeScreen(open = go) }
            entry<TaskKey> { key -> TaskScreen(key.pid, key.tid, open = go, back = back) }
            entry<SettingsKey> { SettingsScreen(open = go, back = back) }
            entry<LicenceKey> { LicenceScreen(back = back) }
            entry<SignInKey> { SignInScreen(onBack = back) }
        },
    )
}
