package sld.saladafun.platform.purpur.discord;

import net.dv8tion.jda.api.entities.Member;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.entities.User;
import net.dv8tion.jda.api.entities.channel.unions.MessageChannelUnion;
import net.dv8tion.jda.api.entities.sticker.StickerItem;
import net.dv8tion.jda.api.events.message.MessageReceivedEvent;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class DiscordMessageListenerTest {
    private static final String CHANNEL_ID = "987654321098765432";

    @Test
    void forwardsDisplayContentAndCountsImagesAndStickers() {
        AtomicReference<DiscordInboundMessage> delivered = new AtomicReference<>();
        MessageReceivedEvent event = textEvent(CHANNEL_ID, false, false);
        Message message = event.getMessage();
        Member member = mock(Member.class);
        when(member.getEffectiveName()).thenReturn("Server Nickname");
        when(event.getMember()).thenReturn(member);
        when(message.getContentDisplay()).thenReturn("hello @VisibleName");

        Message.Attachment firstImage = attachment(true);
        Message.Attachment secondImage = attachment(true);
        Message.Attachment document = attachment(false);
        when(message.getAttachments()).thenReturn(
            List.of(firstImage, secondImage, document)
        );
        when(message.getStickers()).thenReturn(
            List.of(mock(StickerItem.class))
        );

        new DiscordMessageListener(CHANNEL_ID, () -> true, delivered::set)
            .onMessageReceived(event);

        assertEquals(
            new DiscordInboundMessage("Server Nickname", "hello @VisibleName", 2, 1),
            delivered.get()
        );
        verify(message).getContentDisplay();
        verify(message, never()).getContentRaw();
    }

    @Test
    void forwardsMediaOnlyMessagesUsingTheUserNameFallback() {
        AtomicReference<DiscordInboundMessage> delivered = new AtomicReference<>();
        MessageReceivedEvent event = textEvent(CHANNEL_ID, false, false);
        when(event.getMember()).thenReturn(null);
        when(event.getAuthor().getEffectiveName()).thenReturn("Discord User");
        when(event.getMessage().getContentDisplay()).thenReturn("");
        Message.Attachment image = attachment(true);
        when(event.getMessage().getAttachments()).thenReturn(List.of(image));
        when(event.getMessage().getStickers()).thenReturn(List.of());

        new DiscordMessageListener(CHANNEL_ID, () -> true, delivered::set)
            .onMessageReceived(event);

        assertEquals(
            new DiscordInboundMessage("Discord User", "", 1, 0),
            delivered.get()
        );
    }

    @Test
    void ignoresInactiveWrongChannelBotAndWebhookMessages() {
        AtomicReference<DiscordInboundMessage> delivered = new AtomicReference<>();
        DiscordMessageListener inactive = new DiscordMessageListener(
            CHANNEL_ID,
            () -> false,
            delivered::set
        );
        inactive.onMessageReceived(textEvent(CHANNEL_ID, false, false));

        DiscordMessageListener active = new DiscordMessageListener(
            CHANNEL_ID,
            () -> true,
            delivered::set
        );
        active.onMessageReceived(textEvent("111111111111111111", false, false));
        active.onMessageReceived(textEvent(CHANNEL_ID, true, false));
        active.onMessageReceived(textEvent(CHANNEL_ID, false, true));

        assertNull(delivered.get());
    }

    private MessageReceivedEvent textEvent(
        String channelId,
        boolean bot,
        boolean webhook
    ) {
        MessageReceivedEvent event = mock(MessageReceivedEvent.class);
        MessageChannelUnion channel = mock(MessageChannelUnion.class);
        User author = mock(User.class);
        Message message = mock(Message.class);
        when(channel.getId()).thenReturn(channelId);
        when(author.isBot()).thenReturn(bot);
        when(event.getChannel()).thenReturn(channel);
        when(event.getAuthor()).thenReturn(author);
        when(event.isWebhookMessage()).thenReturn(webhook);
        when(event.getMessage()).thenReturn(message);
        return event;
    }

    private Message.Attachment attachment(boolean image) {
        Message.Attachment attachment = mock(Message.Attachment.class);
        when(attachment.isImage()).thenReturn(image);
        return attachment;
    }
}
