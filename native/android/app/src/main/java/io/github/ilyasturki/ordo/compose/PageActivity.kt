package io.github.ilyasturki.ordo.compose

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.LaunchedEffect
import io.github.ilyasturki.ordo.compose.model.json
import io.github.ilyasturki.ordo.compose.ui.App
import io.github.ilyasturki.ordo.compose.ui.LicenceKey
import io.github.ilyasturki.ordo.compose.ui.NavKey
import io.github.ilyasturki.ordo.compose.ui.OrdoTheme
import io.github.ilyasturki.ordo.compose.ui.SettingsKey
import io.github.ilyasturki.ordo.compose.ui.SignInKey
import io.github.ilyasturki.ordo.compose.ui.TaskKey
import io.github.ilyasturki.ordo.compose.ui.settings.LicenceScreen
import io.github.ilyasturki.ordo.compose.ui.settings.SettingsScreen
import io.github.ilyasturki.ordo.compose.ui.signin.SignInScreen
import io.github.ilyasturki.ordo.compose.ui.task.TaskScreen

private const val KEY = "key"

fun Context.openPage(key: NavKey) {
    startActivity(Intent(this, PageActivity::class.java).putExtra(KEY, json.encodeToString(NavKey.serializer(), key)))
}

// A page is an activity so that the system draws its open, its close and the predictive back gesture.
class PageActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        val key = json.decodeFromString(NavKey.serializer(), intent.getStringExtra(KEY).orEmpty())
        setContent {
            OrdoTheme {
                App(application as OrdoApp, signedOut = { LaunchedEffect(Unit) { finish() } }) {
                    when (key) {
                        is TaskKey -> TaskScreen(key.pid, key.tid, open = ::openPage, back = ::finish)
                        SettingsKey -> SettingsScreen(open = ::openPage, back = ::finish)
                        LicenceKey -> LicenceScreen(back = ::finish)
                        SignInKey -> SignInScreen(onBack = ::finish)
                    }
                }
            }
        }
    }
}
