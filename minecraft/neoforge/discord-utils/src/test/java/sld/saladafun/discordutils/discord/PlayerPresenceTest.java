package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

import java.util.Optional;
import org.junit.jupiter.api.Test;

/** Verifies player-presence invariants and disconnect-reason normalization. */
class PlayerPresenceTest {
    @Test
    void normalizesDisconnectReasonWhitespace() {
        PlayerPresence presence = PlayerPresence.disconnected("  Internal\n exception  ");

        assertEquals(Optional.of("Internal exception"), presence.disconnectReason());
    }

    @Test
    void rejectsDisconnectReasonForLogin() {
        assertThrows(
            IllegalArgumentException.class,
            () -> new PlayerPresence(PlayerPresence.Type.JOINED, Optional.of("invalid"))
        );
    }
}
