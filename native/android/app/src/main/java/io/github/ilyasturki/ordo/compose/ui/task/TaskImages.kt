package io.github.ilyasturki.ordo.compose.ui.task

import android.content.ContentResolver
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.ImageDecoder
import android.net.Uri
import android.os.Build
import android.provider.OpenableColumns
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.data.ImageRefused
import io.github.ilyasturki.ordo.compose.data.Images
import io.github.ilyasturki.ordo.compose.data.Kept
import io.github.ilyasturki.ordo.compose.model.ImageRef
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.File
import kotlin.math.max
import kotlin.math.min
import kotlin.math.roundToInt

// A picked or photographed image as a file waiting for the server. A phone's HEIC or other format
// the server refuses becomes a jpeg.
fun Images.take(resolver: ContentResolver, uri: Uri, alt: String = displayName(resolver, uri)): Kept {
    val input = resolver.openInputStream(uri) ?: throw ImageRefused(Images.NOT_AN_IMAGE)
    val temp = input.use(::copy)
    if (Images.extensionOf(temp) != null) return keep(temp, alt)
    val jpeg = newTemp()
    try {
        val bitmap = decode(temp, MAX_SIDE, MAX_SIDE, cover = false) ?: throw ImageRefused(Images.NOT_AN_IMAGE)
        jpeg.outputStream().use { out ->
            bitmap.compress(Bitmap.CompressFormat.JPEG, 90, out)
            out.fd.sync()
        }
        return keep(jpeg, alt)
    } finally {
        temp.delete()
        jpeg.delete()
    }
}

private fun displayName(resolver: ContentResolver, uri: Uri): String =
    resolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { c ->
        if (c.moveToFirst()) c.getString(0) else null
    } ?: uri.lastPathSegment.orEmpty()

// At the size it shows, the system applying the photo's rotation: a phone's photo decoded whole is
// 50 MB. Cover fills the box and may overflow it, otherwise it fits inside.
fun decode(file: File, width: Int, height: Int, cover: Boolean): Bitmap? = try {
    fun scale(w: Int, h: Int): Float {
        val x = width.toFloat() / w
        val y = height.toFloat() / h
        return if (cover) max(x, y) else min(x, y)
    }
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
        ImageDecoder.decodeBitmap(ImageDecoder.createSource(file)) { decoder, info, _ ->
            decoder.allocator = ImageDecoder.ALLOCATOR_SOFTWARE
            val s = scale(info.size.width, info.size.height)
            if (s < 1f) decoder.setTargetSize((info.size.width * s).roundToInt().coerceAtLeast(1), (info.size.height * s).roundToInt().coerceAtLeast(1))
        }
    } else {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(file.path, bounds)
        var sample = 1
        val s = scale(bounds.outWidth.coerceAtLeast(1), bounds.outHeight.coerceAtLeast(1))
        while (s * sample * 2 <= 1f) sample *= 2
        BitmapFactory.decodeFile(file.path, BitmapFactory.Options().apply { inSampleSize = sample })
    }
} catch (_: Exception) {
    null
}

// The image once it's on the phone and decoded, asked for again after each sync while it isn't.
@Composable
fun rememberImage(name: String, width: Int, height: Int, cover: Boolean): ImageBitmap? {
    val engine = LocalEngine.current
    val synced = engine.state.collectAsStateWithLifecycle().value.lastSync
    var bitmap by remember(name, width, height) { mutableStateOf<ImageBitmap?>(null) }
    LaunchedEffect(name, width, height, if (bitmap == null) synced else 0L) {
        if (bitmap != null) return@LaunchedEffect
        val file = engine.image(name) ?: return@LaunchedEffect
        bitmap = withContext(Dispatchers.IO) { decode(file, width, height, cover)?.asImageBitmap() }
    }
    return bitmap
}

@Composable
fun Thumbnails(refs: List<ImageRef>, onOpen: (Int) -> Unit, onRemove: (ImageRef) -> Unit) {
    val stored = refs.filter { it.file != null }
    Row(
        Modifier
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        for (ref in refs) {
            Thumbnail(ref, onOpen = stored.indexOf(ref).takeIf { it >= 0 }?.let { i -> { onOpen(i) } }, onRemove = { onRemove(ref) })
        }
    }
}

@Composable
private fun Thumbnail(ref: ImageRef, onOpen: (() -> Unit)?, onRemove: () -> Unit) {
    val side = with(LocalDensity.current) { THUMB.roundToPx() }
    val name = ref.name.ifEmpty { "image" }
    val bitmap = ref.file?.let { rememberImage(it, side, side, cover = true) }
    Box(
        Modifier
            .size(THUMB)
            .clip(RoundedCornerShape(12.dp))
            .background(MaterialTheme.colorScheme.surfaceContainerHigh)
            .then(if (onOpen != null) Modifier.clickable(onClick = onOpen) else Modifier)
            .semantics { contentDescription = name },
        contentAlignment = Alignment.Center,
    ) {
        if (bitmap != null) {
            Image(bitmap, null, Modifier.fillMaxSize(), contentScale = ContentScale.Crop)
        } else {
            Icon(Icons.Image, null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Box(
            Modifier
                .align(Alignment.TopEnd)
                .size(40.dp)
                .clickable(onClick = onRemove)
                .semantics { contentDescription = "Remove $name" },
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Close,
                null,
                Modifier
                    .background(MaterialTheme.colorScheme.inverseSurface.copy(alpha = 0.7f), CircleShape)
                    .padding(2.dp)
                    .size(16.dp),
                tint = MaterialTheme.colorScheme.inverseOnSurface,
            )
        }
    }
}

private val THUMB = 96.dp

// Keeps a converted phone photo well under the server's 25 MB.
private const val MAX_SIDE = 4096
