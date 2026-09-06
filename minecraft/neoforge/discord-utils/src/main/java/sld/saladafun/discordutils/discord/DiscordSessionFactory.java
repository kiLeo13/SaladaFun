package sld.saladafun.discordutils.discord;

import java.util.function.Consumer;
import sld.saladafun.discordutils.config.DiscordChatSettings;

/** Creates Discord sessions without coupling bridge lifecycle code to JDA. */
interface DiscordSessionFactory {
    /** Connects a candidate session using settings and lifecycle callbacks. */
    DiscordSession connect(
        DiscordChatSettings settings,
        Consumer<DiscordInboundMessage> destination,
        DiscordConnectionObserver observer
    );
}
