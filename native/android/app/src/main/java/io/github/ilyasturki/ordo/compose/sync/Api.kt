package io.github.ilyasturki.ordo.compose.sync

import io.github.ilyasturki.ordo.compose.model.json
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.Call
import okhttp3.Callback
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.sse.EventSource
import okhttp3.sse.EventSourceListener
import okhttp3.sse.EventSources
import java.io.File
import java.io.IOException
import java.net.URLEncoder
import java.util.concurrent.TimeUnit
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

const val PROTOCOL = 1

// The server answered, with its HTTP status and a message for the user.
class ServerError(val status: Int, message: String) : Exception(message)

// The request never got an answer.
class Offline(cause: Throwable) : Exception("Can't reach ordo serve", cause)

@Serializable
data class SyncResult(val id: String, val status: String, val msg: String = "", val theirs: String? = null)

@Serializable
data class ProjectState(
    val id: String,
    val name: String = "",
    val stamp: String = "",
    val data: JsonElement? = null,
    val error: String? = null,
)

@Serializable
data class SyncAnswer(val protocol: Int, val results: List<SyncResult> = emptyList(), val projects: List<ProjectState> = emptyList())

@Serializable
data class Hello(@SerialName("has_passphrase") val hasPassphrase: Boolean = false)

@Serializable
data class DeviceInfo(
    val id: String,
    val name: String,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("last_seen") val lastSeen: String = "",
    val from: String = "",
    val also: List<String> = emptyList(),
    val self: Boolean = false,
)

@Serializable
data class Devices(val devices: List<DeviceInfo> = emptyList(), @SerialName("has_passphrase") val hasPassphrase: Boolean = false)

@Serializable
data class PairCode(val link: String, val qr: String, val minutes: Int)

@Serializable
private data class RevokeAnswer(val self: Boolean = false)

@Serializable
private data class TokenAnswer(val token: String)

@Serializable
private data class ErrorAnswer(val error: String = "")

private val jsonType = "application/json".toMediaType()
private val bytesType = "application/octet-stream".toMediaType()

// /api of docs/sync.md, with the device token as a bearer.
class Api(val server: String, private val token: String, private val http: OkHttpClient = client) {
    // An image takes longer than an answer on a slow line, though a stall still fails.
    private val transfers: OkHttpClient by lazy { http.newBuilder().callTimeout(120, TimeUnit.SECONDS).build() }

    private fun request(path: String): Request.Builder {
        val b = Request.Builder().url(server.trimEnd('/') + path)
        if (token != "") b.header("Authorization", "Bearer $token")
        return b
    }

    private suspend fun <T> call(req: Request, out: KSerializer<T>, via: OkHttpClient = http): T {
        val res = try {
            via.newCall(req).await()
        } catch (e: IOException) {
            throw Offline(e)
        }
        res.use {
            val body = try {
                it.body.string()
            } catch (e: IOException) {
                throw Offline(e)
            }
            if (!it.isSuccessful) {
                val msg = runCatching { json.decodeFromString(ErrorAnswer.serializer(), body).error }.getOrNull()
                throw ServerError(it.code, msg?.takeIf(String::isNotEmpty) ?: "ordo serve answered ${it.code}")
            }
            if (it.code == 204 || body.isEmpty()) {
                @Suppress("UNCHECKED_CAST")
                return Unit as T
            }
            return try {
                json.decodeFromString(out, body)
            } catch (e: Exception) {
                throw ServerError(it.code, "ordo serve answered something this app can't read")
            }
        }
    }

    private suspend fun <T> post(path: String, body: JsonObject, out: KSerializer<T>): T =
        call(request(path).post(body.toString().toRequestBody(jsonType)).build(), out)

    suspend fun pair(code: String, name: String): String =
        post("/api/pair", buildJsonObject { put("token", code); put("name", name) }, TokenAnswer.serializer()).token

    suspend fun sync(ops: List<JsonObject>, have: Map<String, String>): SyncAnswer = post(
        "/api/sync",
        buildJsonObject {
            put("protocol", PROTOCOL)
            put("ops", kotlinx.serialization.json.JsonArray(ops))
            put("have", JsonObject(have.mapValues { JsonPrimitive(it.value) }))
        },
        SyncAnswer.serializer(),
    )

    suspend fun logout() {
        post("/api/logout", JsonObject(emptyMap()), Unit.serializer())
    }

    suspend fun hello(): Hello = call(request("/api/hello").build(), Hello.serializer())

    suspend fun login(passphrase: String, name: String): String =
        post("/api/login", buildJsonObject { put("passphrase", passphrase); put("name", name) }, TokenAnswer.serializer()).token

    suspend fun devices(): Devices = call(request("/api/devices").build(), Devices.serializer())

    suspend fun pairDevice(passphrase: String): PairCode =
        post("/api/devices/pair", buildJsonObject { put("passphrase", passphrase) }, PairCode.serializer())

    // True when the device asking is one of those it signed out.
    suspend fun revoke(id: String): Boolean =
        post("/api/devices/${URLEncoder.encode(id, "UTF-8")}/revoke", JsonObject(emptyMap()), RevokeAnswer.serializer()).self

    suspend fun putImage(name: String, file: File) {
        call(request("/api/attachments/$name").put(file.asRequestBody(bytesType)).build(), Unit.serializer(), transfers)
    }

    suspend fun image(name: String, to: File) {
        val res = try {
            transfers.newCall(request("/api/attachments/$name").build()).await()
        } catch (e: IOException) {
            throw Offline(e)
        }
        res.use {
            if (!it.isSuccessful) throw ServerError(it.code, "ordo serve answered ${it.code}")
            try {
                to.outputStream().use { out -> it.body.byteStream().copyTo(out) }
            } catch (e: IOException) {
                throw Offline(e)
            }
        }
    }

    // Calls changed with a project's ID each time ordo serve sees its file change, until cancelled.
    fun listen(changed: (String) -> Unit, closed: () -> Unit): EventSource =
        EventSources.createFactory(events).newEventSource(
            request("/api/events").header("Accept", "text/event-stream").build(),
            object : EventSourceListener() {
                override fun onEvent(eventSource: EventSource, id: String?, type: String?, data: String) {
                    if (type == "change") changed(data)
                }

                override fun onClosed(eventSource: EventSource) = closed()

                override fun onFailure(eventSource: EventSource, t: Throwable?, response: Response?) = closed()
            },
        )

    companion object {
        val client: OkHttpClient = OkHttpClient.Builder()
            .callTimeout(20, TimeUnit.SECONDS)
            .build()

        // The stream sends a comment every 25 seconds: a minute of silence is a dead connection.
        private val events: OkHttpClient = client.newBuilder()
            .callTimeout(0, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .build()
    }
}

private suspend fun Call.await(): Response = suspendCancellableCoroutine { cont ->
    enqueue(object : Callback {
        override fun onFailure(call: Call, e: IOException) {
            if (!cont.isCancelled) cont.resumeWithException(e)
        }

        override fun onResponse(call: Call, response: Response) {
            cont.resume(response) { _, r, _ -> r.close() }
        }
    })
    cont.invokeOnCancellation { cancel() }
}

fun ProjectState.raw(): String? = data?.toString()

fun SyncResult.isProblem(): Boolean = status != "applied" && status != "unchanged"

internal fun JsonObject.withIds(qid: String, pid: String): JsonObject =
    JsonObject(this + mapOf("id" to JsonPrimitive(qid), "project" to JsonPrimitive(pid)))

internal val JsonObject.action: String get() = this["action"]?.jsonPrimitive?.content.orEmpty()
