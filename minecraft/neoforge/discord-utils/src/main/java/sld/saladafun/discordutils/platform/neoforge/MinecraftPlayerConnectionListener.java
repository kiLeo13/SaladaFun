package sld.saladafun.discordutils.platform.neoforge;

import java.util.Objects;
import net.neoforged.neoforge.event.entity.player.PlayerEvent;
import sld.saladafun.discordutils.discord.DiscordChatBridge;
import sld.saladafun.discordutils.discord.PlayerPresence;

/** Publishes Minecraft player login and logout events through the Discord bridge. */
public final class MinecraftPlayerConnectionListener {
    private final DiscordChatBridge chatBridge;

    /** Creates a listener that forwards player presence changes to Discord. */
    public MinecraftPlayerConnectionListener(DiscordChatBridge chatBridge) {
        this.chatBridge = Objects.requireNonNull(chatBridge, "chatBridge");
    }

    /** Handles a completed player login. */
    public void onPlayerLoggedIn(PlayerEvent.PlayerLoggedInEvent event) {
        publishLoggedIn(event.getEntity().getGameProfile().getName());
    }

    /** Handles a player logout. */
    public void onPlayerLoggedOut(PlayerEvent.PlayerLoggedOutEvent event) {
        publishLoggedOut(event.getEntity().getGameProfile().getName());
    }

    /** Publishes an extracted player name as a login for platform-free testing. */
    void publishLoggedIn(String playerName) {
        chatBridge.publishPresence(playerName, PlayerPresence.JOINED);
    }

    /** Publishes an extracted player name as a logout for platform-free testing. */
    void publishLoggedOut(String playerName) {
        chatBridge.publishPresence(playerName, PlayerPresence.LEFT);
    }
}
