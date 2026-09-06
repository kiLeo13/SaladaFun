package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;
import org.junit.jupiter.api.Test;

/** Tests visible-content filtering before Discord callbacks cross into Minecraft. */
class DiscordInboundMessageTest {
    @Test void visibleContentIncludesTextOrSupportedMedia() {
        assertFalse(new DiscordInboundMessage("name", " ", 0, 0).hasVisibleContent());
        assertTrue(new DiscordInboundMessage("name", "text", 0, 0).hasVisibleContent());
        assertTrue(new DiscordInboundMessage("name", "", 1, 0).hasVisibleContent());
        assertTrue(new DiscordInboundMessage("name", "", 0, 1).hasVisibleContent());
    }
    @Test void negativeMediaCountsAreRejected() {
        assertThrows(IllegalArgumentException.class, () -> new DiscordInboundMessage("name", "", -1, 0));
    }
}
