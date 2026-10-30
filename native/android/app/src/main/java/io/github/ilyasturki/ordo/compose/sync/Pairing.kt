package io.github.ilyasturki.ordo.compose.sync

import java.net.URI
import java.net.URLDecoder

data class Pairing(val server: String, val code: String)

private val pairPath = Regex("""/pair/([^/?#]+)/?$""")

// The server and code of a pairing link, https://host[/prefix]/pair/<code>, as app/src/lib/api.ts reads it.
fun parsePairing(text: String): Pairing? {
    val raw = text.trim()
    if (raw.isEmpty()) return null
    val uri = try {
        URI(raw)
    } catch (_: Exception) {
        return null
    }
    val scheme = uri.scheme?.lowercase() ?: return null
    if (scheme != "http" && scheme != "https" || uri.host == null) return null
    val path = uri.rawPath.orEmpty()
    val m = pairPath.find(path) ?: return null
    val port = if (uri.port == -1) "" else ":${uri.port}"
    val prefix = path.substring(0, m.range.first)
    return Pairing("$scheme://${uri.host}$port$prefix", URLDecoder.decode(m.groupValues[1], "UTF-8"))
}

// The server a typed address names; a bare host gets https. Null for a pairing link, or while it can't be one yet.
fun parseServer(text: String): String? {
    val raw = text.trim().trimEnd('/')
    if (raw.isEmpty() || parsePairing(raw) != null) return null
    val uri = try {
        URI(if ("://" in raw) raw else "https://$raw")
    } catch (_: Exception) {
        return null
    }
    val scheme = uri.scheme?.lowercase() ?: return null
    val host = uri.host ?: return null
    if (scheme != "http" && scheme != "https" || ('.' !in host && host != "localhost")) return null
    val port = if (uri.port == -1) "" else ":${uri.port}"
    return "$scheme://$host$port${uri.rawPath.orEmpty()}"
}
