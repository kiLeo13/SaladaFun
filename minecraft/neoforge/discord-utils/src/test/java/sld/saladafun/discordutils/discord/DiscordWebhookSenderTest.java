package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import java.util.EnumSet;
import java.util.function.Consumer;
import net.dv8tion.jda.api.entities.IncomingWebhookClient;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.requests.restaction.WebhookMessageCreateAction;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;

/** Tests webhook identity customization and Discord content limiting. */
class DiscordWebhookSenderTest {
    @Test
    @SuppressWarnings("unchecked")
    void sendsUsernameBasedHeadAvatarForOfflinePlayers() {
        IncomingWebhookClient webhook = mock(IncomingWebhookClient.class);
        WebhookMessageCreateAction<Message> action = mock(WebhookMessageCreateAction.class);
        String avatar = "https://api.mcheads.org/head/Offline_Player/128/hat";
        when(webhook.sendMessage("hello")).thenReturn(action);
        when(action.setUsername("Offline_Player")).thenReturn(action);
        when(action.setAvatarUrl(avatar)).thenReturn(action);
        when(action.setAllowedMentions(EnumSet.noneOf(Message.MentionType.class)))
            .thenReturn(action);

        new DiscordWebhookSender(webhook, mock(Logger.class))
            .send("Offline_Player", "hello");

        verify(action).setUsername("Offline_Player");
        verify(action).setAvatarUrl(avatar);
        verify(action).setAllowedMentions(EnumSet.noneOf(Message.MentionType.class));
        verify(action).queue(any(Consumer.class), any(Consumer.class));
    }

    @Test
    void contentAtTheLimitIsPreserved() {
        String content = "a".repeat(Message.MAX_CONTENT_LENGTH);

        assertEquals(content, DiscordWebhookSender.limitToDiscordContent(content));
    }

    @Test
    void contentBeyondTheLimitUsesOneEllipsisWithoutSplittingEmoji() {
        String content = "😀".repeat(Message.MAX_CONTENT_LENGTH + 1);
        String limited = DiscordWebhookSender.limitToDiscordContent(content);

        assertEquals(Message.MAX_CONTENT_LENGTH, limited.codePointCount(0, limited.length()));
        assertEquals("…", limited.substring(limited.offsetByCodePoints(0, Message.MAX_CONTENT_LENGTH - 1)));
    }
}
