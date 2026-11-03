package io.github.ilyasturki.ordo.compose.ui.home

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import io.github.ilyasturki.ordo.compose.model.Priority
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.marks

// A task's value, the same on the list (24dp, read only) and wherever a tap changes it (32dp, opening its picker).
@Composable
fun MarkChip(
    label: String,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    tag: String = "",
    tint: Color = Color.Unspecified,
    trailing: ImageVector? = null,
    name: String = "",
    onClick: (() -> Unit)? = null,
) {
    val color = if (tint == Color.Unspecified) MaterialTheme.colorScheme.onSurfaceVariant else tint
    val big = onClick != null
    val shape = RoundedCornerShape(if (big) 8.dp else 6.dp)
    val iconSize = if (big) 18.dp else 14.dp
    Row(
        modifier
            .height(if (big) 32.dp else 24.dp)
            .clip(shape)
            .border(1.dp, MaterialTheme.colorScheme.outline, shape)
            .then(
                if (onClick == null) {
                    Modifier
                } else {
                    // Hit testing widens the 32dp target to 48dp.
                    Modifier
                        .clickable(onClick = onClick, role = Role.Button)
                        .semantics {
                            if (name != "") {
                                contentDescription = name
                                stateDescription = label
                            }
                        }
                },
            )
            .padding(start = 6.dp, end = if (trailing != null) 4.dp else if (big) 8.dp else 6.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) Icon(icon, null, Modifier.size(iconSize), tint = color)
        if (tag != "") TagDot(tag, Modifier.padding(horizontal = 2.dp).size(if (big) 12.dp else 10.dp))
        if (label != "") {
            Text(
                label,
                style = if (big) MaterialTheme.typography.labelLarge else MaterialTheme.typography.labelMedium,
                color = color,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f, fill = false),
            )
        }
        if (trailing != null) Icon(trailing, null, Modifier.size(18.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

// Red for up, teal for down, the plain colour for medium and for a finished task's.
@Composable
fun priorityTint(priority: String, finished: Boolean = false): Color = when {
    finished -> MaterialTheme.colorScheme.onSurfaceVariant
    priority == "critical" || priority == "high" -> MaterialTheme.marks.up
    priority == "low" || priority == "trivial" -> MaterialTheme.marks.down
    else -> MaterialTheme.colorScheme.onSurfaceVariant
}

@Composable
fun PriorityChip(priority: String, finished: Boolean = false, modifier: Modifier = Modifier, onClick: (() -> Unit)? = null) {
    MarkChip(Priority.label(priority), modifier, icon = Icons.priority(priority), tint = priorityTint(priority, finished), name = "Priority", onClick = onClick)
}
