package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertThrows;

import org.junit.jupiter.api.Test;

/** Tests validation of copied Discord reply references. */
class DiscordReplyReferenceTest {
    @Test
    void nullContentIsRejected() {
        assertThrows(
            NullPointerException.class,
            () -> new DiscordReplyReference("Lucas", null, 0, 0)
        );
    }

    @Test
    void negativeMediaCountsAreRejected() {
        assertThrows(
            IllegalArgumentException.class,
            () -> new DiscordReplyReference("Lucas", "Hello", 0, -1)
        );
    }
}
