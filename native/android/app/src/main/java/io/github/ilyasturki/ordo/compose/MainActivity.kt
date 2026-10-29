package io.github.ilyasturki.ordo.compose

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.github.ilyasturki.ordo.compose.ui.App
import io.github.ilyasturki.ordo.compose.ui.OrdoTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        val app = application as OrdoApp
        setContent { OrdoTheme { App(app.engine, app.scope) } }
    }
}
