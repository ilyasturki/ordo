package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.data.Store
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.File
import java.nio.file.Files
import java.util.Base64

private const val PHRASE = "correct horse battery"

class DevicesTest {
    private lateinit var server: Server
    private lateinit var dir: File
    private val cores = mutableListOf<PipeCore>()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private val quiet = Delays(sync = 600_000, poll = 600_000, firstBackoff = 600_000, maxBackoff = 600_000, reconnect = 600_000)

    @Before
    fun setUp() {
        server = Server()
        dir = Files.createTempDirectory("ordo-phone").toFile()
    }

    @After
    fun tearDown() {
        scope.cancel()
        cores.forEach(PipeCore::close)
        server.close()
        dir.deleteRecursively()
    }

    private fun engine(name: String = "test phone"): SyncEngine {
        val core = PipeCore().also(cores::add)
        return SyncEngine(Store(File(dir, name)), { core }, scope, name, quiet)
    }

    private fun <T> test(block: suspend CoroutineScope.() -> T): T = runBlocking { withTimeout(60_000) { block() } }

    private suspend fun paired(name: String = "test phone"): SyncEngine {
        val e = engine(name)
        e.boot()
        assertNull(e.pair(server.pairLink()))
        return e
    }

    @Test
    fun aServerWithoutAPassphraseOffersNone() = test {
        val e = engine()
        e.boot()
        assertFalse(e.offersPassphrase(server.url))
        assertFalse("no answer, no offer", e.offersPassphrase("http://127.0.0.1:9"))
        server.passphrase(PHRASE)
        assertTrue(e.offersPassphrase(server.url))
    }

    @Test
    fun theRightPassphraseSignsInAndLoadsTheProjects() = test {
        server.passphrase(PHRASE)
        val e = engine()
        e.boot()
        assertNull(e.login(server.url, PHRASE))
        assertTrue(e.state.value.signedIn)
        assertEquals(listOf("Ordo"), e.state.value.projects.map { it.name })
        assertNotNull(Store(File(dir, "test phone")).session())
        assertTrue(server.ordo("serve", "devices").contains("test phone"))
    }

    @Test
    fun theWrongPassphraseSaysSoAndLeavesThePhoneOut() = test {
        server.passphrase(PHRASE)
        val e = engine()
        e.boot()
        assertEquals("Wrong passphrase", e.login(server.url, "not the passphrase"))
        assertNull(e.state.value.session)
        assertTrue(server.ordo("serve", "devices").lines().none { it.contains("test phone") })
    }

    @Test
    fun anExpiredSessionSignsBackInWithThePassphrase() = test {
        server.passphrase(PHRASE)
        val e = engine()
        e.boot()
        assertNull(e.login(server.url, PHRASE))
        server.revoke("test phone")
        assertEquals(Round.STOP, e.sync())
        assertEquals(Trouble.Expired, e.state.value.trouble)
        assertNull(e.login(server.url, PHRASE))
        assertTrue(e.state.value.signedIn)
        assertEquals("the copy stayed", listOf("Ordo"), e.state.value.projects.map { it.name })
    }

    @Test
    fun theListMarksThisDeviceAndWhoPairedWhom() = test {
        val e = paired()
        val code = e.pairDevice("")
        val second = engine("second phone")
        second.boot()
        assertNull(second.pair(code.link))
        val devices = e.devices()
        assertFalse(devices.hasPassphrase)
        val mine = devices.devices.single { it.self }
        assertEquals("test phone", mine.name)
        assertEquals(listOf("second phone"), mine.also)
        assertEquals("test phone", devices.devices.single { !it.self }.from)
        assertTrue(mine.createdAt.isNotEmpty())
        assertTrue(mine.lastSeen.isNotEmpty())
    }

    @Test
    fun aPairingCodeComesWithAQrOfItsLink() = test {
        val code = paired().pairDevice("")
        assertTrue(code.link.contains("/pair/"))
        assertEquals(10, code.minutes)
        assertTrue(code.qr.startsWith("data:image/png;base64,"))
        val png = Base64.getDecoder().decode(code.qr.substringAfter(','))
        assertEquals(0x89, png[0].toInt() and 0xff)
    }

    @Test
    fun aPassphraseGuardsANewPairingCode() = test {
        val e = paired()
        server.passphrase(PHRASE)
        assertTrue(e.devices().hasPassphrase)
        val refused = runCatching { e.pairDevice("not the passphrase") }.exceptionOrNull()
        assertEquals("Wrong passphrase", refused?.message)
        assertTrue(e.pairDevice(PHRASE).link.contains("/pair/"))
    }

    @Test
    fun revokingAnotherDeviceSignsOutOnlyThatOne() = test {
        val e = paired()
        val second = engine("second phone")
        second.boot()
        assertNull(second.pair(e.pairDevice("").link))
        e.revoke(e.devices().devices.single { !it.self }.id)
        assertEquals(listOf("test phone"), e.devices().devices.map { it.name })
        assertTrue(e.state.value.signedIn)
        assertEquals(Round.STOP, second.sync())
        assertEquals(Trouble.Expired, second.state.value.trouble)
    }

    @Test
    fun revokingThisDeviceSignsItOutAndDropsItsCopy() = test {
        val e = paired()
        e.revoke(e.devices().devices.single { it.self }.id)
        val s = e.state.value
        assertNull(s.session)
        assertTrue(s.projects.isEmpty())
        assertNull(Store(File(dir, "test phone")).session())
        assertTrue(server.ordo("serve", "devices").lines().none { it.contains("test phone") })
    }

    @Test
    fun aRefusedTokenEndsTheSessionOnTheDevicesPageToo() = test {
        val e = paired()
        server.revoke("test phone")
        assertEquals(401, (runCatching { e.devices() }.exceptionOrNull() as ServerError).status)
        assertEquals(Trouble.Expired, e.state.value.trouble)
    }
}
