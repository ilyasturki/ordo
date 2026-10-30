package io.github.ilyasturki.ordo.compose

import androidx.compose.ui.test.ExperimentalTestApi
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performScrollToNode
import androidx.test.core.app.ActivityScenario
import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.runner.lifecycle.ActivityLifecycleMonitorRegistry
import androidx.test.runner.lifecycle.Stage
import io.github.ilyasturki.ordo.compose.core.GoCore
import io.github.ilyasturki.ordo.compose.data.Session
import io.github.ilyasturki.ordo.compose.data.Store
import io.github.ilyasturki.ordo.compose.data.StoredProject
import io.github.ilyasturki.ordo.compose.model.Task
import io.github.ilyasturki.ordo.compose.sync.Delays
import io.github.ilyasturki.ordo.compose.sync.SyncEngine
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.job
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Before
import org.junit.Rule
import java.io.File

const val PID = "0rd0-web-fixture"

// The app on the seed project under each name, signed in to a server that never answers, so nothing leaves the phone.
@OptIn(ExperimentalTestApi::class)
abstract class AppFixture(private val names: List<String> = listOf("Ordo")) {
    @get:Rule
    val compose = createEmptyComposeRule()

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private lateinit var scenario: ActivityScenario<MainActivity>
    protected lateinit var engine: SyncEngine

    @Before
    fun setUp() {
        val ins = InstrumentationRegistry.getInstrumentation()
        val raw = ins.context.assets.open("seed.json").bufferedReader().use { it.readText() }
        ins.targetContext.cacheDir.listFiles { f -> f.name.startsWith("ui-") }?.forEach { it.deleteRecursively() }
        val store = Store(File(ins.targetContext.cacheDir, "ui-${System.nanoTime()}"))
        store.saveSession(Session("http://127.0.0.1:9", "token"))
        names.forEachIndexed { i, name ->
            val id = if (i == 0) PID else "$PID-$i"
            store.saveProject(StoredProject(id, name, "stamp"), raw.replace(PID, id))
        }
        engine = SyncEngine(
            store, ::GoCore, scope, "test",
            Delays(sync = 50, poll = 600_000, firstBackoff = 100, maxBackoff = 200, reconnect = 600_000),
        )
        runBlocking {
            engine.boot()
            engine.foreground(true)
        }
        (ins.targetContext.applicationContext as OrdoApp).use(engine)
        scenario = ActivityScenario.launch(MainActivity::class.java)
    }

    // The next test's setUp deletes the files: this one's screens may still be writing until they are gone.
    @After
    fun tearDown() {
        InstrumentationRegistry.getInstrumentation().runOnMainSync {
            val monitor = ActivityLifecycleMonitorRegistry.getInstance()
            for (stage in listOf(Stage.RESUMED, Stage.PAUSED, Stage.STOPPED)) monitor.getActivitiesInStage(stage).forEach { it.finish() }
        }
        scenario.close()
        runBlocking { scope.coroutineContext.job.cancelAndJoin() }
    }

    protected fun task(title: String): Task? =
        engine.state.value.project(PID)?.project?.categories?.flatMap { it.tasks }?.firstOrNull { it.title == title }

    protected fun scrollTo(text: String) {
        compose.waitUntilAtLeastOneExists(androidx.compose.ui.test.hasTestTag("list"), 20_000)
        compose.onNodeWithTag("list").performScrollToNode(hasText(text, substring = true))
    }
}
