package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.List;
import java.util.function.Consumer;
import java.util.stream.IntStream;
import net.dv8tion.jda.api.components.container.Container;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.entities.channel.middleman.GuildMessageChannel;
import net.dv8tion.jda.api.requests.restaction.MessageCreateAction;
import net.dv8tion.jda.api.utils.messages.MessageCreateData;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;

/** Verifies the minimal Components V2 online-player response. */
class DiscordPlayersCommandResponderTest {
    @Test
    void createsOneSortedDarkGreenPlayerList() {
        List<MessageCreateData> messages = DiscordPlayersCommandResponder.createMessages(
            List.of("zeta", "Alpha", "bravo")
        );

        assertEquals(1, messages.size());
        assertMessage(
            messages.getFirst(),
            DiscordPlayersCommandResponder.HEADING + "\n- Alpha\n- bravo\n- zeta"
        );
    }

    @Test
    void createsExplicitEmptyPlayerList() {
        MessageCreateData message = DiscordPlayersCommandResponder.createMessages(List.of()).getFirst();

        assertMessage(
            message,
            DiscordPlayersCommandResponder.HEADING + "\n- " + DiscordPlayersCommandResponder.NO_PLAYERS
        );
    }

    @Test
    void preservesEveryPlayerAcrossOversizedResponses() {
        List<String> playerNames = IntStream.range(0, 250)
            .mapToObj(index -> "Player_%03d_abcdef".formatted(index))
            .toList();

        List<MessageCreateData> messages = DiscordPlayersCommandResponder.createMessages(playerNames);
        long listedPlayers = messages.stream()
            .map(DiscordPlayersCommandResponderTest::text)
            .flatMap(String::lines)
            .filter(line -> line.startsWith("- "))
            .count();

        assertTrue(messages.size() > 1);
        assertEquals(playerNames.size(), listedPlayers);
    }

    @Test
    @SuppressWarnings("unchecked")
    void sendsResponseThroughPadinhosBotChannel() {
        GuildMessageChannel channel = mock(GuildMessageChannel.class);
        MessageCreateAction action = mock(MessageCreateAction.class);
        when(channel.sendMessage(any(MessageCreateData.class))).thenReturn(action);

        new DiscordPlayersCommandResponder(channel, mock(Logger.class)).send(List.of("Player"));

        verify(channel).sendMessage(any(MessageCreateData.class));
        verify(action).queue(any(Consumer.class), any(Consumer.class));
    }

    @Test
    @SuppressWarnings("unchecked")
    void logsDeliveryFailureWithoutPlayerNames() {
        GuildMessageChannel channel = mock(GuildMessageChannel.class);
        MessageCreateAction action = mock(MessageCreateAction.class);
        Logger logger = mock(Logger.class);
        when(channel.sendMessage(any(MessageCreateData.class))).thenReturn(action);
        doAnswer(invocation -> {
            Consumer<Throwable> failure = invocation.getArgument(1);
            failure.accept(new IllegalStateException("PlayerName"));
            return null;
        }).when(action).queue(any(Consumer.class), any(Consumer.class));

        new DiscordPlayersCommandResponder(channel, logger).send(List.of("PlayerName"));

        verify(logger).warn(
            "Could not deliver the Discord online-players response ({})",
            "IllegalStateException"
        );
    }

    private static void assertMessage(MessageCreateData message, String expectedText) {
        assertTrue(message.isUsingComponentsV2());
        assertTrue(message.getAllowedMentions().isEmpty());
        assertEquals(1, message.getComponents().size());
        Container container = message.getComponents().getFirst().asContainer();
        assertEquals(DiscordPlayersCommandResponder.ACCENT_COLOR, container.getAccentColorRaw());
        assertEquals(1, container.getComponents().size());
        assertEquals(expectedText, container.getComponents().getFirst().asTextDisplay().getContent());
    }

    private static String text(MessageCreateData message) {
        return message.getComponents().getFirst().asContainer()
            .getComponents().getFirst().asTextDisplay().getContent();
    }
}
