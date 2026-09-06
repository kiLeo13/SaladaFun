package sld.saladafun.platform.purpur.discord;

import sld.saladafun.platform.purpur.config.DiscordChatSettings;

import java.util.function.Consumer;

@FunctionalInterface
interface DiscordSessionFactory {
    DiscordSession connect(
        DiscordChatSettings settings,
        Consumer<DiscordInboundMessage> destination,
        DiscordConnectionObserver observer
    );
}
