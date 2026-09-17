package sld.saladafun.discordutils.platform.neoforge;

import java.util.Objects;
import java.util.Optional;
import net.minecraft.network.DisconnectionDetails;
import net.minecraft.network.chat.Component;
import net.minecraft.network.chat.contents.TranslatableContents;
import net.minecraft.server.level.ServerPlayer;
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
        ServerPlayer player = (ServerPlayer) event.getEntity();
        DisconnectionDetails details = player.connection.getConnection().getDisconnectionDetails();
        publishLoggedOut(
            player.getGameProfile().getName(),
            displayedDisconnectReason(details)
        );
    }

    /** Publishes an extracted player name as a login for platform-free testing. */
    void publishLoggedIn(String playerName) {
        chatBridge.publishPresence(playerName, PlayerPresence.JOINED);
    }

    /** Publishes an extracted player name as a logout for platform-free testing. */
    void publishLoggedOut(String playerName, Optional<String> disconnectReason) {
        PlayerPresence presence = disconnectReason
            .map(PlayerPresence::disconnected)
            .orElse(PlayerPresence.LEFT);
        chatBridge.publishPresence(playerName, presence);
    }

    /** Extracts a visible reason while suppressing vanilla's routine connection-close reason. */
    static Optional<String> displayedDisconnectReason(DisconnectionDetails details) {
        if (details == null) {
            return Optional.empty();
        }

        Component reason = details.reason();
        String translationKey = reason.getContents() instanceof TranslatableContents contents
            ? contents.getKey()
            : null;
        return displayedDisconnectReason(translationKey, reason.getString());
    }

    /** Classifies platform-extracted reason text without requiring Minecraft types in unit tests. */
    static Optional<String> displayedDisconnectReason(String translationKey, String renderedReason) {
        if ("disconnect.endOfStream".equals(translationKey)) {
            return Optional.empty();
        }

        return Optional.ofNullable(renderedReason)
            .map(String::strip)
            .filter(reason -> !reason.isEmpty());
    }
}
