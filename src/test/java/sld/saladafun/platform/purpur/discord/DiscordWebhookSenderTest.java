package sld.saladafun.platform.purpur.discord;

import net.dv8tion.jda.api.entities.IncomingWebhookClient;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.requests.restaction.WebhookMessageCreateAction;
import org.junit.jupiter.api.Test;

import java.util.EnumSet;
import java.util.UUID;
import java.util.function.Consumer;
import java.util.logging.Logger;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class DiscordWebhookSenderTest {

    @Test
    @SuppressWarnings("unchecked")
    void sendsPlayerIdentityHeadAvatarAndNoMentions() {
        IncomingWebhookClient webhook = mock(IncomingWebhookClient.class);
        WebhookMessageCreateAction<Message> action = mock(WebhookMessageCreateAction.class);
        when(webhook.sendMessage("hello @everyone")).thenReturn(action);
        when(action.setUsername("Alex")).thenReturn(action);
        UUID playerId = UUID.fromString("01234567-89ab-cdef-0123-456789abcdef");
        String avatar = "https://api.mcheads.org/head/" + playerId + "/128";
        when(action.setAvatarUrl(avatar)).thenReturn(action);
        when(action.setAllowedMentions(EnumSet.noneOf(Message.MentionType.class)))
            .thenReturn(action);

        new DiscordWebhookSender(webhook, Logger.getAnonymousLogger())
            .send("Alex", playerId, "hello @everyone");

        verify(action).setUsername("Alex");
        verify(action).setAvatarUrl(avatar);
        verify(action).setAllowedMentions(EnumSet.noneOf(Message.MentionType.class));
        verify(action).queue(any(Consumer.class), any(Consumer.class));
    }

    @Test
    void ignoresBlankMinecraftMessages() {
        IncomingWebhookClient webhook = mock(IncomingWebhookClient.class);

        new DiscordWebhookSender(webhook, Logger.getAnonymousLogger())
            .send("Alex", UUID.randomUUID(), "  ");

        verify(webhook, never()).sendMessage(any(String.class));
    }

    @Test
    @SuppressWarnings("unchecked")
    void truncatesOversizedContentWithoutSplittingUnicodeCodePoints() {
        IncomingWebhookClient webhook = mock(IncomingWebhookClient.class);
        WebhookMessageCreateAction<Message> action = mock(WebhookMessageCreateAction.class);
        when(webhook.sendMessage((String) org.mockito.ArgumentMatchers.<String>argThat(content ->
            content.codePointCount(0, content.length()) == Message.MAX_CONTENT_LENGTH
                && content.endsWith("…")
        ))).thenReturn(action);
        when(action.setUsername(any(String.class))).thenReturn(action);
        when(action.setAvatarUrl(any(String.class))).thenReturn(action);
        when(action.setAllowedMentions(any())).thenReturn(action);
        String oversized = "😀".repeat(Message.MAX_CONTENT_LENGTH + 1);

        new DiscordWebhookSender(webhook, Logger.getAnonymousLogger())
            .send("Alex", UUID.randomUUID(), oversized);

        verify(webhook).sendMessage((String) org.mockito.ArgumentMatchers.<String>argThat(content ->
            content.codePointCount(0, content.length()) == Message.MAX_CONTENT_LENGTH
                && content.endsWith("…")
        ));
    }
}
