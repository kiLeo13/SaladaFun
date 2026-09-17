package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.function.Consumer;
import net.dv8tion.jda.api.components.container.Container;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.entities.channel.middleman.GuildMessageChannel;
import net.dv8tion.jda.api.requests.restaction.MessageCreateAction;
import net.dv8tion.jda.api.utils.messages.MessageCreateData;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;

/** Verifies Padinho's Components V2 player-presence notifications. */
class DiscordPlayerNotificationSenderTest {
    @Test
    void createsGreenJoinedNotificationWithOneTextDisplay() {
        MessageCreateData message = DiscordPlayerNotificationSender.createMessage(
            "Player",
            PlayerPresence.JOINED
        );

        assertNotification(
            message,
            DiscordPlayerNotificationSender.JOINED_ACCENT_COLOR,
            "**Player** entrou no servidor"
        );
    }

    @Test
    void createsRedLeftNotificationWithOneTextDisplay() {
        MessageCreateData message = DiscordPlayerNotificationSender.createMessage(
            "Player",
            PlayerPresence.LEFT
        );

        assertNotification(
            message,
            DiscordPlayerNotificationSender.LEFT_ACCENT_COLOR,
            "**Player** saiu do servidor"
        );
    }

    @Test
    void createsReasonedDisconnectNotification() {
        MessageCreateData message = DiscordPlayerNotificationSender.createMessage(
            "Player",
            PlayerPresence.disconnected("Timed out")
        );

        assertNotification(
            message,
            DiscordPlayerNotificationSender.LEFT_ACCENT_COLOR,
            "**Player** desconectou: Timed out"
        );
    }

    @Test
    void normalizesAndEscapesDisconnectReasonMarkdown() {
        MessageCreateData message = DiscordPlayerNotificationSender.createMessage(
            "Player",
            PlayerPresence.disconnected("  Bad  **packet**\nreceived  ")
        );

        Container container = message.getComponents().getFirst().asContainer();
        assertEquals(
            "**Player** desconectou: Bad \\*\\*packet\\*\\* received",
            container.getComponents().getFirst().asTextDisplay().getContent()
        );
        assertTrue(message.getAllowedMentions().isEmpty());
    }

    @Test
    void limitsLongDisconnectReasonWithoutSplittingUnicode() {
        MessageCreateData message = DiscordPlayerNotificationSender.createMessage(
            "Player",
            PlayerPresence.disconnected("😀".repeat(DiscordPlayerNotificationSender.MAX_COMPONENT_TEXT_LENGTH))
        );

        String content = message.getComponents()
            .getFirst()
            .asContainer()
            .getComponents()
            .getFirst()
            .asTextDisplay()
            .getContent();
        assertTrue(content.length() <= DiscordPlayerNotificationSender.MAX_COMPONENT_TEXT_LENGTH);
        assertFalse(Character.isHighSurrogate(content.charAt(content.length() - 2)));
        assertTrue(content.endsWith("…"));
    }

    @Test
    @SuppressWarnings("unchecked")
    void sendsNotificationThroughTheBotChannel() {
        GuildMessageChannel channel = mock(GuildMessageChannel.class);
        MessageCreateAction action = mock(MessageCreateAction.class);
        when(channel.sendMessage(any(MessageCreateData.class))).thenReturn(action);

        new DiscordPlayerNotificationSender(channel, mock(Logger.class))
            .send("Player", PlayerPresence.JOINED);

        verify(channel).sendMessage(any(MessageCreateData.class));
        verify(action).queue(any(Consumer.class), any(Consumer.class));
    }

    @Test
    void escapesPlayerMarkdownWithoutEnablingMentions() {
        MessageCreateData message = DiscordPlayerNotificationSender.createMessage(
            "**Player**",
            PlayerPresence.JOINED
        );

        Container container = message.getComponents().getFirst().asContainer();
        assertEquals(
            "**\\*\\*Player\\*\\*** entrou no servidor",
            container.getComponents().getFirst().asTextDisplay().getContent()
        );
        assertTrue(message.getAllowedMentions().isEmpty());
    }

    @Test
    @SuppressWarnings("unchecked")
    void logsBotChannelDeliveryFailureWithoutMessageContent() {
        GuildMessageChannel channel = mock(GuildMessageChannel.class);
        MessageCreateAction action = mock(MessageCreateAction.class);
        Logger logger = mock(Logger.class);
        when(channel.sendMessage(any(MessageCreateData.class))).thenReturn(action);
        doAnswer(invocation -> {
            Consumer<Throwable> failure = invocation.getArgument(1);
            failure.accept(new IllegalStateException("sensitive details"));
            return null;
        }).when(action).queue(any(Consumer.class), any(Consumer.class));

        new DiscordPlayerNotificationSender(channel, logger)
            .send("Player", PlayerPresence.LEFT);

        verify(logger).warn(
            "Could not deliver a Minecraft player presence notification through Discord ({})",
            "IllegalStateException"
        );
    }

    private static void assertNotification(MessageCreateData message, int accentColor, String text) {
        assertTrue(message.isUsingComponentsV2());
        assertTrue(message.getAllowedMentions().isEmpty());
        assertTrue(message.getContent().isEmpty());
        assertTrue(message.getEmbeds().isEmpty());
        assertEquals(1, message.getComponents().size());

        Container container = message.getComponents().getFirst().asContainer();
        assertEquals(accentColor, container.getAccentColorRaw());
        assertEquals(1, container.getComponents().size());
        assertEquals(text, container.getComponents().getFirst().asTextDisplay().getContent());
    }
}
