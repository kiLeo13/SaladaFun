package sld.saladafun.discordutils.discord;

import java.time.Duration;
import sld.saladafun.discordutils.config.DiscordChatSettings;

/** Transport abstraction for a candidate or active Discord connection. */
interface DiscordSession extends AutoCloseable {
    /** Returns the immutable settings that created this session. */
    DiscordChatSettings settings();

    /** Enables traffic after the bridge accepts this ready session. */
    void activate();

    /** Queues one Minecraft-originated message for Discord delivery. */
    void publish(String playerName, String content);

    /** Queues one Minecraft player-presence notification for Discord delivery. */
    void publishPresence(String playerName, PlayerPresence presence);

    /** Waits for session shutdown up to timeout. */
    boolean awaitTermination(Duration timeout) throws InterruptedException;

    /** Stops the session without waiting for termination. */
    @Override void close();
}
