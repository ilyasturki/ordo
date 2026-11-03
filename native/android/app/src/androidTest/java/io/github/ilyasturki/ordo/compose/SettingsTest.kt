package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollToNode
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Test
import org.junit.runner.RunWith

@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class SettingsTest : AppFixture() {
    private fun openSettings() {
        compose.waitUntilAtLeastOneExists(hasTestTag("list"), 20_000)
        compose.onNodeWithContentDescription("Settings").performClick()
        compose.waitUntilAtLeastOneExists(hasText("Devices"), 20_000)
    }

    @Test
    fun everyRowIsATargetAndTheServerSitsUnderDevices() {
        openSettings()
        compose.onNodeWithText("Server").assertDoesNotExist()
        compose.onNodeWithText("Sync").assertDoesNotExist()
        compose.onNodeWithText("Signed in to http://127.0.0.1:9").assertExists()
    }

    @Test
    fun theVersionOpensTheChangelog() {
        openSettings()
        compose.onNodeWithText("Version").performClick()
        compose.waitUntilAtLeastOneExists(hasTestTag("changelog"), 20_000)
        compose.onNodeWithTag("changelog").performScrollToNode(hasText("0.1.1"))
        compose.onNodeWithText("0.1.1").assertExists()
    }
}
