package io.github.ilyasturki.ordo.compose

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import android.widget.Toast
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

// just dev-phone's way in: adb shell am start -n <app id>/io.github.ilyasturki.ordo.compose.DevSignInActivity --es link <pairing link>
class DevSignInActivity : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val app = application as OrdoApp
        val link = intent.getStringExtra("link").orEmpty()
        app.scope.launch {
            app.engine.awaitBoot()
            val refused = app.engine.pair(link) ?: return@launch
            withContext(Dispatchers.Main) { Toast.makeText(app, refused, Toast.LENGTH_LONG).show() }
        }
        startActivity(Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP))
        finish()
    }
}
