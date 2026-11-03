package io.github.ilyasturki.ordo.compose.ui.home

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.PagerState
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryScrollableTabRow
import androidx.compose.material3.SearchBarDefaults
import androidx.compose.material3.TabIndicatorScope
import androidx.compose.material3.TabRowDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.lerp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.ilyasturki.ordo.compose.model.Board
import io.github.ilyasturki.ordo.compose.model.Filter
import io.github.ilyasturki.ordo.compose.model.Ops
import io.github.ilyasturki.ordo.compose.sync.Entry
import io.github.ilyasturki.ordo.compose.ui.CategoriesKey
import io.github.ilyasturki.ordo.compose.ui.Icons
import io.github.ilyasturki.ordo.compose.ui.LocalEngine
import io.github.ilyasturki.ordo.compose.ui.NavKey
import io.github.ilyasturki.ordo.compose.ui.SettingsKey
import io.github.ilyasturki.ordo.compose.ui.SignInKey
import io.github.ilyasturki.ordo.compose.ui.SnackScaffold
import io.github.ilyasturki.ordo.compose.ui.TaskKey
import io.github.ilyasturki.ordo.compose.ui.rememberFeedback
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.launch
import kotlin.math.abs
import kotlin.math.max

private sealed interface HomeSheet {
    data class Add(val pid: String, val cat: String) : HomeSheet
    data class Filters(val pid: String) : HomeSheet
    data class ProjectMenu(val pid: String) : HomeSheet
    data class OfCategory(val pid: String, val sheet: CategorySheet) : HomeSheet
    data object NewProject : HomeSheet
    data object Sync : HomeSheet
}

private sealed interface Confirm {
    data class DeleteProject(val pid: String) : Confirm
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
    // Saved: a task opened from the results comes back to them.
    var searching by rememberSaveable { mutableStateOf(false) }
    var query by rememberSaveable { mutableStateOf("") }
    val closeSearch = {
        searching = false
        query = ""
    }
    BackHandler(searching, closeSearch)
    val uis = remember { mutableStateMapOf<String, ListUi>() }
    val focus = remember { mutableStateMapOf<String, String?>() }

    val pager = rememberPagerState(initialPage = projects.indexOfFirst { it.id == state.prefs.current }.coerceAtLeast(0)) { projects.size }
    val current = projects.getOrNull(pager.currentPage.coerceAtMost(projects.lastIndex))
    val latest by rememberUpdatedState(projects)
    fun filterOf(e: Entry): Filter = (state.views[e.id]?.filter ?: Filter.NONE).pruned(e.project)
    fun targetOf(e: Entry) = Board.target(e.project, filterOf(e), focus[e.id])

