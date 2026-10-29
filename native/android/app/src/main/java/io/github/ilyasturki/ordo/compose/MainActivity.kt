package io.github.ilyasturki.ordo.compose

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.material3.Text
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.core.perform
import io.github.ilyasturki.ordo.compose.model.Ops

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        val a = GoCore().perform(Ops.newId(), Ops.projectCreate("Inbox"))
        setContent { Text(a.toString()) }
    }
}
