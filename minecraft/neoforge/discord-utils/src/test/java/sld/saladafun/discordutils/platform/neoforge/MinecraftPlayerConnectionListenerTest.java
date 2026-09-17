package sld.saladafun.discordutils.platform.neoforge;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;

import java.util.Optional;
import org.junit.jupiter.api.Test;
import sld.saladafun.discordutils.discord.DiscordChatBridge;
import sld.saladafun.discordutils.discord.PlayerPresence;

/** Verifies NeoForge player connection events are mapped to Discord presence events. */
class MinecraftPlayerConnectionListenerTest {
    @Test
    void publishesPlayerLogin() {
        DiscordChatBridge bridge = mock(DiscordChatBridge.class);

        new MinecraftPlayerConnectionListener(bridge)
            .publishLoggedIn("Player");

        verify(bridge).publishPresence("Player", PlayerPresence.JOINED);
    }

    @Test
    void publishesPlayerLogout() {
        DiscordChatBridge bridge = mock(DiscordChatBridge.class);

        new MinecraftPlayerConnectionListener(bridge)
            .publishLoggedOut("Player", Optional.empty());

        verify(bridge).publishPresence("Player", PlayerPresence.LEFT);
    }

    @Test
    void publishesAbnormalLogoutWithItsReason() {
        DiscordChatBridge bridge = mock(DiscordChatBridge.class);

        new MinecraftPlayerConnectionListener(bridge)
            .publishLoggedOut("Player", Optional.of("Timed out"));

        verify(bridge).publishPresence("Player", PlayerPresence.disconnected("Timed out"));
    }

    @Test
    void suppressesRoutineEndOfStreamReason() {
        Optional<String> reason = MinecraftPlayerConnectionListener.displayedDisconnectReason(
            "disconnect.endOfStream",
            "Disconnected"
        );

        assertTrue(reason.isEmpty());
    }

    @Test
    void exposesMeaningfulDisconnectReason() {
        Optional<String> reason = MinecraftPlayerConnectionListener.displayedDisconnectReason(
            "disconnect.timeout",
            "Timed out"
        );

        assertEquals(Optional.of("Timed out"), reason);
    }
}
