package io.github.ilyasturki.ordo.compose.core

import io.github.ilyasturki.ordo.compose.model.Answer
import io.github.ilyasturki.ordo.compose.model.json
import io.github.ilyasturki.ordo.core.ordocore.Ordocore
import kotlinx.serialization.json.JsonObject

// The calls of internal/core, JSON strings in and out; not safe for concurrent use.
interface CoreApi {
    fun open(raw: String): String
    fun perform(pid: String, op: String): String
    fun replay(pid: String, op: String): String
    fun project(pid: String): String
    fun close(pid: String)
}

class GoCore : CoreApi {
    private val core = Ordocore.new_()

    override fun open(raw: String): String = core.open(raw)
    override fun perform(pid: String, op: String): String = core.perform(pid, op)
    override fun replay(pid: String, op: String): String = core.replay(pid, op)
    override fun project(pid: String): String = core.project(pid)
    override fun close(pid: String) = core.close(pid)
}

fun CoreApi.perform(pid: String, op: JsonObject): Answer =
    json.decodeFromString(Answer.serializer(), perform(pid, op.toString()))

fun CoreApi.replay(pid: String, op: JsonObject): Answer =
    json.decodeFromString(Answer.serializer(), replay(pid, op.toString()))
