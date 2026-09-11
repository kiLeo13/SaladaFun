package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.concurrent.atomic.AtomicReference;
import org.junit.jupiter.api.Test;

/** Verifies immutable registered-command dispatch before chat fallthrough. */
class DiscordMessageCommandRegistryTest {
    @Test
    void dispatchesCaseInsensitiveExactFirstTokenWithArguments() {
        AtomicReference<DiscordMessageCommandRequest> received = new AtomicReference<>();
        DiscordMessageCommandRegistry registry = new DiscordMessageCommandRegistry();
        registry.register("!players", received::set);
        registry.freeze();

        assertTrue(registry.dispatch("  !PLAYERS   one\ttwo  "));

        DiscordMessageCommandRequest request = received.get();
        assertEquals("!PLAYERS", request.trigger());
        assertEquals("!PLAYERS   one\ttwo", request.content());
        assertEquals("one\ttwo", request.rawArguments());
        assertEquals(java.util.List.of("one", "two"), request.arguments());
    }

    @Test
    void unknownOrNonLeadingCommandsFallThrough() {
        DiscordMessageCommandRegistry registry = new DiscordMessageCommandRegistry();
        registry.register("!players", request -> { });
        registry.freeze();

        assertFalse(registry.dispatch(""));
        assertFalse(registry.dispatch("!players-extra"));
        assertFalse(registry.dispatch("hello !players"));
    }

    @Test
    void validatesRegistryLifecycleAndDuplicateTriggers() {
        DiscordMessageCommandRegistry unfrozen = new DiscordMessageCommandRegistry();
        assertThrows(IllegalStateException.class, () -> unfrozen.dispatch("!players"));

        DiscordMessageCommandRegistry duplicate = new DiscordMessageCommandRegistry();
        duplicate.register("!players", request -> { });
        duplicate.register("!PLAYERS", request -> { });
        assertThrows(IllegalArgumentException.class, duplicate::freeze);

        DiscordMessageCommandRegistry frozen = new DiscordMessageCommandRegistry();
        frozen.freeze();
        assertThrows(
            IllegalStateException.class,
            () -> frozen.register("!players", request -> { })
        );
    }
}
