package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.data.ImageRefused
import io.github.ilyasturki.ordo.compose.data.Images
import io.github.ilyasturki.ordo.compose.data.Kept
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.model.Description
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.model.Project
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.async
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeout
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.File
import java.nio.file.Files
import java.util.Collections
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import kotlin.random.Random

private const val PID = "0rd0-web-fixture"
private const val EXPORT = "aaaaaaaa-0000-4000-8000-000000000005"
private const val RECURRING = "aaaaaaaa-0000-4000-8000-000000000007"

private val PNG = byteArrayOf(0x89.toByte(), 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a)

// The real file only the server's sniff reads: a header and bytes no other test made.
private fun png(): ByteArray = PNG + Random.nextBytes(256)

class ImagesTest {
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

    private fun engine(http: OkHttpClient = Api.client, background: () -> Unit = {}, at: File = dir): SyncEngine {
        val core = PipeCore().also(cores::add)
        return SyncEngine(Store(at), { core }, scope, "test phone", quiet, api = { Api(it.server, it.token, http) }, background = background)
    }

    private fun <T> test(block: suspend CoroutineScope.() -> T): T = runBlocking { withTimeout(60_000) { block() } }

    private suspend fun paired(e: SyncEngine): SyncEngine {
        e.boot()
        assertNull(e.pair(server.pairLink()))
        return e
    }

    private fun keep(e: SyncEngine, bytes: ByteArray = png(), alt: String = "photo.png"): Kept {
        val temp = e.images.newTemp()
        temp.writeBytes(bytes)
        return e.images.keep(temp, alt)
    }

    private fun serverDescription(id: String) = Project.parse(server.project(PID)).find(id)!!.second.description

    private fun phoneDescription(e: SyncEngine, id: String) = e.state.value.project(PID)!!.project.find(id)!!.second.description

    private suspend fun attachOne(e: SyncEngine, id: String = EXPORT, bytes: ByteArray = png()): Kept {
        val kept = keep(e, bytes)
        assertTrue(e.attach(PID, id, listOf(kept)).applied)
        return kept
    }

    @Test
    fun aPhotoReachesTheServerThenItsLine() = test {
        val e = paired(engine())
        val bytes = png()
        val kept = attachOne(e, bytes = bytes)
        val s = e.state.value
        assertEquals(1, s.pending)
        assertEquals(1, s.uploads)
        assertEquals("1 edit and 1 photo", s.waitingWords)
        assertTrue(phoneDescription(e, EXPORT).contains(kept.line))
        assertEquals(Round.OK, e.sync())
        assertArrayEquals(bytes, server.attachment(kept.name))
        assertTrue(serverDescription(EXPORT).contains(kept.line))
        assertFalse(e.state.value.waiting)
        assertNotNull("it stays on the phone, no download", e.image(kept.name))
    }

    @Test
    fun anOfflinePhotoRetriesAndArrivesOnceTheServerIsBack() = test {
        val asked = AtomicInteger()
        val e = paired(engine(background = { asked.incrementAndGet() }))
        server.stop()
        val kept = attachOne(e)
        assertEquals("the app is out of sight: WorkManager takes it", 1, asked.get())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(1, e.state.value.uploads)
        assertEquals(1, e.state.value.pending)
        assertTrue((e.state.value.trouble as Trouble.Failing).offline)
        assertTrue(asked.get() >= 4)
        server.start()
        assertEquals(Round.OK, e.sync())
        assertNotNull(server.attachment(kept.name))
        assertTrue(serverDescription(EXPORT).contains(kept.line))
        assertNull(e.state.value.trouble)
    }

    @Test
    fun onlyAPhotoWaitingStillCountsAsTrouble() = test {
        val e = paired(engine())
        server.stop()
        keep(e)
        e.attach(PID, EXPORT, emptyList())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(Round.RETRY, e.sync())
        assertEquals(0, e.state.value.pending)
        assertTrue(e.state.value.trouble is Trouble.Failing)
    }

