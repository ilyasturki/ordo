package io.github.ilyasturki.ordo.compose.ui.settings

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ChangelogTest {
    private val sample = """
        # Changelog

        Preamble that never shows.

        ## [Unreleased]

        ## [1.1.0] - 2026-10-11

        Read the [guide](https://example.com) first.

        ### Added

        - `gc` shows the changelog

        [Unreleased]: https://example.com/compare/v1.1.0...HEAD
        [1.1.0]: https://example.com/compare/v1.0.0...v1.1.0
    """.trimIndent()

    @Test
    fun releasesSectionsAndBulletsComeOutInOrderWithLinksStripped() {
        assertEquals(
            listOf(
                ChangelogLine(ChangelogKind.Release, "1.1.0", "2026-10-11"),
                ChangelogLine(ChangelogKind.Paragraph, "Read the guide first."),
                ChangelogLine(ChangelogKind.Section, "Added"),
                ChangelogLine(ChangelogKind.Bullet, "`gc` shows the changelog"),
            ),
            parseChangelog(sample),
        )
    }

    @Test
    fun theRealChangelogEndsWithTheFirstReleaseAndKeepsNoReferences() {
        val lines = parseChangelog(File(System.getProperty("ordo.repo"), "CHANGELOG.md").readText())
        assertTrue(lines.any { it.kind == ChangelogKind.Release && it.text == "0.14.1" })
        assertEquals("0.1.1", lines.last { it.kind == ChangelogKind.Release }.text)
        assertFalse(lines.any { it.text.contains("compare/") })
    }
}
