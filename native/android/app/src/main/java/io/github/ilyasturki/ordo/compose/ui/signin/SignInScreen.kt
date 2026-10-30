package io.github.ilyasturki.ordo.compose.ui.signin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboard
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.codescanner.GmsBarcodeScannerOptions
import com.google.mlkit.vision.codescanner.GmsBarcodeScanning
import io.github.ilyasturki.ordo.compose.sync.parseServer
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SignInScreen(onBack: (() -> Unit)?) {
    val engine = LocalEngine.current
    val state by engine.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val clipboard = LocalClipboard.current
    val context = LocalContext.current
    val feedback = rememberFeedback()
    val focus = LocalFocusManager.current
    var link by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var passphrase by remember { mutableStateOf("") }
    var passError by remember { mutableStateOf<String?>(null) }
    // The session's server when it expired, else the address typed in the field: it decides if a passphrase is offered.
    val known = state.session?.server
    val server = known ?: parseServer(link)
    var offering by remember { mutableStateOf<String?>(null) }
    LaunchedEffect(server) {
        offering = null
        if (server == null) return@LaunchedEffect
        if (known == null) delay(400)
        if (engine.offersPassphrase(server)) offering = server
    }

    fun finished(failure: String?) {
        if (failure == null) {
            focus.clearFocus()
            feedback.commit()
            onBack?.invoke()
        } else {
            feedback.reject()
        }
    }

    fun pair(text: String) {
        if (busy) return
        busy = true
        error = null
        scope.launch {
            error = engine.pair(text)
            busy = false
            finished(error)
        }
    }

    fun login() {
        val at = offering
        if (busy || at == null || passphrase == "") return
        busy = true
        passError = null
        scope.launch {
            passError = engine.login(at, passphrase)
            busy = false
            finished(passError)
        }
    }

    Scaffold(
        topBar = {
            if (onBack != null) {
                TopAppBar(
                    title = { Text("Sign in again") },
                    navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.Back, "Back") } },
                )
            }
        },
    ) { pad ->
        Column(
            Modifier
                .fillMaxSize()
                .padding(pad)
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 32.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            if (onBack == null) {
                Text("Ordo", style = MaterialTheme.typography.displaySmall)
            }
            Text(
                "Pair this phone with your ordo serve. On a paired device, open Settings, then Devices, and pick “Pair a new device”, or run “ordo serve pair” on the server. Then scan its code or paste its link.",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (state.session?.expired == true && state.waiting) {
                Text(
                    "${state.waitingWords} on this phone: they sync once it is signed in again to the same server.",
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            FilledTonalButton(
                onClick = {
                    val options = GmsBarcodeScannerOptions.Builder().setBarcodeFormats(Barcode.FORMAT_QR_CODE).build()
                    GmsBarcodeScanning.getClient(context, options).startScan()
                        .addOnSuccessListener { code ->
                            code.rawValue?.let {
                                link = it
                                pair(it)
                            }
                        }
                        .addOnFailureListener { error = it.message }
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.QrScanner, null)
                Spacer(Modifier.width(8.dp))
                Text("Scan the code")
            }
            OutlinedTextField(
                link,
                {
                    link = it
                    error = null
                },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(if (known == null) "Pairing link or server address" else "Pairing link") },
                placeholder = { Text("https://…") },
                singleLine = true,
                isError = error != null,
                supportingText = error?.let { { Text(it) } },
                trailingIcon = {
                    IconButton(onClick = {
                        scope.launch {
                            clipboard.getClipEntry()?.clipData?.takeIf { it.itemCount > 0 }?.getItemAt(0)?.coerceToText(context)?.toString()?.let { link = it.trim() }
                        }
                    }) { Icon(Icons.Paste, "Paste") }
                },
                keyboardOptions = KeyboardOptions(autoCorrectEnabled = false, keyboardType = KeyboardType.Uri, imeAction = ImeAction.Go),
                keyboardActions = KeyboardActions(onGo = { pair(link) }),
            )
            if (offering == null || known != null) {
                Button(onClick = { pair(link) }, enabled = link.isNotBlank() && !busy, modifier = Modifier.fillMaxWidth()) {
                    Text(if (busy) "Pairing…" else "Pair")
                }
            }
            if (offering != null) {
                OutlinedTextField(
                    passphrase,
                    {
                        passphrase = it
                        passError = null
                    },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text("Passphrase") },
                    singleLine = true,
                    isError = passError != null,
                    supportingText = passError?.let { { Text(it) } },
                    visualTransformation = PasswordVisualTransformation(),
                    keyboardOptions = KeyboardOptions(autoCorrectEnabled = false, keyboardType = KeyboardType.Password, imeAction = ImeAction.Go),
                    keyboardActions = KeyboardActions(onGo = { login() }),
                )
                Button(onClick = ::login, enabled = passphrase != "" && !busy, modifier = Modifier.fillMaxWidth()) {
                    Text(if (busy) "Signing in…" else "Sign in")
                }
            }
            Spacer(Modifier.height(8.dp))
        }
    }
}
