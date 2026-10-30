package io.github.ilyasturki.ordo.compose

import android.app.Activity
import android.app.Instrumentation
import android.content.ClipData
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Color
import android.net.Uri
import android.provider.MediaStore
import android.view.KeyEvent
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.semantics.getOrNull
import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.doubleClick
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasProgressBarRangeInfo
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasTestTag
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithContentDescription
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.performTextInput
import androidx.compose.ui.test.performTextReplacement
import androidx.compose.ui.test.performTouchInput
import androidx.compose.ui.test.pinch
import androidx.compose.ui.test.swipeLeft
import androidx.compose.ui.test.swipeRight
import androidx.compose.ui.geometry.Offset
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.data.Session
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.model.Description
import io.github.ilyasturki.ordo.compose.model.Task
import io.github.ilyasturki.ordo.compose.sync.Api
import io.github.ilyasturki.ordo.compose.sync.Delays
import io.github.ilyasturki.ordo.compose.sync.ServerError
import io.github.ilyasturki.ordo.compose.sync.SyncEngine
import io.github.ilyasturki.ordo.compose.ui.App
import io.github.ilyasturki.ordo.compose.ui.OrdoTheme
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.job
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import java.io.ByteArrayOutputStream
import java.io.File

private const val PID = "0rd0-web-fixture"
private const val WAIT = 30_000L

// The app against the real ordo serve scripts/native.sh starts for these tests: -e server, -e passphrase.
@OptIn(ExperimentalTestApi::class)
@RunWith(AndroidJUnit4::class)
class ServerTest {
    @get:Rule
    val compose = createComposeRule()

    private val ins = InstrumentationRegistry.getInstrumentation()
    private val args = InstrumentationRegistry.getArguments()
    private val server get() = args.getString("server")!!
    private val passphrase get() = args.getString("passphrase")!!
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val name = "ui test ${System.nanoTime()}"
    private val delays = Delays(sync = 50, poll = 600_000, firstBackoff = 100, maxBackoff = 200, reconnect = 600_000)
    private val monitors = mutableListOf<Instrumentation.ActivityMonitor>()
    private lateinit var engine: SyncEngine
    private lateinit var dir: File

    @Before
    fun setUp() {
        assumeTrue("run through just native-ui", args.containsKey("server"))
        ins.targetContext.cacheDir.listFiles { f -> f.name.startsWith("live-") }?.forEach { it.deleteRecursively() }
        dir = File(ins.targetContext.cacheDir, "live-${System.nanoTime()}")
        engine = SyncEngine(Store(dir), ::GoCore, scope, name, delays)
        runBlocking {
            engine.boot()
            engine.foreground(true)
        }
    }

    @After
    fun tearDown() = runBlocking {
        monitors.forEach(ins::removeMonitor)
        // Its event stream would stay open: OkHttp queues a host's calls past five, and a queued call never times out.
        engine.foreground(false)
        scope.coroutineContext.job.cancelAndJoin()
    }

    private fun show() = compose.setContent { OrdoTheme { App(engine, scope) } }

    private fun signedIn() {
        runBlocking { assertNull(engine.login(server, passphrase)) }
        show()
    }

    private fun shows(text: String) = compose.onAllNodesWithText(text).fetchSemanticsNodes().isNotEmpty()

    private fun waitForText(text: String) = compose.waitUntil(WAIT) { shows(text) }

    private fun waitForDescription(text: String) =
        compose.waitUntil(WAIT) { compose.onAllNodesWithContentDescription(text).fetchSemanticsNodes().isNotEmpty() }

    private fun field(i: Int): SemanticsNodeInteraction = compose.onAllNodes(hasSetTextAction())[i]

    private fun back() {
        ins.sendKeyDownUpSync(KeyEvent.KEYCODE_BACK)
        compose.waitForIdle()
    }

    private fun inDialog(text: String) = compose.onNode(hasText(text) and hasAnyAncestor(isDialog()))

    private fun task(title: String): Task =
        engine.state.value.project(PID)!!.project.categories.flatMap { it.tasks }.first { it.title == title }

    private fun openTask(title: String) {
        compose.onNodeWithTag("list").performScrollToNode(hasText(title, substring = true))
        compose.onNodeWithText(title).performClick()
        compose.onNodeWithContentDescription("Add an image").assertIsDisplayed()
    }

