package io.github.ilyasturki.ordo.compose.model

import kotlinx.serialization.Serializable

// What every device shows of one project: projects/<id>.view.json of docs/file-format.md.
@Serializable
data class View(
    val folds: Set<String> = emptySet(),
    val filter: Filter = Filter.NONE,
) {
    val empty: Boolean get() = folds.isEmpty() && !filter.active
}

// A change to a view: a field left out keeps what the view holds.
@Serializable
data class ViewEdit(
    val folds: Set<String>? = null,
    val filter: Filter? = null,
) {
    fun onto(view: View) = View(folds ?: view.folds, filter ?: view.filter)

    fun then(later: ViewEdit) = ViewEdit(later.folds ?: folds, later.filter ?: filter)
}
