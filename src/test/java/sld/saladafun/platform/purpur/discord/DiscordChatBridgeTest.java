package sld.saladafun.platform.purpur.discord;

import org.junit.jupiter.api.Test;
import sld.saladafun.platform.purpur.config.DiscordChatSettings;

import java.util.ArrayList;
import java.util.List;
import java.util.UUID;
import java.util.function.Consumer;
import java.util.logging.Logger;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class DiscordChatBridgeTest {

    @Test
    void keepsTheActiveSessionUntilAReplacementIsReady() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = new DiscordChatBridge(
            factory,
            ignored -> {
            },
            Logger.getAnonymousLogger()
        );
        DiscordChatSettings firstSettings = settings("111111111111111111");
        DiscordChatSettings secondSettings = settings("222222222222222222");
        UUID playerId = UUID.randomUUID();

        bridge.reconfigure(firstSettings);
        FakeSession first = factory.latest();
        bridge.publish("Alex", playerId, "before ready");
        assertEquals(List.of(), first.published);

        factory.ready(first);
        bridge.publish("Alex", playerId, "first");
        assertEquals(List.of("first"), first.published);

        bridge.reconfigure(secondSettings);
        FakeSession candidate = factory.latest();
        bridge.publish("Alex", playerId, "during replacement");
        assertEquals(List.of("first", "during replacement"), first.published);
        assertEquals(List.of(), candidate.published);

        factory.ready(candidate);
        assertTrue(first.closed);
        assertTrue(candidate.active);
        bridge.publish("Alex", playerId, "second");
        assertEquals(List.of("second"), candidate.published);

        bridge.close();
        assertTrue(candidate.closed);
    }

    @Test
    void failedReplacementLeavesThePreviousSessionActive() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = new DiscordChatBridge(
            factory,
            ignored -> {
            },
            Logger.getAnonymousLogger()
        );
        UUID playerId = UUID.randomUUID();

        bridge.reconfigure(settings("111111111111111111"));
        FakeSession active = factory.latest();
        factory.ready(active);
        bridge.reconfigure(settings("222222222222222222"));
        FakeSession failed = factory.latest();
        factory.failed(failed);

        bridge.publish("Alex", playerId, "still online");

        assertTrue(failed.closed);
        assertFalse(active.closed);
        assertEquals(List.of("still online"), active.published);
    }

    @Test
    void disablingClosesActiveAndCandidateSessions() {
        FakeSessionFactory factory = new FakeSessionFactory();
        DiscordChatBridge bridge = new DiscordChatBridge(
            factory,
            ignored -> {
            },
            Logger.getAnonymousLogger()
        );

        bridge.reconfigure(settings("111111111111111111"));
        FakeSession active = factory.latest();
        factory.ready(active);
        bridge.reconfigure(settings("222222222222222222"));
        FakeSession candidate = factory.latest();
        bridge.reconfigure(DiscordChatSettings.disabled());

        assertTrue(active.closed);
        assertTrue(candidate.closed);
    }

    private DiscordChatSettings settings(String channelId) {
        return new DiscordChatSettings(
            true,
            "token-" + channelId,
            "https://discord.com/api/webhooks/" + channelId + "/secret",
            channelId
        );
    }

    private static final class FakeSessionFactory implements DiscordSessionFactory {
        private final List<FakeSession> sessions = new ArrayList<>();
        private DiscordConnectionObserver observer;

        @Override
        public DiscordSession connect(
            DiscordChatSettings settings,
            Consumer<DiscordInboundMessage> destination,
            DiscordConnectionObserver observer
        ) {
            this.observer = observer;
            FakeSession session = new FakeSession(settings);
            sessions.add(session);
            return session;
        }

        FakeSession latest() {
            return sessions.getLast();
        }

        void ready(FakeSession session) {
            observer.ready(session);
        }

        void failed(FakeSession session) {
            observer.failed(session, new IllegalStateException("test failure"));
        }
    }

    private static final class FakeSession implements DiscordSession {
        private final DiscordChatSettings settings;
        private final List<String> published = new ArrayList<>();
        private boolean active;
        private boolean closed;

        private FakeSession(DiscordChatSettings settings) {
            this.settings = settings;
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
        public void publish(String playerName, UUID playerId, String content) {
            if (active && !closed) {
                published.add(content);
            }
        }

        @Override
        public void close() {
            active = false;
            closed = true;
        }
    }
}
