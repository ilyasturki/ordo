package io.github.ilyasturki.ordo.compose.sync

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PairingTest {
    @Test
    fun aServerAddressIsReadFromWhatIsTyped() {
        assertEquals("https://ordo.example.com", parseServer(" https://ordo.example.com/ "))
        assertEquals("https://ordo.example.com", parseServer("ordo.example.com"))
        assertEquals("http://10.0.2.2:7466", parseServer("http://10.0.2.2:7466"))
        assertEquals("https://example.com/ordo", parseServer("example.com/ordo/"))
        assertNull(parseServer(""))
        assertNull("no dot: still typing", parseServer("ordo"))
        assertNull(parseServer("ftp://ordo.example.com"))
        assertNull("a pairing link pairs", parseServer("https://ordo.example.com/pair/abc"))
    }

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
