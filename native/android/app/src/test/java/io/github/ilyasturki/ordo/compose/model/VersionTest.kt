package io.github.ilyasturki.ordo.compose.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class VersionTest {
    @Test
    fun aNameHoldsAVersionAsTheTuiReadsIt() {
        for (name in listOf("v1.3.0", "1.3", "Release v0.9 - polish", "app 2.0.1")) assertTrue(name, hasVersion(name))
        for (name in listOf("Inbox", "v1", "1.2.3.4", "x1.2", "")) assertFalse(name, hasVersion(name))
    }

    @Test
    fun aSubCategoryGoesFromItsParentsVersion() {
        val project = Project(
            id = "p",
            categories = listOf(Category("v", "v1.0.0"), Category("ui", "ui", parentId = "v"), Category("f", "Future"), Category("x", "x", parentId = "f")),
        )
        assertEquals(listOf(true, true, false, false), project.categories.map(project::versioned))
    }
}
