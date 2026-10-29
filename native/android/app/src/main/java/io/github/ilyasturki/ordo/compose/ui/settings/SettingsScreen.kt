package io.github.ilyasturki.ordo.compose.ui.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation3.runtime.NavKey
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LicenceKey
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.home.ConfirmDialog
import io.github.ilyasturki.ordo.compose.ui.home.ago
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(open: (NavKey) -> Unit, back: () -> Unit) {
    val engine = LocalEngine.current
    val state by engine.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var signingOut by remember { mutableStateOf(false) }
    val version = remember { context.packageManager.getPackageInfo(context.packageName, 0).versionName.orEmpty() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = { IconButton(onClick = back) { Icon(Icons.Back, "Back") } },
            )
        },
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(pad).verticalScroll(rememberScrollState())) {
            Label("Account")
            Item(Icons.Dns, "Server", state.session?.server.orEmpty())
            Item(
                Icons.Sync,
                "Sync",
                buildString {
                    append(if (state.lastSync == 0L) "Not synced since the app started" else "Synced ${ago(state.lastSync)}")
                    if (state.pending > 0) append(" · ${state.pending} waiting")
                },
            )
            Item(Icons.Logout, "Sign out", "Drops this phone's copy; the server keeps everything it got", danger = true) { signingOut = true }
            Label("About")
            Item(Icons.Notes, "Font licence", "Google Sans Flex, SIL Open Font License 1.1") { open(LicenceKey) }
            Item(Icons.Label, "Version", version)
        }
    }

    if (signingOut) {
        val text = if (state.pending > 0) {
            "${state.pending} edits haven't reached the server yet: signing out drops them."
        } else {
            "The server keeps your projects; pair this phone again to get them back."
        }
        ConfirmDialog("Sign out?", text, "Sign out", { signingOut = false }) {
            scope.launch { engine.signOut() }
        }
    }
}

@Composable
private fun Label(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.padding(start = 24.dp, top = 24.dp, bottom = 8.dp),
    )
}

@Composable
private fun Item(icon: ImageVector, title: String, detail: String, danger: Boolean = false, onClick: (() -> Unit)? = null) {
    val color = if (danger) MaterialTheme.colorScheme.error else Color.Unspecified
    Row(
        Modifier
            .fillMaxWidth()
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 24.dp, vertical = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, tint = if (danger) color else MaterialTheme.colorScheme.onSurfaceVariant)
        Spacer(Modifier.width(20.dp))
        Column {
            Text(title, style = MaterialTheme.typography.bodyLarge, color = color)
            if (detail != "") Text(detail, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LicenceScreen(back: () -> Unit) {
    val context = LocalContext.current
    val text = remember { context.assets.open("google-sans-flex-OFL.txt").bufferedReader().use { it.readText() } }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Google Sans Flex") },
                navigationIcon = { IconButton(onClick = back) { Icon(Icons.Back, "Back") } },
            )
        },
    ) { pad ->
        Text(
            text,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(pad).verticalScroll(rememberScrollState()).padding(24.dp),
        )
    }
}
