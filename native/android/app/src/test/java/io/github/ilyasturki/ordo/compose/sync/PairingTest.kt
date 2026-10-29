package io.github.ilyasturki.ordo.compose.sync

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PairingTest {
    @Test
    fun aLinkGivesTheServerAndTheCode() {
        assertEquals(Pairing("https://ordo.example.com", "ABC123"), parsePairing(" https://ordo.example.com/pair/ABC123 "))
        assertEquals(Pairing("http://localhost:7320", "X"), parsePairing("http://localhost:7320/pair/X/"))
        assertEquals(Pairing("https://example.com/ordo", "C D"), parsePairing("https://example.com/ordo/pair/C%20D"))
    }

    @Test
    fun anythingElseIsNoPairing() {
        assertNull(parsePairing(""))
        assertNull(parsePairing("ABC123"))
        assertNull(parsePairing("ftp://example.com/pair/X"))
        assertNull(parsePairing("https://example.com/devices"))
    }
}
