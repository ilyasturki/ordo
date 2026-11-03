package io.github.ilyasturki.ordo.compose.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.InlineTextContent
import androidx.compose.foundation.text.appendInlineContent
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.Placeholder
import androidx.compose.ui.text.PlaceholderVerticalAlign
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.em
import io.github.ilyasturki.ordo.compose.model.Estimates
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Hidden
import io.github.ilyasturki.ordo.compose.model.Priority
import io.github.ilyasturki.ordo.compose.model.Row
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.marks

// The chevron is a target of its own: a tap or a long press anywhere else on the header is the header's, and a drag never starts from either.
@Composable
fun CategoryHeader(header: Header, onLongClick: () -> Unit, onFold: (() -> Unit)?, onClick: () -> Unit) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Row(
        Modifier
            .fillMaxWidth()
            .combinedClickable(onClick = onClick, onLongClick = onLongClick)
            .padding(start = 20.dp, end = 4.dp, top = 4.dp)
            .defaultMinSize(minHeight = 48.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            buildAnnotatedString {
                if (header.parentName != "") {
                    withStyle(SpanStyle(color = muted)) { append(header.parentName + "  ›  ") }
                }
                append(header.category.name)
                if (header.category.pinned) pinMark()
            },
            inlineContent = pinContent(muted),
            style = MaterialTheme.typography.titleSmall,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.weight(1f).padding(vertical = 8.dp),
        )
        if (header.folded) {
            Spacer(Modifier.width(8.dp))
            FoldedMarks(header)
        }
        if (onFold != null) {
            FoldButton(header.folded, header.category.name, onFold)
        } else {
            Spacer(Modifier.width(12.dp))
        }
    }
}

fun AnnotatedString.Builder.pinMark() {
    append(" ")
    appendInlineContent(PIN, "pinned")
}

private const val PIN = "pin"

@Composable
fun pinContent(tint: Color): Map<String, InlineTextContent> = mapOf(
    PIN to InlineTextContent(Placeholder(1.em, 1.em, PlaceholderVerticalAlign.TextCenter)) {
        Icon(Icons.Pin, "Pinned", tint = tint, modifier = Modifier.fillMaxSize())
    },
)

@Composable
private fun FoldedMarks(header: Header) {
    val est = Estimates.format(header.category.estimateMinutes)
    Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
        if (header.status != "") {
            Chip(Icons.status(header.status), if (header.total > 0) "${header.done}/${header.total}" else "")
        }
        if (est != "") Chip(Icons.Schedule, est)
    }
}

