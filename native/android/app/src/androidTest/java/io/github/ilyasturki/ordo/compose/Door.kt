package io.github.ilyasturki.ordo.compose

import io.github.ilyasturki.ordo.compose.core.CoreApi
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

// The core with a door on its edits: shut, an edit waits there, as on a phone slow to write.
class Door(private val core: CoreApi) : CoreApi by core {
    @Volatile
    private var latch: CountDownLatch? = null
    val knocks = AtomicInteger()

    fun shut() {
        latch = CountDownLatch(1)
    }

    fun open() {
        latch?.countDown()
        latch = null
    }

    override fun perform(pid: String, op: String): String {
        latch?.let {
            knocks.incrementAndGet()
            it.await(30, TimeUnit.SECONDS)
        }
        return core.perform(pid, op)
    }
}
