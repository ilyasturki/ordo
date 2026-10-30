package io.github.ilyasturki.ordo.compose.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import io.github.ilyasturki.ordo.compose.model.Estimates
import io.github.ilyasturki.ordo.compose.model.Header
import io.github.ilyasturki.ordo.compose.model.Hidden
import io.github.ilyasturki.ordo.compose.model.Priority
import io.github.ilyasturki.ordo.compose.model.Row
import io.github.ilyasturki.ordo.compose.model.Status
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.marks

@Composable
fun CategoryHeader(header: Header, onClick: () -> Unit) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Text(
        buildAnnotatedString {
            if (header.parentName != "") {
                withStyle(SpanStyle(color = muted)) { append(header.parentName + "  ›  ") }
            }
            append(header.category.name)
        },
        style = MaterialTheme.typography.titleSmall,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(start = 20.dp, end = 16.dp, top = 20.dp, bottom = 8.dp),
    )
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
fun TaskRow(row: Row, child: Boolean, onCycle: () -> Unit, onOpen: () -> Unit, modifier: Modifier = Modifier) {
    val t = row.task
    if (t.separator) {
        SeparatorRow(t.title, child, onOpen, modifier)
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
            Text(
                t.title,
                style = MaterialTheme.typography.bodyLarge,
                color = if (done) colors.onSurfaceVariant else colors.onSurface,
                textDecoration = if (done) TextDecoration.LineThrough else null,
            )
            if (row.preview != "") {
                Text(
                    row.preview,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.onSurfaceVariant,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            val pri = t.priority.takeIf { it != "" && it != "medium" }
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
        Spacer(Modifier.width(4.dp))
        Text(label, style = MaterialTheme.typography.labelMedium, color = tint)
    }
}

@Composable
private fun SeparatorRow(label: String, child: Boolean, onOpen: () -> Unit, modifier: Modifier) {
    Row(
        modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen)
            .padding(start = if (child) 56.dp else 20.dp, end = 16.dp)
            .height(40.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (label != "") {
            Text(
                label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.width(12.dp))
        }
        HorizontalDivider(Modifier.weight(1f), color = MaterialTheme.colorScheme.outlineVariant)
    }
}

@Composable
fun NoTasksLine() {
    Text(
        "No tasks",
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(start = 56.dp, end = 16.dp, top = 4.dp, bottom = 12.dp),
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
