package sld.saladafun.discordutils.platform.neoforge;

import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;

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
            .publishLoggedOut("Player");

        verify(bridge).publishPresence("Player", PlayerPresence.LEFT);
    }
}
