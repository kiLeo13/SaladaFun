package sld.saladafun.platform.purpur.discord;

import sld.saladafun.platform.purpur.config.DiscordChatSettings;

import java.time.Duration;
import java.util.UUID;

interface DiscordSession extends AutoCloseable {
    DiscordChatSettings settings();

    void activate();

    void publish(String playerName, UUID playerId, String content);

    boolean awaitTermination(Duration timeout) throws InterruptedException;

    @Override
    void close();
}
