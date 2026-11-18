package io.github.ilyasturki.ordo.compose.model

import org.junit.Assert.assertEquals
import org.junit.Test

class DescriptionTest {
    @Test
    fun bothVersionsStayTheOtherDevicesFirst() {
        assertEquals("Theirs\n\nMine", Description.keepBoth("Theirs\n", "Mine"))
    }

    @Test
    fun aVersionTheOtherHoldsIsNotRepeated() {
        assertEquals("Mine, then more", Description.keepBoth("Mine", "Mine, then more"))
        assertEquals("Theirs and mine", Description.keepBoth("Theirs and mine", "mine"))
        assertEquals("Theirs", Description.keepBoth("Theirs", "  "))
    }
}
