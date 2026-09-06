package sld.saladafun.discordutils.discord;

import java.time.Duration;
import java.util.Objects;
import java.util.UUID;
import java.util.function.Consumer;
import org.slf4j.Logger;
import sld.saladafun.discordutils.config.DiscordChatSettings;

/** Coordinates staged replacement of one active Discord transport session. */
public final class DiscordChatBridge implements AutoCloseable {
    private static final Duration SESSION_TERMINATION_TIMEOUT = Duration.ofSeconds(10);

    private final Consumer<DiscordInboundMessage> inboundDestination;
    private final DiscordSessionFactory sessionFactory;
    private final Logger logger;

    private DiscordSession activeSession;
    private DiscordSession candidateSession;
    private long configurationGeneration;
    private boolean closed;

    /** Creates a production bridge backed by JDA sessions. */
    public DiscordChatBridge(Consumer<DiscordInboundMessage> inboundDestination, Logger logger) {
        this(inboundDestination, new JdaDiscordSessionFactory(logger), logger);
    }

    /** Creates a bridge with an explicit factory for focused lifecycle tests. */
    DiscordChatBridge(
        Consumer<DiscordInboundMessage> inboundDestination,
        DiscordSessionFactory sessionFactory,
        Logger logger
    ) {
        this.inboundDestination = Objects.requireNonNull(inboundDestination, "inboundDestination");
        this.sessionFactory = Objects.requireNonNull(sessionFactory, "sessionFactory");
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    /** Applies settings while preserving active traffic until a candidate is ready. */
    public synchronized void reconfigure(DiscordChatSettings settings) {
        if (closed) {
            throw new IllegalStateException("Discord chat bridge is closed");
        }

        configurationGeneration++;
        closeCandidate();
        if (!settings.enabled()) {
            closeActive();
            logger.info("Discord chat bridge is disabled.");
            return;
        }
        if (activeSession != null && activeSession.settings().equals(settings)) {
            return;
        }

        long generation = configurationGeneration;
        ConnectionObserver observer = new ConnectionObserver(generation);
        try {
            DiscordSession session = sessionFactory.connect(
                settings,
                inboundDestination,
                observer
            );
            candidateSession = session;
            observer.bind(session);
            logger.info("Connecting the Discord chat bridge.");
        } catch (RuntimeException exception) {
            logger.warn(
                "Could not start a Discord chat bridge candidate ({}); the previous session remains active.",
                exception.getClass().getSimpleName()
            );
        }
    }

    /** Publishes content through the active session when one is available. */
    public synchronized void publish(String playerName, UUID playerId, String content) {
        if (activeSession != null) {
            activeSession.publish(playerName, playerId, content);
        }
    }

    /** Closes both active and candidate sessions and waits for termination. */
    @Override
    public synchronized void close() {
        if (closed) {
            return;
        }

        closed = true;
        configurationGeneration++;
        DiscordSession candidate = candidateSession;
        DiscordSession active = activeSession;
        candidateSession = null;
        activeSession = null;
        closeAndAwait(candidate);
        closeAndAwait(active);
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
        logger.warn(
            "Discord chat bridge candidate failed ({}); the previous session remains active.",
            failure.getClass().getSimpleName()
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

    private void closeAndAwait(DiscordSession session) {
        if (session == null) {
            return;
        }

        session.close();
        try {
            if (!session.awaitTermination(SESSION_TERMINATION_TIMEOUT)) {
                logger.warn("Discord session did not stop within {} seconds.", SESSION_TERMINATION_TIMEOUT.toSeconds());
            }
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            logger.warn("Interrupted while waiting for a Discord session to stop.");
        }
    }

    private final class ConnectionObserver implements DiscordConnectionObserver {
        private final long generation;
        private DiscordSession session;
        private DiscordSession readySession;
        private DiscordSession failedSession;
        private Throwable failure;

        private ConnectionObserver(long generation) {
            this.generation = generation;
        }

        /** Binds callbacks to the candidate after its factory has returned it. */
        private void bind(DiscordSession candidate) {
            DiscordSession ready;
            DiscordSession failed;
            Throwable recordedFailure;
            synchronized (this) {
                session = candidate;
                ready = readySession;
                failed = failedSession;
                recordedFailure = failure;
                readySession = null;
                failedSession = null;
                failure = null;
            }

            if (failed != null) {
                rejectCandidate(generation, failed, recordedFailure);
            } else if (ready != null) {
                activateCandidate(generation, ready);
            }
        }

        @Override
        public void ready(DiscordSession session) {
            synchronized (this) {
                if (this.session == null) {
                    readySession = session;
                    return;
                }
            }

            activateCandidate(generation, session);
        }

        @Override
        public void failed(DiscordSession session, Throwable failure) {
            synchronized (this) {
                if (this.session == null) {
                    failedSession = session;
                    this.failure = failure;
                    return;
                }
            }

            rejectCandidate(generation, session, failure);
        }
    }
}
