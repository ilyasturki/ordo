package io.github.ilyasturki.ordo.compose

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.width
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasContentDescription
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.unit.DpRect
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.height
import androidx.compose.ui.unit.width
import androidx.test.ext.junit.runners.AndroidJUnit4
import io.github.ilyasturki.ordo.compose.model.Category
import io.github.ilyasturki.ordo.compose.ui.OrdoTheme
import io.github.ilyasturki.ordo.compose.ui.home.AddBar
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

// The add sheet's bar on the narrowest phone it targets.
@RunWith(AndroidJUnit4::class)
class AddBarTest {
    @get:Rule
    val compose = createComposeRule()

    private fun bar(name: String, describing: Boolean = false, priority: String = "", tag: String = "", estimate: Int = 0) {
        compose.setContent {
            OrdoTheme(dark = false) {
                Box(Modifier.width(360.dp)) {
                    AddBar(listOf(Category("c", name)), "c", describing, priority, tag, estimate, true, {}, {}, {}, {}, {}, {})
                }
            }
        }
    }

    private fun bounds(matcher: SemanticsMatcher): DpRect = compose.onNode(matcher).assertIsDisplayed().getUnclippedBoundsInRoot()

    private fun assertOneLine(category: String, vararg controls: String) {
        val save = bounds(hasText("Save") and hasClickAction())
        assertTrue("Save is at $save", save.right <= 360.dp && save.width >= 56.dp)
        val row = listOf(bounds(hasText(category, substring = true) and hasClickAction())) + controls.map { bounds(hasContentDescription(it)) }
        for ((i, b) in row.withIndex()) {
            assertEquals("control $i is off Save's line", (save.top + save.height / 2).value, (b.top + b.height / 2).value, 1f)
            assertTrue("control $i runs into Save", b.right <= save.left)
            assertTrue("control $i is squeezed to ${b.width}", b.width >= 39.5.dp)
        }
        row.zipWithNext().forEach { (a, b) -> assertTrue("controls overlap", a.right <= b.left) }
    }

    @Test
    fun nothingSetFitsOneLine() {
        bar("v1.3.0")
        assertOneLine("v1.3.0", "Description", "Priority", "Colour", "Estimate")
    }

    @Test
    fun everythingSetFitsOneLine() {
        bar("v1.3.0", priority = "critical", tag = "magenta", estimate = 2400)
        assertOneLine("v1.3.0", "Description", "Priority", "Colour", "Estimate")
    }

    @Test
    fun everythingSetWithTheDescriptionOpenFitsOneLine() {
        bar("v1.3.0", describing = true, priority = "critical", tag = "magenta", estimate = 2400)
        assertOneLine("v1.3.0", "Priority", "Colour", "Estimate")
    }

    @Test
    fun aLongCategoryNameShortensInsteadOfPushingSaveOff() {
        bar("Everything that waits for the next release", priority = "trivial", tag = "cyan", estimate = 30)
        assertOneLine("Everything", "Description", "Priority", "Colour", "Estimate")
    }
}