    private fun images(title: String) = Description.refs(task(title).description)

    private fun png(color: Int, size: Int = 64): ByteArray {
        val bitmap = Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888).apply { eraseColor(color) }
        return ByteArrayOutputStream().also { bitmap.compress(Bitmap.CompressFormat.PNG, 100, it) }.toByteArray()
    }

    // A 2×2 BMP: a format the server refuses, as a phone's HEIC is, that any phone decodes.
    private fun bmp(): ByteArray {
        val pixel = byteArrayOf(0, 0, -1, 0)
        fun le(v: Int) = byteArrayOf(v.toByte(), (v shr 8).toByte(), (v shr 16).toByte(), (v shr 24).toByte())
        val head = "BM".toByteArray() + le(70) + le(0) + le(54) + le(40) + le(2) + le(2) +
            byteArrayOf(1, 0, 32, 0) + le(0) + le(16) + le(2835) + le(2835) + le(0) + le(0)
        return head + pixel + pixel + pixel + pixel
    }

    private fun file(name: String, bytes: ByteArray): Uri =
        Uri.fromFile(File(dir, name).apply { parentFile?.mkdirs(); writeBytes(bytes) })

    private fun answering(action: String, answer: (Intent) -> Instrumentation.ActivityResult) {
        val monitor = object : Instrumentation.ActivityMonitor() {
            override fun onStartActivity(intent: Intent): Instrumentation.ActivityResult? =
                if (intent.action == action) answer(intent) else null
        }
        ins.addMonitor(monitor)
        monitors += monitor
    }

    private fun pickerGives(vararg uris: Uri) = answering(MediaStore.ACTION_PICK_IMAGES) {
        val data = Intent().apply {
            clipData = ClipData.newRawUri("", uris[0]).also { c -> uris.drop(1).forEach { c.addItem(ClipData.Item(it)) } }
        }
        Instrumentation.ActivityResult(Activity.RESULT_OK, data)
    }

    private fun synced() = compose.waitUntil(WAIT) { !engine.state.value.waiting }

    private fun onServer(file: String): ByteArray {
        val s = engine.state.value.session!!
        val to = File(dir, "from-server")
        runBlocking { Api(s.server, s.token).image(file, to) }
        return to.readBytes()
    }

    @Test
    fun thePassphraseIsOfferedOnceTheAddressNamesAServerThatTakesOne() {
        show()
        compose.onAllNodesWithText("Passphrase").assertCountEquals(0)
        field(0).performTextInput(server)
        waitForText("Passphrase")
        field(1).performTextInput("not the passphrase")
        compose.onNodeWithText("Sign in").performClick()
        waitForText("Wrong passphrase")
        assertNull(engine.state.value.session)
        field(1).performTextReplacement(passphrase)
        compose.onNodeWithText("Sign in").performClick()
        waitForDescription("Add a task")
        assertEquals(listOf("Ordo"), engine.state.value.projects.map { it.name })
    }

    @Test
    fun anExpiredSessionOffersThePassphraseWithoutAskingTheAddress() {
        runBlocking {
            engine.signIn(Session(server, "a token the server never gave"))
            engine.sync()
        }
        assertTrue(engine.state.value.session!!.expired)
        show()
        compose.onNodeWithContentDescription("Not synced").performClick()
        compose.onNode(hasText("Sign in again") and hasClickAction()).performClick()
        waitForText("Passphrase")
        field(1).performTextInput(passphrase)
        compose.onNodeWithText("Sign in").performClick()
        compose.waitUntil(WAIT) { engine.state.value.signedIn }
    }

    @Test
    fun theDevicesPageMarksThisDeviceAndRevokesAnother() {
        val other = "other ${System.nanoTime()}"
        val token = runBlocking { Api(server, "").login(passphrase, other) }
        signedIn()
        compose.onNodeWithContentDescription("Account and settings").performClick()
        compose.onNodeWithText("Devices").performClick()
        waitForText("$name · this device")
        compose.onNodeWithContentDescription("Sign out $other").performClick()
        compose.onNodeWithText("Sign out “$other”?").assertExists()
        inDialog("Sign out").performClick()
        compose.waitUntil(WAIT) { compose.onAllNodesWithContentDescription("Sign out $other").fetchSemanticsNodes().isEmpty() }
        val refused = runCatching { runBlocking { Api(server, token).devices() } }.exceptionOrNull()
        assertEquals(401, (refused as ServerError).status)
    }

    @Test
    fun revokingThisDeviceEndsAtTheSignInScreen() {
        signedIn()
        compose.onNodeWithContentDescription("Account and settings").performClick()
        compose.onNodeWithText("Devices").performClick()
        waitForText("$name · this device")
        compose.onNodeWithContentDescription("Sign out $name").performClick()
        compose.onNodeWithText("This phone signs out and forgets its copy of your projects.").assertIsDisplayed()
        inDialog("Sign out").performClick()
        waitForText("Pair")
        assertNull(engine.state.value.session)
    }

    @Test
    fun signingOutAsksFirstAndEndsAtTheSignInScreen() {
        signedIn()
        compose.onNodeWithContentDescription("Account and settings").performClick()
        compose.onNodeWithText("Devices").performClick()
        waitForText("$name · this device")
        compose.onNodeWithText("Sign out").performClick()
        compose.onNodeWithText("Sign out?").assertIsDisplayed()
        inDialog("Sign out").performClick()
        waitForText("Pair")
        assertNull(engine.state.value.session)
    }

    @Test
    fun aNewDevicePairsFromTheLinkBesideTheCode() {
        signedIn()
        compose.onNodeWithContentDescription("Account and settings").performClick()
        compose.onNodeWithText("Devices").performClick()
        waitForText("$name · this device")
        compose.onNodeWithText("Pair a new device").performClick()
        waitForText("Type the passphrase again to make a pairing code.")
        compose.onNode(hasSetTextAction()).performTextInput(passphrase)
        compose.onNodeWithText("Make a code").performClick()
        compose.waitUntil(WAIT) { compose.onAllNodes(hasTestTag("qr")).fetchSemanticsNodes().isNotEmpty() }
        compose.onNodeWithTag("qr").assertIsDisplayed()
        compose.onNodeWithText("Share the link").assertExists()
        val link = compose.onNode(hasText("/pair/", substring = true)).fetchSemanticsNode()
            .config.getOrNull(SemanticsProperties.Text)!!.first().text
        val second = "second ${System.nanoTime()}"
        val other = SyncEngine(Store(File(dir, "second")), ::GoCore, scope, second, delays)
        runBlocking {
            other.boot()
            assertNull(other.pair(link))
        }
        // The first back may only close the keyboard.
        repeat(2) { if (compose.onAllNodes(hasTestTag("qr")).fetchSemanticsNodes().isNotEmpty()) back() }
        waitForDescription("Sign out $second")
    }

    @Test
    fun aPickedImageUploadsOpensAndIsRemovedWithUndo() {
        val bytes = png(Color.RED)
        pickerGives(file("Holiday in Rome.png", bytes))
        signedIn()
        openTask("Recurring tasks")
        compose.onNodeWithContentDescription("Holiday in Rome").assertDoesNotExist()
        compose.onNodeWithContentDescription("Add an image").performClick()
        compose.onNodeWithText("Choose a photo").performClick()
        compose.waitUntil(WAIT) { images("Recurring tasks").isNotEmpty() }
        val stored = images("Recurring tasks").single().file!!
        synced()
        assertArrayEquals(bytes, onServer(stored))
        compose.onNodeWithContentDescription("Holiday in Rome").performClick()
        compose.onNodeWithTag("viewer").assertIsDisplayed()
        compose.waitUntil(WAIT) { compose.onAllNodes(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).fetchSemanticsNodes().isEmpty() }
        back()
        compose.onNodeWithContentDescription("Remove Holiday in Rome").performClick()
        compose.waitUntil(WAIT) { images("Recurring tasks").isEmpty() }
        waitForText("Undo")
        compose.onNodeWithText("Undo").performClick()
        compose.waitUntil(WAIT) { images("Recurring tasks").isNotEmpty() }
        compose.onNodeWithContentDescription("Holiday in Rome").assertExists()
    }

    @Test
    fun aPhotoFromTheCameraUploads() {
        val bytes = png(Color.GREEN)
        answering(MediaStore.ACTION_IMAGE_CAPTURE) { intent ->
            val out = intent.getParcelableExtra(MediaStore.EXTRA_OUTPUT, Uri::class.java)!!
            ins.targetContext.contentResolver.openOutputStream(out)!!.use { it.write(bytes) }
            Instrumentation.ActivityResult(Activity.RESULT_OK, null)
        }
        signedIn()
        openTask("Export a project as Markdown")
        compose.onNodeWithContentDescription("Add an image").performClick()
        compose.onNodeWithText("Take a photo").performClick()
        compose.waitUntil(WAIT) { images("Export a project as Markdown").isNotEmpty() }
        val ref = images("Export a project as Markdown").single()
        assertEquals("photo", ref.name)
        synced()
        assertArrayEquals(bytes, onServer(ref.file!!))
    }

    @Test
    fun aFormatTheServerRefusesBecomesAJpeg() {
        pickerGives(file("scan.bmp", bmp()))
        signedIn()
        openTask("Write the install guide")
        compose.onNodeWithContentDescription("Add an image").performClick()
        compose.onNodeWithText("Choose a photo").performClick()
        compose.waitUntil(WAIT) { images("Write the install guide").isNotEmpty() }
        val stored = images("Write the install guide").single().file!!
        assertTrue(stored, stored.endsWith(".jpg"))
        synced()
        assertEquals(0xff, onServer(stored)[0].toInt() and 0xff)
    }

    @Test
    fun whatIsNotAnImageIsRefusedInPlainWords() {
        pickerGives(file("notes.txt", "not an image".toByteArray()))
        signedIn()
        openTask("Tap a row to focus it")
        compose.onNodeWithContentDescription("Add an image").performClick()
        compose.onNodeWithText("Choose a photo").performClick()
        waitForText("Only png, jpeg, gif and webp images can be attached")
        assertTrue(images("Tap a row to focus it").isEmpty())
    }

    @Test
    fun theViewerSwipesBetweenImagesAndAZoomHoldsTheSwipe() {
        pickerGives(file("first.png", png(Color.RED, 512)), file("second.png", png(Color.BLUE, 512)))
        signedIn()
        openTask("Pairing link login")
        compose.onNodeWithContentDescription("Add an image").performClick()
        compose.onNodeWithText("Choose a photo").performClick()
        compose.waitUntil(WAIT) { images("Pairing link login").size == 2 }
        compose.onNodeWithContentDescription("first").performClick()
        compose.onNodeWithText("1 of 2").assertIsDisplayed()
        compose.onNodeWithTag("viewer").performTouchInput { swipeLeft() }
        waitForText("2 of 2")
        compose.waitForIdle()
        compose.onNodeWithTag("viewer").performTouchInput {
            val c = center
            pinch(c - Offset(20f, 0f), c - Offset(300f, 0f), c + Offset(20f, 0f), c + Offset(300f, 0f))
        }
        compose.waitForIdle()
        compose.onNodeWithTag("viewer").performTouchInput { swipeLeft() }
        compose.onNodeWithText("2 of 2").assertIsDisplayed()
        compose.onNodeWithTag("viewer").performTouchInput { doubleClick() }
        compose.waitForIdle()
        compose.onNodeWithTag("viewer").performTouchInput { swipeRight() }
        waitForText("1 of 2")
        back()
        compose.onNodeWithContentDescription("Mark done").assertIsDisplayed()
    }

    @Test
    fun aRemoteImageDownloadsWithTheDevicesAuth() {
        val other = SyncEngine(Store(File(dir, "other")), ::GoCore, scope, "other ${System.nanoTime()}", delays)
        val bytes = png(Color.YELLOW)
        runBlocking {
            other.boot()
            assertNull(other.login(server, passphrase))
            val temp = other.images.newTemp().apply { writeBytes(bytes) }
            other.attach(PID, idOf(other, "Material 3 motion specs"), listOf(other.images.keep(temp, "remote.png")))
            other.sync()
        }
        signedIn()
        openTask("Material 3 motion specs")
        compose.onNodeWithContentDescription("remote").performClick()
        compose.waitUntil(WAIT) { compose.onAllNodes(hasProgressBarRangeInfo(ProgressBarRangeInfo.Indeterminate)).fetchSemanticsNodes().isEmpty() }
        val stored = images("Material 3 motion specs").single().file!!
        assertArrayEquals(bytes, engine.images.file(stored)!!.readBytes())
    }

    private fun idOf(e: SyncEngine, title: String): String =
        e.state.value.project(PID)!!.project.categories.flatMap { it.tasks }.first { it.title == title }.id
}
