package sld.saladafun.discordutils.discord;

import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;

import java.util.List;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.Consumer;
import org.junit.jupiter.api.Test;

/** Verifies delayed player snapshots remain bound to the originating active session. */
class DiscordPlayersCommandHandlerTest {
    @Test
    void activeSessionSendsCompletedPlayerSnapshot() {
        AtomicReference<Consumer<List<String>>> destination = new AtomicReference<>();
        OnlinePlayerNamesProvider provider = destination::set;
        DiscordPlayersCommandResponder responder = mock(DiscordPlayersCommandResponder.class);
        DiscordPlayersCommandHandler handler = new DiscordPlayersCommandHandler(
            provider,
            () -> true,
            () -> responder
        );

        handler.accept(request());
        destination.get().accept(List.of("Player"));

        verify(responder).send(List.of("Player"));
    }

    @Test
    void replacedSessionDropsDelayedPlayerSnapshot() {
        AtomicReference<Consumer<List<String>>> destination = new AtomicReference<>();
        AtomicBoolean active = new AtomicBoolean(true);
        OnlinePlayerNamesProvider provider = destination::set;
        DiscordPlayersCommandResponder responder = mock(DiscordPlayersCommandResponder.class);
        DiscordPlayersCommandHandler handler = new DiscordPlayersCommandHandler(
            provider,
            active::get,
            () -> responder
        );

        handler.accept(request());
        active.set(false);
        destination.get().accept(List.of("Player"));

        verify(responder, never()).send(List.of("Player"));
    }

    private static DiscordMessageCommandRequest request() {
        return new DiscordMessageCommandRequest("!players", "!players", "", List.of());
    }
}
