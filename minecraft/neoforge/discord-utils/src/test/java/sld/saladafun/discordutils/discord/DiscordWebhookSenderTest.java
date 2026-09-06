package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;

import net.dv8tion.jda.api.entities.Message;
import org.junit.jupiter.api.Test;

/** Tests Discord content limiting independently of webhook delivery. */
class DiscordWebhookSenderTest {
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
