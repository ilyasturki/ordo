package io.github.ilyasturki.ordo.compose

import android.app.Application
import android.net.ConnectivityManager
import android.net.Network
import android.os.Build
import android.provider.Settings
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
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.io.File

class OrdoApp : Application() {
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    lateinit var engine: SyncEngine
        private set

    override fun onCreate() {
        super.onCreate()
        engine = SyncEngine(
            store = Store(File(filesDir, "ordo")),
            newCore = ::GoCore,
            scope = scope,
            deviceName = deviceName(),
            background = { SyncWorker.enqueue(this) },
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

    // What the server's Devices page shows for this phone, told apart from the Capacitor app.
    private fun deviceName(): String {
        val name = Settings.Global.getString(contentResolver, Settings.Global.DEVICE_NAME)?.takeIf(String::isNotBlank) ?: Build.MODEL
        return "$name · Ordo Native"
    }
}
