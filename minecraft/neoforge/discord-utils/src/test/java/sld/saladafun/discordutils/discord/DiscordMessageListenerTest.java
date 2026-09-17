package sld.saladafun.discordutils.discord;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.entities.User;
import net.dv8tion.jda.api.entities.channel.unions.MessageChannelUnion;
import net.dv8tion.jda.api.events.message.MessageReceivedEvent;
import org.junit.jupiter.api.Test;

/** Verifies the single inbound gateway dispatches commands before chat. */
class DiscordMessageListenerTest {
    @Test
    void registeredCommandIsConsumedBeforeMinecraftBroadcast() {
        AtomicInteger commandCalls = new AtomicInteger();
        List<DiscordInboundMessage> broadcasts = new ArrayList<>();
        DiscordMessageListener listener = listener(commandCalls, broadcasts);

        listener.onMessageReceived(event("minecraft", " !PLAYERS ", false, false));

        assertEquals(1, commandCalls.get());
        assertEquals(List.of(), broadcasts);
    }

    @Test
    void unknownCommandFallsThroughToMinecraftBroadcast() {
        AtomicInteger commandCalls = new AtomicInteger();
        List<DiscordInboundMessage> broadcasts = new ArrayList<>();
        DiscordMessageListener listener = listener(commandCalls, broadcasts);

        listener.onMessageReceived(event("minecraft", "!unknown", false, false));

        assertEquals(0, commandCalls.get());
        assertEquals("!unknown", broadcasts.getFirst().content());
    }

    @Test
    void wrongChannelBotsAndWebhooksCannotDispatchCommands() {
        AtomicInteger commandCalls = new AtomicInteger();
        List<DiscordInboundMessage> broadcasts = new ArrayList<>();
        DiscordMessageListener listener = listener(commandCalls, broadcasts);

        listener.onMessageReceived(event("other", "!players", false, false));
        listener.onMessageReceived(event("minecraft", "!players", true, false));
        listener.onMessageReceived(event("minecraft", "!players", false, true));

        assertEquals(0, commandCalls.get());
        assertEquals(List.of(), broadcasts);
    }

    @Test
    void resolvedReplyReferenceIsCopiedForMinecraftRendering() {
        AtomicInteger commandCalls = new AtomicInteger();
        List<DiscordInboundMessage> broadcasts = new ArrayList<>();
        DiscordMessageListener listener = listener(commandCalls, broadcasts);
        Message referencedMessage = referencedMessage("Lucas", "Hello, guys");

        listener.onMessageReceived(event(
            "minecraft",
            "Oiee, Lucas",
            false,
            false,
            referencedMessage
        ));

        DiscordReplyReference reference = broadcasts.getFirst().replyReference().orElseThrow();
        assertEquals("Lucas", reference.authorName());
        assertEquals("Hello, guys", reference.content());
        assertEquals(0, reference.imageCount());
        assertEquals(0, reference.stickerCount());
    }

    @Test
    void unavailableReplyReferenceDoesNotDiscardMessage() {
        AtomicInteger commandCalls = new AtomicInteger();
        List<DiscordInboundMessage> broadcasts = new ArrayList<>();
        DiscordMessageListener listener = listener(commandCalls, broadcasts);

        listener.onMessageReceived(event("minecraft", "Still visible", false, false, null));

        assertEquals(1, broadcasts.size());
        assertTrue(broadcasts.getFirst().replyReference().isEmpty());
    }

    private static DiscordMessageListener listener(
        AtomicInteger commandCalls,
        List<DiscordInboundMessage> broadcasts
    ) {
        DiscordMessageCommandRegistry commands = new DiscordMessageCommandRegistry();
        commands.register("!players", request -> commandCalls.incrementAndGet());
        commands.freeze();
        return new DiscordMessageListener("minecraft", () -> true, commands, broadcasts::add);
    }

    private static MessageReceivedEvent event(
        String channelId,
        String content,
        boolean bot,
        boolean webhook
    ) {
        return event(channelId, content, bot, webhook, null);
    }

    private static MessageReceivedEvent event(
        String channelId,
        String content,
        boolean bot,
        boolean webhook,
        Message referencedMessage
    ) {
        MessageReceivedEvent event = mock(MessageReceivedEvent.class);
        MessageChannelUnion channel = mock(MessageChannelUnion.class);
        User author = mock(User.class);
        Message message = mock(Message.class);
        when(event.getChannel()).thenReturn(channel);
        when(channel.getId()).thenReturn(channelId);
        when(event.getAuthor()).thenReturn(author);
        when(author.isBot()).thenReturn(bot);
        when(author.getEffectiveName()).thenReturn("DiscordUser");
        when(event.isWebhookMessage()).thenReturn(webhook);
        when(event.getMessage()).thenReturn(message);
        when(message.getContentRaw()).thenReturn(content);
        when(message.getContentDisplay()).thenReturn(content);
        when(message.getAttachments()).thenReturn(List.of());
        when(message.getStickers()).thenReturn(List.of());
        when(message.getReferencedMessage()).thenReturn(referencedMessage);
        return event;
    }

    private static Message referencedMessage(String authorName, String content) {
        Message message = mock(Message.class);
        User author = mock(User.class);
        when(message.getAuthor()).thenReturn(author);
        when(author.getEffectiveName()).thenReturn(authorName);
        when(message.getContentDisplay()).thenReturn(content);
        when(message.getAttachments()).thenReturn(List.of());
        when(message.getStickers()).thenReturn(List.of());
        return message;
    }
}
