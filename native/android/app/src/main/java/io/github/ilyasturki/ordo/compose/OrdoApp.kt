package io.github.ilyasturki.ordo.compose

import android.app.Application
import android.net.ConnectivityManager
import android.net.Network
import android.os.Build
import android.provider.Settings
import androidx.compose.material3.SnackbarDuration
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SnackbarResult
import androidx.lifecycle.DefaultLifecycleObserver
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.ProcessLifecycleOwner
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.sync.SyncEngine
import io.github.ilyasturki.ordo.compose.sync.SyncWorker
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.io.File

class OrdoApp : Application() {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val snackbar = SnackbarHostState()
    lateinit var engine: SyncEngine
        private set
    private var snacks: Job? = null

    override fun onCreate() {
        super.onCreate()
        use(
            SyncEngine(
                store = Store(File(filesDir, "ordo"), File(cacheDir, "images")),
                newCore = ::GoCore,
                scope = scope,
                deviceName = deviceName(),
                background = { SyncWorker.enqueue(this) },
            ),
        )
        scope.launch { engine.boot() }
        ProcessLifecycleOwner.get().lifecycle.addObserver(object : DefaultLifecycleObserver {
            override fun onStart(owner: LifecycleOwner) {
                scope.launch(start = CoroutineStart.UNDISPATCHED) { engine.foreground(true) }
            }

            override fun onStop(owner: LifecycleOwner) {
                scope.launch(start = CoroutineStart.UNDISPATCHED) { engine.foreground(false) }
            }
        })
        getSystemService(ConnectivityManager::class.java).registerDefaultNetworkCallback(
            object : ConnectivityManager.NetworkCallback() {
                override fun onAvailable(network: Network) {
                    scope.launch { engine.online() }
                }
            },
        )
    }

    // A snack outlives the page that made it: a task deleted on its page offers Undo on the list below.
    internal fun use(next: SyncEngine) {
        engine = next
        snacks?.cancel()
        snacks = scope.launch(start = CoroutineStart.UNDISPATCHED) {
            next.snacks.collect { snack ->
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
    }

    // What the server's Devices page shows for this phone, told apart from the Capacitor app.
    private fun deviceName(): String {
        val name = Settings.Global.getString(contentResolver, Settings.Global.DEVICE_NAME)?.takeIf(String::isNotBlank) ?: Build.MODEL
        return "$name · Ordo Native"
    }
}
