package io.github.ilyasturki.ordo.compose.ui.home

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.PagerState
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryScrollableTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.TabIndicatorScope
import androidx.compose.material3.TabRowDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.lerp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.LocalSnackbar
import io.github.ilyasturki.ordo.compose.ui.NavKey
import io.github.ilyasturki.ordo.compose.ui.SettingsKey
import io.github.ilyasturki.ordo.compose.ui.SignInKey
import io.github.ilyasturki.ordo.compose.ui.TaskKey
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch
import kotlin.math.abs
import kotlin.math.max

private sealed interface HomeSheet {
    data class Add(val pid: String) : HomeSheet
    data class Filters(val pid: String) : HomeSheet
    data class ProjectMenu(val pid: String) : HomeSheet
    data class CategoryMenu(val pid: String, val cat: String) : HomeSheet
    data class NewCategory(val pid: String, val parent: String) : HomeSheet
    data class NewSeparator(val pid: String, val cat: String) : HomeSheet
    data object NewProject : HomeSheet
    data object Sync : HomeSheet
}

private sealed interface Confirm {
    data class DeleteProject(val pid: String) : Confirm
    data class DeleteCategory(val pid: String, val cat: String) : Confirm
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(open: (NavKey) -> Unit) {
    val engine = LocalEngine.current
    val state by engine.state.collectAsStateWithLifecycle()
    val projects = state.projects
    val scope = rememberCoroutineScope()
    val feedback = rememberFeedback()
    var sheet by remember { mutableStateOf<HomeSheet?>(null) }
    var confirm by remember { mutableStateOf<Confirm?>(null) }
    val uis = remember { mutableStateMapOf<String, ListUi>() }
    val lastCat = remember { mutableStateMapOf<String, String>() }

    val pager = rememberPagerState(initialPage = projects.indexOfFirst { it.id == state.prefs.current }.coerceAtLeast(0)) { projects.size }
    val current = projects.getOrNull(pager.currentPage.coerceAtMost(projects.lastIndex))
    val latest by rememberUpdatedState(projects)
    fun filterOf(e: Entry): Filter = (state.prefs.filters[e.id] ?: Filter.NONE).pruned(e.project)

    LaunchedEffect(projects.map { it.id }) {
        val i = projects.indexOfFirst { it.id == state.prefs.current }
        if (i >= 0 && i != pager.currentPage) pager.scrollToPage(i)
    }
    LaunchedEffect(pager) {
        snapshotFlow { pager.settledPage }.distinctUntilChanged().drop(1).collect { page ->
            val id = latest.getOrNull(page)?.id ?: return@collect
            uis.keys.filter { it != id }.forEach(uis::remove)
            if (id != engine.state.value.prefs.current) {
                feedback.tick()
                engine.setCurrent(id)
            }
        }
    }

    Scaffold(
        topBar = {
            Column(Modifier.statusBarsPadding()) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    PrimaryScrollableTabRow(
                        selectedTabIndex = pager.currentPage.coerceIn(0, projects.lastIndex.coerceAtLeast(0)),
                        modifier = Modifier.weight(1f),
                        containerColor = Color.Transparent,
                        edgePadding = 4.dp,
                        divider = {},
                        indicator = { PagerIndicator(pager, projects.size) },
                        minTabWidth = 0.dp,
                    ) {
                        projects.forEachIndexed { i, p ->
                            ProjectTab(
                                p.name,
                                selected = i == pager.currentPage,
                                onClick = {
                                    if (i == pager.currentPage) sheet = HomeSheet.ProjectMenu(p.id)
                                    else scope.launch { pager.animateScrollToPage(i) }
                                },
                                onLongClick = { sheet = HomeSheet.ProjectMenu(p.id) },
                            )
                        }
                        NewProjectTab { sheet = HomeSheet.NewProject }
                    }
                    if (state.trouble != null) {
                        IconButton(onClick = { sheet = HomeSheet.Sync }) {
                            Icon(Icons.CloudOff, "Not synced", tint = MaterialTheme.colorScheme.error)
                        }
                    }
                    if (current != null) {
                        val active = filterOf(current).active
                        IconButton(onClick = { sheet = HomeSheet.Filters(current.id) }) {
                            Icon(Icons.Filter, if (active) "Filter, on" else "Filter", tint = if (active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    IconButton(onClick = { open(SettingsKey) }) {
                        Icon(Icons.Account, "Account and settings", Modifier.size(28.dp), tint = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                HorizontalDivider()
            }
        },
        floatingActionButton = {
            if (current != null && current.project.categories.isNotEmpty()) {
                FloatingActionButton(onClick = { sheet = HomeSheet.Add(current.id) }) { Icon(Icons.Add, "Add a task") }
            }
        },
        snackbarHost = { SnackbarHost(LocalSnackbar.current) },
    ) { pad ->
        if (projects.isEmpty()) {
            Column(
                Modifier.fillMaxSize().padding(pad).padding(32.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                val waiting = state.lastSync == 0L
                Text(
                    when {
                        waiting && state.error != "" -> "Can't load your projects yet: ${state.error}"
                        waiting -> "Loading your projects…"
                        else -> "No projects yet"
                    },
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                )
                if (waiting && state.error != "") {
                    FilledTonalButton(onClick = { scope.launch { engine.sync() } }, modifier = Modifier.padding(top = 16.dp)) { Text("Retry") }
                } else if (!waiting) {
                    FilledTonalButton(onClick = { sheet = HomeSheet.NewProject }, modifier = Modifier.padding(top = 16.dp)) { Text("New project") }
                }
            }
        } else {
            HorizontalPager(
                state = pager,
                key = { latest.getOrNull(it)?.id ?: it },
                modifier = Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()),
                beyondViewportPageCount = 1,
            ) { page ->
                val entry = projects.getOrNull(page) ?: return@HorizontalPager
                ProjectList(
                    entry = entry,
                    filter = filterOf(entry),
                    ui = uis[entry.id] ?: ListUi(),
                    onUi = { uis[entry.id] = it },
                    onOpen = { open(TaskKey(entry.id, it)) },
                    onHeader = { sheet = HomeSheet.CategoryMenu(entry.id, it.id) },
                    onNewCategory = { sheet = HomeSheet.NewCategory(entry.id, "") },
                )
            }
        }
    }

    fun entry(pid: String): Entry? = projects.firstOrNull { it.id == pid }

    when (val s = sheet) {
        null -> {}
        is HomeSheet.Add -> entry(s.pid)?.let { e ->
            AddSheet(e, lastCat[e.id].orEmpty(), onDismiss = { sheet = null }) { cat -> lastCat[e.id] = cat }
        }
        is HomeSheet.Filters -> entry(s.pid)?.let { e ->
            FilterSheet(e.project, filterOf(e), onChange = { f ->
                uis.remove(e.id)
                scope.launch { engine.setFilter(e.id, f) }
            }, onDismiss = { sheet = null })
        }
        is HomeSheet.ProjectMenu -> entry(s.pid)?.let { e ->
            NameSheet(e.name, "Project name", "Rename", onDismiss = { sheet = null }, onDone = { name ->
                name == e.name || engine.edit(e.id, Ops.projectRename(name), quiet = true).applied
            }) {
                SheetAction(Icons.NewFolder, "New category") { sheet = HomeSheet.NewCategory(e.id, "") }
                SheetAction(Icons.Delete, "Delete project", danger = true) {
                    sheet = null
                    confirm = Confirm.DeleteProject(e.id)
                }
            }
        }
        is HomeSheet.CategoryMenu -> entry(s.pid)?.let { e ->
            val cat = e.project.category(s.cat) ?: return@let
            NameSheet(cat.name, "Category name", "Rename", onDismiss = { sheet = null }, onDone = { name ->
                name == cat.name || engine.edit(e.id, Ops.categoryRename(cat.id, name), quiet = true).applied
            }) {
                if (cat.parentId == "") {
                    SheetAction(Icons.NewFolder, "New sub-category") { sheet = HomeSheet.NewCategory(e.id, cat.id) }
                }
                SheetAction(Icons.Separator, "Add a separator") { sheet = HomeSheet.NewSeparator(e.id, cat.id) }
                SheetAction(Icons.Delete, "Delete category", danger = true) {
                    sheet = null
                    confirm = Confirm.DeleteCategory(e.id, cat.id)
                }
            }
        }
        is HomeSheet.NewCategory -> NameSheet("", if (s.parent == "") "New category" else "New sub-category", "Add", onDismiss = { sheet = null }, onDone = { name ->
            engine.edit(s.pid, Ops.categoryAdd(name, s.parent), quiet = true).applied.also { if (it) feedback.commit() }
        })
        is HomeSheet.NewSeparator -> NameSheet("", "Separator label", "Add", onDismiss = { sheet = null }, onDone = { label ->
            val cat = entry(s.pid)?.project?.category(s.cat)
            engine.edit(s.pid, Ops.separator(s.cat, cat?.tasks?.lastOrNull { it.parentId == "" }?.id.orEmpty(), label), quiet = true).applied
        })
        HomeSheet.NewProject -> NameSheet("", "New project", "Add", onDismiss = { sheet = null }, onDone = { name ->
            val pid = engine.createProject(name)
            if (pid != null) {
                feedback.commit()
                engine.setCurrent(pid)
            }
            pid != null
        })
        HomeSheet.Sync -> SyncSheet(
            state,
            onRetry = { scope.launch { engine.sync() } },
            onSignIn = {
                sheet = null
                open(SignInKey)
            },
            onDismiss = { sheet = null },
        )
    }

    when (val c = confirm) {
        null -> {}
        is Confirm.DeleteProject -> entry(c.pid)?.let { e ->
            val n = e.project.categories.sumOf { cat -> cat.tasks.count { !it.separator } }
            ConfirmDialog("Delete “${e.name}”?", "Its ${e.project.categories.size} categories and $n tasks go too. This can't be undone.", "Delete", { confirm = null }) {
                scope.launch { engine.edit(e.id, Ops.projectDelete()) }
            }
        }
        is Confirm.DeleteCategory -> entry(c.pid)?.let { e ->
            val cat = e.project.category(c.cat) ?: return@let
            ConfirmDialog("Delete “${cat.name}”?", "Its ${countTasks(e.project, cat)} tasks and sub-categories go too. This can't be undone.", "Delete", { confirm = null }) {
                scope.launch { engine.edit(e.id, Ops.categoryDelete(cat.id)) }
            }
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ProjectTab(name: String, selected: Boolean, onClick: () -> Unit, onLongClick: () -> Unit) {
    val color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
    Box(
        Modifier
            .height(48.dp)
            .combinedClickable(onClick = onClick, onLongClick = onLongClick, role = Role.Tab)
            .padding(horizontal = 16.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(name, style = MaterialTheme.typography.titleSmall, color = color, maxLines = 1)
    }
}

@Composable
private fun NewProjectTab(onClick: () -> Unit) {
    Box(
        Modifier
            .height(48.dp)
            .combinedClickable(onClick = onClick, role = Role.Button)
            .padding(horizontal = 16.dp),
        contentAlignment = Alignment.Center,
    ) {
        Icon(Icons.Add, "New project", tint = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

// The indicator slides with the pager as the finger drags it, between the two tabs it is over.
@Composable
private fun TabIndicatorScope.PagerIndicator(pager: PagerState, count: Int) {
    TabRowDefaults.PrimaryIndicator(
        Modifier.tabIndicatorLayout { measurable, constraints, positions ->
            if (count == 0 || positions.size < count) return@tabIndicatorLayout layout(0, 0) {}
            val page = pager.currentPage.coerceIn(0, count - 1)
            val fraction = pager.currentPageOffsetFraction
            val next = (if (fraction > 0) page + 1 else page - 1).coerceIn(0, count - 1)
            val f = abs(fraction)
            val a = positions[page]
            val b = positions[next]
            val width: Dp = lerp(a.contentWidth, b.contentWidth, f)
            val start: Dp = lerp(a.left + (a.width - a.contentWidth) / 2, b.left + (b.width - b.contentWidth) / 2, f)
            val w = width.roundToPx()
            val placeable = measurable.measure(constraints.copy(minWidth = w, maxWidth = w))
            // PrimaryScrollableTabRow places this node at half the selected tab's width left over by the node's: taken back off here.
            val shift = max(0, (a.width.roundToPx() - constraints.maxWidth) / 2)
            layout(constraints.maxWidth, placeable.height) { placeable.placeRelative(start.roundToPx() - shift, 0) }
        }.testTag("indicator"),
        width = Dp.Unspecified,
    )
}
