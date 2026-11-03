package io.github.ilyasturki.ordo.compose.ui.settings

import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import io.github.ilyasturki.ordo.compose.ui.Icons

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChangelogScreen(back: () -> Unit) {
    val context = LocalContext.current
    val lines = remember { parseChangelog(context.assets.open("CHANGELOG.md").bufferedReader().use { it.readText() }) }
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Changelog") },
                navigationIcon = { IconButton(onClick = back) { Icon(Icons.Back, "Back") } },
            )
        },
    ) { pad ->
        LazyColumn(Modifier.padding(pad).testTag("changelog"), contentPadding = PaddingValues(horizontal = 24.dp, vertical = 8.dp)) {
            items(lines) { line ->
                when (line.kind) {
                    ChangelogKind.Release -> Row(Modifier.padding(top = 24.dp, bottom = 4.dp)) {
                        Text(line.text, style = MaterialTheme.typography.titleLarge)
                        if (line.date != "") {
                            Text(
                                line.date,
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(start = 12.dp, top = 6.dp),
                            )
                        }
                    }
                    ChangelogKind.Section -> Text(
                        line.text,
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.padding(top = 12.dp, bottom = 4.dp),
                    )
                    ChangelogKind.Bullet -> Row(Modifier.padding(vertical = 4.dp)) {
                        Text("•", style = MaterialTheme.typography.bodyMedium, modifier = Modifier.width(16.dp))
                        Text(withCode(line.text), style = MaterialTheme.typography.bodyMedium)
                    }
                    ChangelogKind.Paragraph -> Text(withCode(line.text), style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(vertical = 4.dp))
                }
            }
        }
    }
}

@Composable
private fun withCode(text: String): AnnotatedString {
    val background = MaterialTheme.colorScheme.surfaceVariant
    return buildAnnotatedString {
        text.split('`').forEachIndexed { i, part ->
            if (i % 2 == 1) withStyle(SpanStyle(fontFamily = FontFamily.Monospace, background = background)) { append(part) } else append(part)
        }
    }
}
