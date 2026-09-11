package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.function.Consumer;
import org.junit.jupiter.api.Test;
import org.slf4j.LoggerFactory;
import sld.saladafun.discordutils.config.DiscordChatSettings;

/** Exercises safe replacement and shutdown behavior without connecting to Discord. */
class DiscordChatBridgeTest {
    private static final DiscordChatSettings FIRST_SETTINGS = enabledSettings("1");
    private static final DiscordChatSettings SECOND_SETTINGS = enabledSettings("2");

    @Test
    void readyCandidateReplacesTheActiveSession() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = bridge(factory);

        bridge.reconfigure(FIRST_SETTINGS);
        FakeSession first = factory.sessions().getFirst();
        assertFalse(first.active());

        first.ready();
        bridge.publish("Leo", "hello");

        assertTrue(first.active());
        assertEquals(1, first.publishedMessages());

        bridge.reconfigure(SECOND_SETTINGS);
        FakeSession second = factory.sessions().get(1);
        second.ready();

        assertTrue(first.closed());
        assertTrue(second.active());
        bridge.close();
    }

    @Test
    void failedCandidateLeavesTheActiveSessionAvailable() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = bridge(factory);

        bridge.reconfigure(FIRST_SETTINGS);
        FakeSession first = factory.sessions().getFirst();
        first.ready();

        bridge.reconfigure(SECOND_SETTINGS);
        FakeSession second = factory.sessions().get(1);
        second.fail(new IllegalStateException("channel unavailable"));
        bridge.publish("Leo", "still connected");

        assertFalse(first.closed());
        assertTrue(second.closed());
        assertEquals(1, first.publishedMessages());
        bridge.close();
    }

    @Test
    void synchronousReadyCallbackIsActivatedAfterTheFactoryReturns() {
        FakeSessionFactory factory = new FakeSessionFactory();
        factory.readyBeforeReturning(true);
        DiscordChatBridge bridge = bridge(factory);

        bridge.reconfigure(FIRST_SETTINGS);
        FakeSession session = factory.sessions().getFirst();

        assertTrue(session.active());
        assertFalse(session.closed());
        bridge.close();
    }

    @Test
    void disabledSettingsCloseTheActiveSession() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = bridge(factory);

        bridge.reconfigure(FIRST_SETTINGS);
        FakeSession first = factory.sessions().getFirst();
        first.ready();

        bridge.reconfigure(DiscordChatSettings.disabled());

        assertTrue(first.closed());
        bridge.close();
    }

    @Test
    void presenceNotificationsUseOnlyTheActiveSession() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = bridge(factory);

        bridge.reconfigure(FIRST_SETTINGS);
        FakeSession session = factory.sessions().getFirst();
        bridge.publishPresence("Player", PlayerPresence.JOINED);
        assertEquals(0, session.publishedPresenceEvents());

        session.ready();
        bridge.publishPresence("Player", PlayerPresence.JOINED);
        bridge.publishPresence("Player", PlayerPresence.LEFT);

        assertEquals(2, session.publishedPresenceEvents());
        bridge.close();
    }

    private static DiscordChatBridge bridge(FakeSessionFactory factory) {
        return new DiscordChatBridge(
            message -> { },
            factory,
            LoggerFactory.getLogger(DiscordChatBridgeTest.class)
        );
    }

    private static DiscordChatSettings enabledSettings(String channelId) {
        return new DiscordChatSettings(
            true,
            "token",
            "https://discord.com/api/webhooks/123456789012345678/token",
            channelId
        );
    }

    private static final class FakeSessionFactory implements DiscordSessionFactory {
        private final List<FakeSession> sessions = new ArrayList<>();
        private boolean readyBeforeReturning;

        @Override
        public DiscordSession connect(
            DiscordChatSettings settings,
            Consumer<DiscordInboundMessage> destination,
            DiscordConnectionObserver observer
        ) {
            FakeSession session = new FakeSession(settings, observer);
            sessions.add(session);
            if (readyBeforeReturning) {
                session.ready();
            }
            return session;
        }

        private List<FakeSession> sessions() {
            return sessions;
        }

        private void readyBeforeReturning(boolean value) {
            readyBeforeReturning = value;
        }
    }

    private static final class FakeSession implements DiscordSession {
        private final DiscordChatSettings settings;
        private final DiscordConnectionObserver observer;
        private boolean active;
        private boolean closed;
        private int publishedMessages;
        private int publishedPresenceEvents;

        private FakeSession(DiscordChatSettings settings, DiscordConnectionObserver observer) {
            this.settings = settings;
            this.observer = observer;
        }

        @Override
        public DiscordChatSettings settings() {
            return settings;
        }

        @Override
        public void activate() {
            active = true;
        }

        @Override
        public void publish(String playerName, String content) {
            publishedMessages++;
        }

        @Override
        public void publishPresence(String playerName, PlayerPresence presence) {
            publishedPresenceEvents++;
        }

        @Override
        public boolean awaitTermination(Duration timeout) {
            return true;
        }

        @Override
        public void close() {
            closed = true;
            active = false;
        }

        private void ready() {
            observer.ready(this);
        }

        private void fail(Throwable failure) {
            observer.failed(this, failure);
        }

        private boolean active() {
            return active;
        }

        private boolean closed() {
            return closed;
        }

        private int publishedMessages() {
            return publishedMessages;
        }

        private int publishedPresenceEvents() {
            return publishedPresenceEvents;
        }
    }
}
