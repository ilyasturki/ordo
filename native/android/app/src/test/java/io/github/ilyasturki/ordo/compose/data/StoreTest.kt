package io.github.ilyasturki.ordo.compose.data

import kotlinx.serialization.json.JsonObject
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Before
import org.junit.Test
import java.io.File
import java.io.IOException
import java.nio.file.Files

class StoreTest {
    private lateinit var dir: File

    @Before
    fun setUp() {
        dir = Files.createTempDirectory("ordo-store").toFile()
    }

    @After
    fun tearDown() {
        dir.deleteRecursively()
    }

    private fun queued(seq: Long) = Queued(seq, "q$seq", "p", JsonObject(emptyMap()))

    @Test
    fun savesLandInTheOrderTheyWereMade() {
        val store = Store(dir)
        for (n in 1..40L) store.saveQueue((1..n).map(::queued))
        assertEquals(40, store.queue().size)
    }

    @Test
    fun aReadThroughAnotherStoreWaitsForTheWritesBeforeIt() {
        Store(dir).saveSession(Session("http://phone.test", "token"))
        assertEquals("token", Store(dir).session()?.token)
        Store(dir).saveSession(null)
        assertEquals(null, Store(dir).session())
    }

    @Test
    fun aWipeComesAfterTheSavesBeforeIt() {
        val store = Store(dir)
        store.saveProject(StoredProject("p", "P", ""), "{}")
        store.saveQueue(listOf(queued(1)))
        store.wipe()
        store.saveQueue(listOf(queued(2)))
        assertTrue(store.projects().isEmpty())
        assertEquals(listOf(2L), store.queue().map(Queued::seq))
    }

    @Test
    fun aWriteThatFailedSurfacesAtTheNextCall() {
        val store = Store(dir)
        File(dir, "projects").deleteRecursively()
        store.saveProject(StoredProject("p", "P", ""), "{}")
        try {
            store.queue()
            fail("the failed save went unreported")
        } catch (e: IOException) {
            assertFalse(File(dir, "projects/p.json").exists())
        }
        assertTrue("reported once", store.queue().isEmpty())
    }
}
