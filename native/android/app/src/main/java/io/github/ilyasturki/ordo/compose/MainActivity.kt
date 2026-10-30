package io.github.ilyasturki.ordo.compose

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.github.ilyasturki.ordo.compose.ui.App
import io.github.ilyasturki.ordo.compose.ui.OrdoTheme
import io.github.ilyasturki.ordo.compose.ui.home.HomeScreen
import io.github.ilyasturki.ordo.compose.ui.signin.SignInScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContent {
            OrdoTheme {
                App(application as OrdoApp, signedOut = { SignInScreen(onBack = null) }) {
                    HomeScreen(open = ::openPage)
                }
            }
        }
    }
}
