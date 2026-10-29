package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.core.CoreApi
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.io.BufferedReader
import java.io.BufferedWriter
import java.io.File
import java.net.HttpURLConnection
import java.net.ServerSocket
import java.net.URI
import java.nio.file.Files
import java.util.concurrent.TimeUnit

private val repo = File(System.getProperty("ordo.repo") ?: error("run through gradle: ordo.repo is unset"))

private val bin: File by lazy {
    val dir = Files.createTempDirectory("ordo-native-test").toFile()
    go(repo, "build", "-o", File(dir, "ordo").path, "./cmd/ordo")
    go(File(repo, "native/core"), "build", "-o", File(dir, "corepipe").path, "./cmd/corepipe")
    dir
}

private fun go(dir: File, vararg args: String) {
    val p = ProcessBuilder(listOf("go") + args).directory(dir).redirectErrorStream(true).start()
    val out = p.inputStream.bufferedReader().readText()
    check(p.waitFor() == 0) { "go ${args.joinToString(" ")}: $out" }
}

@Serializable
private data class Call(val call: String, val pid: String = "", val arg: String = "")

@Serializable
private data class Reply(val out: String = "", val error: String? = null)

// internal/core through cmd/corepipe, since the JVM can't load the Android library.
class PipeCore : CoreApi, AutoCloseable {
    private val process = ProcessBuilder(File(bin, "corepipe").path).redirectError(ProcessBuilder.Redirect.INHERIT).start()
    private val input: BufferedWriter = process.outputStream.bufferedWriter()
    private val output: BufferedReader = process.inputStream.bufferedReader()

    @Synchronized
    private fun call(call: String, pid: String = "", arg: String = ""): String {
        input.write(Json.encodeToString(Call.serializer(), Call(call, pid, arg)))
        input.newLine()
        input.flush()
        val r = Json.decodeFromString(Reply.serializer(), output.readLine() ?: error("corepipe exited"))
        r.error?.let { throw Exception(it) }
        return r.out
    }

    override fun open(raw: String) = call("open", arg = raw)
    override fun perform(pid: String, op: String) = call("perform", pid, op)
    override fun replay(pid: String, op: String) = call("replay", pid, op)
    override fun project(pid: String) = call("project", pid)
    override fun close(pid: String) {
        call("close", pid)
    }

    override fun close() {
        process.destroy()
    }
}

// A throwaway ordo serve seeded with testdata/web/seed.json, and the ordo CLI on its files.
class Server : AutoCloseable {
    val dir: File = Files.createTempDirectory("ordo-serve").toFile()
    val port = ServerSocket(0).use { it.localPort }
    val url = "http://127.0.0.1:$port"
    private val env = mapOf("ORDO_DATA_PATH" to File(dir, "data").path, "ORDO_CONFIG_PATH" to File(dir, "cfg").path)
    private var process: Process? = null

    init {
        ordo("import", File(repo, "testdata/web/seed.json").path, "-q")
        start()
    }

    fun start() {
        val pb = ProcessBuilder(File(bin, "ordo").path, "serve", "--listen", "127.0.0.1:$port")
            .redirectErrorStream(true)
            .redirectOutput(File(dir, "serve.log"))
        pb.environment().putAll(env)
        process = pb.start()
        repeat(100) {
            if (up()) return
            Thread.sleep(50)
        }
        error("ordo serve didn't start: ${File(dir, "serve.log").readText()}")
    }

    fun stop() {
        process?.destroy()
        process?.waitFor(5, TimeUnit.SECONDS)
        process = null
    }

    private fun up(): Boolean = try {
        (URI("$url/api/hello").toURL().openConnection() as HttpURLConnection).run {
            connectTimeout = 200
            responseCode == 200
        }
    } catch (_: Exception) {
        false
    }

    fun ordo(vararg args: String): String {
        val pb = ProcessBuilder(listOf(File(bin, "ordo").path) + args).redirectErrorStream(true)
        pb.environment().putAll(env)
        val p = pb.start()
        val out = p.inputStream.bufferedReader().readText()
        check(p.waitFor() == 0) { "ordo ${args.joinToString(" ")}: $out" }
        return out
    }

    fun pairLink(): String = ordo("serve", "pair", "--base-url", url).lineSequence().first { it.startsWith("http") }.trim()

    fun revoke(device: String) {
        val id = ordo("serve", "devices").lineSequence().first { it.contains(device) }.substringBefore(' ')
        ordo("serve", "revoke", id)
    }

    // The project file as ordo serve holds it.
    fun project(pid: String): String = ordo("export", "-p", pid, "--format", "json")

    override fun close() {
        stop()
        dir.deleteRecursively()
    }
}
