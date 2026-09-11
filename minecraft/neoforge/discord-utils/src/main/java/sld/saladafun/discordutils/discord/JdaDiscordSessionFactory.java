package sld.saladafun.discordutils.discord;

import java.util.Objects;
import java.util.function.Consumer;
import org.slf4j.Logger;
import sld.saladafun.discordutils.config.DiscordChatSettings;

/** Creates production JDA-backed Discord sessions. */
final class JdaDiscordSessionFactory implements DiscordSessionFactory {
    private final OnlinePlayerNamesProvider onlinePlayerNames;
    private final Logger logger;

    /** Creates a factory that logs through NeoForge and JDA's SLF4J backend. */
    JdaDiscordSessionFactory(OnlinePlayerNamesProvider onlinePlayerNames, Logger logger) {
        this.onlinePlayerNames = Objects.requireNonNull(onlinePlayerNames, "onlinePlayerNames");
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    /** Creates one unactivated JDA candidate session. */
    @Override
    public DiscordSession connect(
        DiscordChatSettings settings,
        Consumer<DiscordInboundMessage> destination,
        DiscordConnectionObserver observer
    ) {
        return new JdaDiscordSession(settings, destination, onlinePlayerNames, observer, logger);
    }
}
