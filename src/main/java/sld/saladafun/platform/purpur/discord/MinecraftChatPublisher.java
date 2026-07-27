package sld.saladafun.platform.purpur.discord;

import java.util.UUID;

/**
 * Publishes accepted Minecraft player chat to the configured external channel.
 */
@FunctionalInterface
public interface MinecraftChatPublisher {
    void publish(String playerName, UUID playerId, String content);
}
