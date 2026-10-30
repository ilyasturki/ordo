package io.github.ilyasturki.ordo.compose.ui.settings

import android.content.Intent
import android.graphics.BitmapFactory
import android.util.Base64
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.selection.SelectionContainer
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.FilterQuality
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.sync.DeviceInfo
import io.github.ilyasturki.ordo.compose.sync.Devices
import io.github.ilyasturki.ordo.compose.sync.PairCode
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.LocalSnackbar
import io.github.ilyasturki.ordo.compose.ui.home.ConfirmDialog
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

private val day = DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM)

private fun dayOf(at: String): String = runCatching { day.format(Instant.parse(at).atZone(ZoneId.systemDefault())) }.getOrDefault("")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DevicesScreen(back: () -> Unit) {
    val engine = LocalEngine.current
    val state by engine.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val snackbar = LocalSnackbar.current
    var devices by remember { mutableStateOf<Devices?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var revoking by remember { mutableStateOf<DeviceInfo?>(null) }
    var signingOut by remember { mutableStateOf(false) }
    var pairing by remember { mutableStateOf(false) }

    suspend fun load() {
        try {
            devices = engine.devices()
            error = null
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            error = e.message ?: "Can't reach ordo serve"
        }
    }
    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Devices") },
                navigationIcon = { IconButton(onClick = back) { Icon(Icons.Back, "Back") } },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(pad).verticalScroll(rememberScrollState())) {
            error?.let {
                Text(it, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(horizontal = 24.dp))
                TextButton(onClick = { scope.launch { load() } }, modifier = Modifier.padding(horizontal = 12.dp)) { Text("Retry") }
            }
            for (d in devices?.devices.orEmpty()) {
                DeviceRow(d) { revoking = d }
            }
            Item(Icons.QrCode, "Pair a new device", "Show a code to scan on the new device") { pairing = true }
            Item(Icons.Logout, "Sign out", "Drops this phone's copy; the server keeps everything it got", danger = true) { signingOut = true }
        }
    }

    revoking?.let { d ->
        val text = when {
            d.also.isNotEmpty() -> "Devices it paired sign out with it: ${d.also.joinToString(", ")}."
            d.self -> "This phone signs out and forgets its copy of your projects."
            else -> "It stops syncing until it pairs again."
        }
        ConfirmDialog("Sign out “${d.name}”?", text, "Sign out", { revoking = null }) {
            scope.launch {
                try {
                    engine.revoke(d.id)
                    load()
                } catch (e: CancellationException) {
                    throw e
                } catch (e: Exception) {
                    snackbar.showSnackbar(e.message ?: "Can't reach ordo serve")
                }
            }
        }
    }

    if (signingOut) {
        val text = if (state.waiting) {
            "Not on the server yet: ${state.waitingWords}. Signing out drops them."
        } else {
            "The server keeps your projects; pair this phone again to get them back."
        }
        ConfirmDialog("Sign out?", text, "Sign out", { signingOut = false }) {
            scope.launch { engine.signOut() }
        }
    }

    if (pairing) {
        PairSheet(devices?.hasPassphrase == true) { pairing = false }
    }
}

@Composable
private fun DeviceRow(d: DeviceInfo, onRevoke: () -> Unit) {
    val mark = MaterialTheme.colorScheme.primary
    ListItem(
        leadingContent = { Icon(Icons.Phone, null) },
        supportingContent = {
            val paired = "Paired ${dayOf(d.createdAt)}" + if (d.from != "") " by ${d.from}" else ""
            Text("$paired · seen ${dayOf(d.lastSeen)}")
        },
        trailingContent = {
            IconButton(onClick = onRevoke) { Icon(Icons.Logout, "Sign out ${d.name}") }
        },
    ) {
        Text(
            if (!d.self) AnnotatedString(d.name) else buildAnnotatedString {
                append(d.name)
                withStyle(SpanStyle(color = mark)) { append(" · this device") }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PairSheet(hasPassphrase: Boolean, onDismiss: () -> Unit) {
    val engine = LocalEngine.current
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    var code by remember { mutableStateOf<PairCode?>(null) }
    var passphrase by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }

    fun make() {
        if (busy) return
        busy = true
        error = null
        scope.launch {
            try {
                code = engine.pairDevice(passphrase)
                passphrase = ""
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                feedback.reject()
                error = e.message ?: "Can't reach ordo serve"
            } finally {
                busy = false
            }
        }
    }
    LaunchedEffect(Unit) { if (!hasPassphrase) make() }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp)
                .padding(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text("Pair a new device", style = MaterialTheme.typography.titleLarge)
            val c = code
            when {
                c != null -> PairingCode(c)
                hasPassphrase -> {
                    Text(
                        "Type the passphrase again to make a pairing code.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    OutlinedTextField(
                        passphrase,
                        {
                            passphrase = it
                            error = null
                        },
                        modifier = Modifier.fillMaxWidth(),
                        label = { Text("Passphrase") },
                        singleLine = true,
                        isError = error != null,
                        supportingText = error?.let { { Text(it) } },
                        visualTransformation = PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(autoCorrectEnabled = false, keyboardType = KeyboardType.Password, imeAction = ImeAction.Go),
                        keyboardActions = KeyboardActions(onGo = { if (passphrase != "") make() }),
                    )
                    Button(onClick = ::make, enabled = passphrase != "" && !busy, modifier = Modifier.fillMaxWidth()) {
                        Text(if (busy) "Making a code…" else "Make a code")
                    }
                }
                busy -> CircularProgressIndicator()
                else -> {
                    Text(error.orEmpty(), style = MaterialTheme.typography.bodyMedium, textAlign = TextAlign.Center)
                    FilledTonalButton(onClick = ::make) { Text("Retry") }
                }
            }
        }
    }
}

// The server draws the code: it answers it with the link, a PNG of black modules with their quiet zone on white.
@Composable
private fun PairingCode(code: PairCode) {
    val context = LocalContext.current
    val qr = remember(code.qr) {
        runCatching {
            val png = Base64.decode(code.qr.substringAfter(','), Base64.DEFAULT)
            BitmapFactory.decodeByteArray(png, 0, png.size)?.asImageBitmap()
        }.getOrNull()
    }
    if (qr != null) {
        Image(qr, "Pairing code", Modifier.size(280.dp).testTag("qr"), filterQuality = FilterQuality.None)
    }
    Text(
        "Scan it from the sign-in screen on the new device, or paste the link there. It works once, for ${code.minutes} minutes.",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        textAlign = TextAlign.Center,
    )
    SelectionContainer {
        Text(code.link, style = MaterialTheme.typography.bodySmall, textAlign = TextAlign.Center)
    }
    FilledTonalButton(
        onClick = {
            val send = Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, code.link)
            context.startActivity(Intent.createChooser(send, null))
        },
    ) {
        Icon(Icons.Share, null)
        Spacer(Modifier.width(8.dp))
        Text("Share the link")
    }
}
