package sld.saladafun.platform.purpur.discord;

import sld.saladafun.platform.purpur.config.DiscordChatSettings;

import java.time.Duration;
import java.util.Objects;
import java.util.UUID;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Owns the active and candidate Discord sessions. Reconfiguration only replaces
 * a working session after the candidate reaches READY.
 */
public final class DiscordChatBridge implements MinecraftChatPublisher, AutoCloseable {
    private static final Duration SESSION_TERMINATION_TIMEOUT = Duration.ofSeconds(10);

    private final DiscordSessionFactory sessionFactory;
    private final Consumer<DiscordInboundMessage> inboundDestination;
    private final Logger logger;

    private DiscordSession activeSession;
    private DiscordSession candidateSession;
    private long configurationGeneration;
    private boolean closed;

    public DiscordChatBridge(
        Consumer<DiscordInboundMessage> inboundDestination,
        Logger logger
    ) {
        this(new JdaDiscordSessionFactory(logger), inboundDestination, logger);
    }

    DiscordChatBridge(
        DiscordSessionFactory sessionFactory,
        Consumer<DiscordInboundMessage> inboundDestination,
        Logger logger
    ) {
        this.sessionFactory = Objects.requireNonNull(sessionFactory, "sessionFactory");
        this.inboundDestination = Objects.requireNonNull(
            inboundDestination,
            "inboundDestination"
        );
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    public synchronized void reconfigure(DiscordChatSettings settings) {
        Objects.requireNonNull(settings, "settings");
        if (closed) {
            throw new IllegalStateException("Discord chat bridge is closed");
        }

        long generation = ++configurationGeneration;
        closeCandidate();
        if (!settings.enabled()) {
            closeActive();
            logger.info("Discord chat bridge is disabled.");
            return;
        }
        if (activeSession != null && activeSession.settings().equals(settings)) {
            return;
        }

        DiscordConnectionObserver observer = new DiscordConnectionObserver() {
            @Override
            public void ready(DiscordSession session) {
                activateCandidate(generation, session);
            }

            @Override
            public void failed(DiscordSession session, Throwable failure) {
                rejectCandidate(generation, session, failure);
            }
        };
        try {
            candidateSession = sessionFactory.connect(
                settings,
                inboundDestination,
                observer
            );
            logger.info("Connecting the Discord chat bridge.");
        } catch (RuntimeException exception) {
            logger.warning(
                "Could not start a Discord chat bridge candidate ("
                    + exception.getClass().getSimpleName()
                    + "); the previous session remains active."
            );
        }
    }

    @Override
    public void publish(String playerName, UUID playerId, String content) {
        DiscordSession session;
        synchronized (this) {
            session = activeSession;
        }
        if (session != null) {
            session.publish(playerName, playerId, content);
        }
    }

    @Override
    public void close() {
        DiscordSession candidate;
        DiscordSession active;
        synchronized (this) {
            if (closed) {
                return;
            }
            closed = true;
            configurationGeneration++;
            candidate = candidateSession;
            candidateSession = null;
            active = activeSession;
            activeSession = null;
        }

        closeSession(candidate);
        closeSession(active);
        awaitTermination(candidate);
        awaitTermination(active);
    }

    private synchronized void activateCandidate(long generation, DiscordSession session) {
        if (closed || generation != configurationGeneration || session != candidateSession) {
            session.close();
            return;
        }
        DiscordSession previous = activeSession;
        activeSession = session;
        candidateSession = null;
        session.activate();
        if (previous != null) {
            previous.close();
        }
        logger.info("Discord chat bridge connected.");
    }

    private synchronized void rejectCandidate(
        long generation,
        DiscordSession session,
        Throwable failure
    ) {
        if (generation != configurationGeneration || session != candidateSession) {
            session.close();
            return;
        }
        candidateSession = null;
        session.close();
        logger.warning(
            "Discord chat bridge candidate failed ("
                + failure.getClass().getSimpleName()
                + "); the previous session remains active."
        );
    }

    private void closeCandidate() {
        if (candidateSession != null) {
            candidateSession.close();
            candidateSession = null;
        }
    }

    private void closeActive() {
        if (activeSession != null) {
            activeSession.close();
            activeSession = null;
        }
    }

    private void closeSession(DiscordSession session) {
        if (session != null) {
            session.close();
        }
    }

    private void awaitTermination(DiscordSession session) {
        if (session == null) {
            return;
        }
        try {
            if (!session.awaitTermination(SESSION_TERMINATION_TIMEOUT)) {
                logger.warning(
                    "Discord session did not stop within "
                        + SESSION_TERMINATION_TIMEOUT.toSeconds()
                        + " seconds."
                );
            }
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            logger.warning("Interrupted while waiting for a Discord session to stop.");
        }
    }
}