    @Test
    fun aPhotoSurvivesARestart() = test {
        val kept = paired(engine()).let {
            server.stop()
            attachOne(it)
        }
        val again = engine()
        again.boot()
        assertEquals(1, again.state.value.uploads)
        assertEquals(1, again.state.value.pending)
        assertTrue(phoneDescription(again, EXPORT).contains(kept.line))
        server.start()
        assertEquals(Round.OK, again.sync())
        assertNotNull(server.attachment(kept.name))
        assertTrue(serverDescription(EXPORT).contains(kept.line))
    }

    @Test
    fun anExpiredSessionKeepsThePhotoUntilTheNextSignIn() = test {
        val e = paired(engine())
        server.revoke("test phone")
        val kept = attachOne(e)
        assertEquals(Round.STOP, e.sync())
        assertEquals(Trouble.Expired, e.state.value.trouble)
        assertEquals(1, e.state.value.uploads)
        assertNull(server.attachment(kept.name))
        assertNull(e.pair(server.pairLink()))
        assertEquals(0, e.state.value.uploads)
        assertNotNull(server.attachment(kept.name))
        assertTrue(serverDescription(EXPORT).contains(kept.line))
    }

    @Test
    fun aRoundThatFailsOnThePhotoSendsNoEdit() = test {
        val syncs = AtomicInteger()
        var refuse = true
        val flaky = Interceptor { chain ->
            val path = chain.request().url.encodedPath
            if (path == "/api/sync") syncs.incrementAndGet()
            if (refuse && chain.request().method == "PUT") answer(chain, 503, "Busy") else chain.proceed(chain.request())
        }
        val e = paired(engine(Api.client.newBuilder().addInterceptor(flaky).build()))
        val before = syncs.get()
        val kept = attachOne(e)
        assertEquals(Round.RETRY, e.sync())
        assertEquals(before, syncs.get())
        assertFalse(serverDescription(EXPORT).contains(kept.line))
        refuse = false
        assertEquals(Round.OK, e.sync())
        assertTrue(serverDescription(EXPORT).contains(kept.line))
    }

    @Test
    fun aPhotoKeptDuringARoundGoesAheadOfItsEdit() = test {
        val order = Collections.synchronizedList(mutableListOf<String>())
        val entered = CountDownLatch(1)
        val release = CountDownLatch(1)
        var hold = true
        val watch = Interceptor { chain ->
            val req = chain.request()
            if (req.method == "PUT") {
                order += "put"
                if (hold) {
                    hold = false
                    entered.countDown()
                    release.await(20, TimeUnit.SECONDS)
                }
            }
            if (req.url.encodedPath == "/api/sync") order += "sync"
            chain.proceed(req)
        }
        val e = paired(engine(Api.client.newBuilder().addInterceptor(watch).build()))
        order.clear()
        attachOne(e)
        val round = async(Dispatchers.IO) { e.sync() }
        assertTrue(entered.await(20, TimeUnit.SECONDS))
        val second = attachOne(e, RECURRING)
        release.countDown()
        assertEquals(Round.OK, round.await())
        assertEquals(listOf("put", "put", "sync"), order.toList())
        assertNotNull(server.attachment(second.name))
        assertTrue(serverDescription(RECURRING).contains(second.line))
    }

    @Test
    fun aRefusedPhotoNeitherBlocksTheEditsNorComesBack() = test {
        val refuse = Interceptor { chain ->
            if (chain.request().method == "PUT") answer(chain, 413, "That image is too large") else chain.proceed(chain.request())
        }
        val e = paired(engine(Api.client.newBuilder().addInterceptor(refuse).build()))
        val snacks = Collections.synchronizedList(mutableListOf<Snack>())
        val collecting = launch { e.snacks.collect { snacks += it } }
        attachOne(e)
        e.edit(PID, Ops.cycle(RECURRING))
        assertEquals(Round.OK, e.sync())
        assertFalse(e.state.value.waiting)
        assertEquals("in_progress", Project.parse(server.project(PID)).find(RECURRING)!!.second.status)
        withTimeout(5_000) {
            while (snacks.none { it.message.contains("too large") }) kotlinx.coroutines.delay(20)
        }
        collecting.cancel()
    }