@Composable
private fun FoldButton(folded: Boolean, name: String, onClick: () -> Unit) {
    IconButton(onClick = onClick) {
        Icon(
            if (folded) Icons.ExpandMore else Icons.ExpandLess,
            if (folded) "Unfold $name" else "Fold $name",
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
fun StatusIcon(status: String, onClick: () -> Unit, modifier: Modifier = Modifier) {
    val active = status == Status.IN_PROGRESS || status == Status.COMPLETED
    IconButton(onClick = onClick, modifier = modifier) {
        Icon(
            Icons.status(status),
            contentDescription = "${Status.label(status)}, change status",
            tint = if (active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun TaskRow(row: Row, child: Boolean, onCycle: () -> Unit, onOpen: () -> Unit, modifier: Modifier = Modifier, onFold: (() -> Unit)? = null) {
    val t = row.task
    if (t.separator) {
        SeparatorRow(t.title, child, onOpen, modifier)
        return
    }
    if (t.note) {
        NoteRow(t.title, onOpen, modifier)
        return
    }
    val colors = MaterialTheme.colorScheme
    val done = t.finished
    Row(
        modifier
            .fillMaxWidth()
            .alpha(if (row.muted) 0.5f else 1f)
            .clickable(onClick = onOpen)
            .padding(start = if (child) 40.dp else 4.dp, end = 12.dp)
            .defaultMinSize(minHeight = 48.dp),
        verticalAlignment = Alignment.Top,
    ) {
        StatusIcon(t.status, onCycle)
        Column(
            Modifier
                .weight(1f)
                .padding(top = 12.dp, bottom = 12.dp, end = 4.dp),
        ) {
            var wrapped by remember(t.title) { mutableStateOf(false) }
            Text(
                t.title,
                style = MaterialTheme.typography.bodyLarge,
                color = if (done) colors.onSurfaceVariant else colors.onSurface,
                textDecoration = if (done) TextDecoration.LineThrough else null,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                onTextLayout = { wrapped = it.lineCount > 1 },
            )
            if (row.preview != "") {
                Text(
                    row.preview,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSurfaceVariant,
                    maxLines = if (wrapped) 1 else 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            val pri = t.priority.takeIf { it != "" }
            val est = Estimates.format(t.estimateMinutes)
            if (pri != null || est != "" || row.childTotal > 0 || row.images > 0) {
                FlowRow(
                    Modifier.padding(top = 6.dp),
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    if (pri != null) {
                        val up = pri == "critical" || pri == "high"
                        Chip(
                            Icons.priority(pri),
                            Priority.label(pri),
                            when {
                                done -> colors.onSurfaceVariant
                                up -> MaterialTheme.marks.up
                                else -> MaterialTheme.marks.down
                            },
                        )
                    }
                    if (est != "") Chip(Icons.Schedule, est)
                    if (row.childTotal > 0) Chip(Icons.Subtask, "${row.childDone}/${row.childTotal}")
                    if (row.images > 0) Chip(Icons.Image, row.images.toString())
                }
            }
        }
        if (t.tagColor != "") {
            Box(Modifier.height(48.dp), contentAlignment = Alignment.Center) { TagDot(t.tagColor, Modifier.alpha(if (done) 0.5f else 1f)) }
        }
        if (onFold != null && row.childTotal > 0) FoldButton(row.folded, "subtasks of ${t.title}", onFold)
    }
}

@Composable
fun TagDot(color: String, modifier: Modifier = Modifier) {
    Box(
        modifier
            .size(10.dp)
            .background(MaterialTheme.marks.tag(color).takeIf { it != Color.Unspecified } ?: MaterialTheme.colorScheme.outline, CircleShape),
    )
}

@Composable
private fun Chip(icon: ImageVector, label: String, tint: Color = MaterialTheme.colorScheme.onSurfaceVariant) {
    Row(
        Modifier
            .height(24.dp)
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, RoundedCornerShape(6.dp))
            .padding(horizontal = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, Modifier.size(14.dp), tint = tint)
        if (label != "") {
            Spacer(Modifier.width(4.dp))
            Text(label, style = MaterialTheme.typography.labelMedium, color = tint)
        }
    }
}

@Composable
private fun SeparatorRow(label: String, child: Boolean, onOpen: () -> Unit, modifier: Modifier) {
    Row(
        modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .then(if (label == "") Modifier.semantics { contentDescription = "Separator" } else Modifier)
            .padding(start = if (child) 56.dp else 20.dp, end = 16.dp)
            .height(48.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (label != "") {
            SeparatorRule(Modifier.width(20.dp))
            Spacer(Modifier.width(10.dp))
            Text(
                label,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.width(10.dp))
        }
        SeparatorRule(Modifier.weight(1f))
    }
}

@Composable
private fun NoteRow(text: String, onOpen: () -> Unit, modifier: Modifier) {
    Text(
        text,
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .defaultMinSize(minHeight = 48.dp)
            .padding(start = 20.dp, end = 16.dp, top = 14.dp, bottom = 14.dp),
    )
}

@Composable
private fun SeparatorRule(modifier: Modifier) {
    Box(modifier.height(3.dp).background(MaterialTheme.colorScheme.onSurface, CircleShape))
}

@Composable
fun NoTasksLine() {
    Text(
        "No tasks",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(start = 52.dp, end = 16.dp, top = 4.dp, bottom = 12.dp),
    )
}

@Composable
fun HiddenLine(line: Hidden, onClick: () -> Unit) {
    Text(
        if (line.peeking) "Hide the ${line.count} filtered out" else "${line.count} hidden by the filter",
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(start = 56.dp, end = 16.dp, top = 12.dp, bottom = 12.dp),
    )
}
