package sld.saladafun.discordutils.platform.neoforge;

import java.util.Objects;
import net.neoforged.neoforge.event.ServerChatEvent;
import sld.saladafun.discordutils.discord.DiscordChatBridge;

/** Publishes accepted final Minecraft chat messages through the Discord bridge. */
public final class MinecraftChatListener {
    private final DiscordChatBridge chatBridge;

    /** Creates a listener that forwards chat to the supplied bridge. */
    public MinecraftChatListener(DiscordChatBridge chatBridge) {
        this.chatBridge = Objects.requireNonNull(chatBridge, "chatBridge");
    }

    /** Handles an uncancelled final chat event from NeoForge's game event bus. */
    public void onServerChat(ServerChatEvent event) {
        if (event.isCanceled()) {
            return;
        }

        chatBridge.publish(
            event.getUsername(),
            event.getPlayer().getUUID(),
            event.getMessage().getString()
        );
    }
}
