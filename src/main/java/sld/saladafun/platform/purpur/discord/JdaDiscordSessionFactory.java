package sld.saladafun.platform.purpur.discord;

import sld.saladafun.platform.purpur.config.DiscordChatSettings;

import java.util.Objects;
import java.util.function.Consumer;
import java.util.logging.Logger;

final class JdaDiscordSessionFactory implements DiscordSessionFactory {
    private final Logger logger;

    JdaDiscordSessionFactory(Logger logger) {
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    @Override
    public DiscordSession connect(
        DiscordChatSettings settings,
        Consumer<DiscordInboundMessage> destination,
        DiscordConnectionObserver observer
    ) {
        return new JdaDiscordSession(settings, destination, observer, logger);
    }
}