    @Test
    fun aPhotoFromAnotherDeviceDownloadsOnceAndStaysOnThePhone() = test {
        val other = paired(engine(at = File(dir, "other")))
        val bytes = png()
        val kept = attachOne(other, bytes = bytes)
        assertEquals(Round.OK, other.sync())
        val e = paired(engine(at = File(dir, "phone")))
        assertTrue(phoneDescription(e, EXPORT).contains(kept.line))
        assertArrayEquals(bytes, e.image(kept.name)!!.readBytes())
        server.stop()
        assertArrayEquals(bytes, e.image(kept.name)!!.readBytes())
        assertNull("nothing on the phone and no server", e.image("0".repeat(32) + ".png"))
    }

    @Test
    fun signingOutDropsTheImages() = test {
        val e = paired(engine())
        val waiting = attachOne(e)
        e.signOut()
        assertEquals(0, e.state.value.uploads)
        assertNull(e.image(waiting.name))
        assertTrue(Store(dir).images.waiting().isEmpty())
    }

    @Test
    fun theLineUsesTheAltTheFileNameGives() {
        assertEquals("holiday in Rome", Images.altOf("holiday  in\tRome.jpg"))
        assertEquals("a b", Images.altOf("a[b].png"))
        assertEquals("photo", Images.altOf("image.jpeg"))
        assertEquals("photo", Images.altOf(""))
    }

    @Test
    fun theSniffAgreesWithTheServer() {
        assertEquals("png", Images.extension(PNG))
        assertEquals("jpg", Images.extension(byteArrayOf(0xff.toByte(), 0xd8.toByte(), 0xff.toByte())))
        assertEquals("gif", Images.extension("GIF89a".toByteArray()))
        assertEquals("webp", Images.extension("RIFFxxxxWEBPVP8 ".toByteArray()))
        assertNull(Images.extension("RIFFxxxxWAVEfmt ".toByteArray()))
        assertNull(Images.extension("ftypheic".toByteArray()))
        assertNull(Images.extension(ByteArray(0)))
    }

    @Test
    fun whatTheServerWouldRefuseIsRefusedOnThePhone() {
        val images = Store(dir).images
        val text = images.newTemp().apply { writeText("not an image") }
        assertEquals(Images.NOT_AN_IMAGE, runCatching { images.keep(text, "a") }.exceptionOrNull()?.message)
        val big = images.newTemp()
        big.outputStream().use { out ->
            out.write(PNG)
            out.write(ByteArray(Images.MAX_BYTES.toInt()))
        }
        val refused = runCatching { images.keep(big, "a") }.exceptionOrNull()
        assertTrue(refused is ImageRefused)
        assertEquals(Images.TOO_LARGE, refused?.message)
        assertTrue("nothing left behind", images.waiting().isEmpty())
    }

    @Test
    fun theCacheKeepsItsBoundDroppingTheOldestAndNeverWhatWaits() = runBlocking {
        val images = Images(File(dir, "up"), File(dir, "cache"), bound = 1000)
        val waiting = images.newTemp().apply { writeBytes(PNG + ByteArray(5000)) }
        val kept = images.keep(waiting, "a")
        for (i in 1..3) {
            val file = images.cached("$i.png") { it.writeBytes(ByteArray(400)) }
            file.setLastModified(1_000L * i)
        }
        assertNull("the oldest went", images.file("1.png"))
        assertNotNull(images.file("2.png"))
        assertNotNull(images.file("3.png"))
        assertNotNull(images.file(kept.name))
        assertEquals(listOf(kept.name), images.waiting())
    }

    @Test
    fun theLineReadsBackAsAnImage() {
        val kept = Kept("0".repeat(32) + ".png", "photo")
        val refs = Description.refs("Some text\n" + kept.line)
        assertEquals(kept.name, refs.single().file)
        assertEquals("photo", refs.single().name)
        assertEquals(listOf(kept.line), Description.fields("Some text\n" + kept.line).second)
    }

    private fun answer(chain: Interceptor.Chain, code: Int, message: String): Response = Response.Builder()
        .request(chain.request())
        .protocol(Protocol.HTTP_1_1)
        .code(code)
        .message(message)
        .body("""{"error":"$message","protocol":1}""".toResponseBody("application/json".toMediaType()))
        .build()
}
