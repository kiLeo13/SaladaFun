package sld.saladafun.discordutils.discord;

import java.util.Objects;
import java.util.function.BooleanSupplier;
import java.util.function.Consumer;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.events.message.MessageReceivedEvent;
import net.dv8tion.jda.api.hooks.ListenerAdapter;

/** Filters JDA messages and reduces accepted messages to JDA-free values. */
final class DiscordMessageListener extends ListenerAdapter {
    private final String channelId;
    private final BooleanSupplier active;
    private final Consumer<DiscordInboundMessage> destination;

    /** Creates a listener for the configured Discord text channel. */
    DiscordMessageListener(
        String channelId,
        BooleanSupplier active,
        Consumer<DiscordInboundMessage> destination
    ) {
        this.channelId = Objects.requireNonNull(channelId, "channelId");
        this.active = Objects.requireNonNull(active, "active");
        this.destination = Objects.requireNonNull(destination, "destination");
    }

    /** Forwards accepted, visible messages after copying all JDA-owned values. */
    @Override
    public void onMessageReceived(MessageReceivedEvent event) {
        if (!accepts(event)) {
            return;
        }

        DiscordInboundMessage inbound = copyInboundMessage(event);
        if (inbound.hasVisibleContent()) {
            destination.accept(inbound);
        }
    }

    private boolean accepts(MessageReceivedEvent event) {
        return active.getAsBoolean()
            && channelId.equals(event.getChannel().getId())
            && !event.getAuthor().isBot()
            && !event.isWebhookMessage();
    }

    private DiscordInboundMessage copyInboundMessage(MessageReceivedEvent event) {
        Message message = event.getMessage();
        int imageCount = Math.toIntExact(
            message.getAttachments().stream().filter(Message.Attachment::isImage).count()
        );
        return new DiscordInboundMessage(
            event.getAuthor().getEffectiveName(),
            message.getContentDisplay(),
            imageCount,
            message.getStickers().size()
        );
    }
}