    LaunchedEffect(projects.map { it.id }, state.prefs.current) {
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

    SnackScaffold(
        topBar = {
            Column(Modifier.statusBarsPadding()) {
                if (searching && current != null) {
                    ProjectSearchBar(current.name, query, { query = it }, closeSearch)
                } else {
                    TopAppBar(
                        title = {
                            Text(
                                current?.let { targetOf(it) }?.name.orEmpty(),
                                modifier = Modifier.padding(start = 4.dp).testTag("focus"),
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        },
                        actions = {
                            if (state.trouble != null) {
                                IconButton(onClick = { sheet = HomeSheet.Sync }) {
                                    Icon(Icons.CloudOff, "Not synced", tint = MaterialTheme.colorScheme.error)
                                }
                            }
                            if (current != null) {
                                val active = filterOf(current).active
                                IconButton(onClick = { searching = true }) {
                                    Icon(Icons.Search, "Search", tint = MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                                IconButton(onClick = { sheet = HomeSheet.Filters(current.id) }) {
                                    Icon(Icons.Filter, if (active) "Filter, on" else "Filter", tint = if (active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                            }
                            IconButton(onClick = { open(SettingsKey) }) {
                                Icon(Icons.Settings, "Settings", tint = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        },
                    )
                    PrimaryScrollableTabRow(
                        selectedTabIndex = pager.currentPage.coerceIn(0, projects.lastIndex.coerceAtLeast(0)),
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
                                onLongClick = {
                                    scope.launch {
                                        if (i != pager.currentPage) pager.scrollToPage(i)
                                        open(CategoriesKey(p.id))
                                    }
                                },
                            )
                        }
                        NewProjectTab { sheet = HomeSheet.NewProject }
                    }
                }
                HorizontalDivider()
            }
        },
        floatingActionButton = {
            if (!searching && current != null && current.project.categories.isNotEmpty()) {
                FloatingActionButton(onClick = { sheet = HomeSheet.Add(current.id, targetOf(current)?.id.orEmpty()) }) { Icon(Icons.Add, "Add a task") }
            }
        },
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
                modifier = Modifier.fillMaxSize().padding(top = pad.calculateTopPadding()).then(if (searching) Modifier.imePadding() else Modifier),
                beyondViewportPageCount = 1,
                userScrollEnabled = !searching,
            ) { page ->
                val entry = projects.getOrNull(page) ?: return@HorizontalPager
                ProjectList(
                    entry = entry,
                    filter = filterOf(entry),
                    folds = state.views[entry.id]?.folds.orEmpty(),
                    ui = uis[entry.id] ?: ListUi(),
                    onUi = { uis[entry.id] = it },
                    onOpen = { open(TaskKey(entry.id, it)) },
                    onHeader = { sheet = HomeSheet.OfCategory(entry.id, CategorySheet.Menu(it.id)) },
                    onCategories = { open(CategoriesKey(entry.id)) },
                    onNewCategory = { sheet = HomeSheet.OfCategory(entry.id, CategorySheet.New("")) },
                    onFocus = { focus[entry.id] = it },
                    query = query,
                )
            }
        }
    }

    fun entry(pid: String): Entry? = projects.firstOrNull { it.id == pid }

    // As the TUI's add does: what just went into a folded category stays in sight.
    suspend fun reveal(e: Entry, cat: String) {
        engine.setFolded(e.id, cat, false)
        e.project.category(cat)?.parentId?.takeIf { it != "" }?.let { engine.setFolded(e.id, it, false) }
    }

    when (val s = sheet) {
        null -> {}
        is HomeSheet.Add -> entry(s.pid)?.let { e ->
            AddSheet(e, filterOf(e), s.cat, onDismiss = { sheet = null }) { cat -> scope.launch { reveal(e, cat) } }
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
                SheetAction(Icons.NewFolder, "New category") { sheet = HomeSheet.OfCategory(e.id, CategorySheet.New("")) }
                SheetAction(Icons.Delete, "Delete project", danger = true) {
                    sheet = null
                    confirm = Confirm.DeleteProject(e.id)
                }
            }
        }
        is HomeSheet.OfCategory -> entry(s.pid)?.let { e ->
            CategorySheets(e, filterOf(e), s.sheet) { next -> sheet = next?.let { HomeSheet.OfCategory(e.id, it) } }
        }
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
            ConfirmDialog("Delete project “${e.name}”?", "Its ${e.project.categories.size} categories and $n tasks go too. This can't be undone.", "Delete", { confirm = null }) {
                val wasCurrent = current?.id == e.id
                scope.launch {
                    if (engine.edit(e.id, Ops.projectDelete()).applied && wasCurrent) {
                        engine.state.value.projects.firstOrNull()?.let { engine.setCurrent(it.id) }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ProjectSearchBar(project: String, query: String, onQuery: (String) -> Unit, onClose: () -> Unit) {
    val focus = remember { FocusRequester() }
    val focusManager = LocalFocusManager.current
    val container = SearchBarDefaults.colors().containerColor
    // SearchBarDefaults.InputField opens no keyboard on focus: it hands over to the expanded search bar, which results in place don't use.
    TextField(
        query,
        onQuery,
        modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp).padding(bottom = 8.dp).focusRequester(focus),
        placeholder = { Text("Search $project", maxLines = 1) },
        leadingIcon = { IconButton(onClick = onClose) { Icon(Icons.Back, "Close search") } },
        trailingIcon = if (query == "") null else ({ IconButton(onClick = { onQuery("") }) { Icon(Icons.Close, "Clear") } }),
        singleLine = true,
        shape = SearchBarDefaults.inputFieldShape,
        colors = TextFieldDefaults.colors(
            focusedContainerColor = container,
            unfocusedContainerColor = container,
            focusedIndicatorColor = Color.Transparent,
            unfocusedIndicatorColor = Color.Transparent,
        ),
        keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
        keyboardActions = KeyboardActions(onSearch = { focusManager.clearFocus() }),
    )
    LaunchedEffect(Unit) { focus.requestFocus() }
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
        Text(name, style = MaterialTheme.typography.titleMedium, color = color, maxLines = 1)
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
