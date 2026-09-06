package sld.saladafun.platform.purpur.discord;

import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.events.message.MessageReceivedEvent;
import net.dv8tion.jda.api.hooks.ListenerAdapter;

import java.util.Objects;
import java.util.function.BooleanSupplier;
import java.util.function.Consumer;

/**
 * Reduces a JDA event to the data Minecraft needs without retaining JDA entities.
 */
public final class DiscordMessageListener extends ListenerAdapter {
    private final String channelId;
    private final BooleanSupplier active;
    private final Consumer<DiscordInboundMessage> destination;

    public DiscordMessageListener(
        String channelId,
        BooleanSupplier active,
        Consumer<DiscordInboundMessage> destination
    ) {
        this.channelId = Objects.requireNonNull(channelId, "channelId");
        this.active = Objects.requireNonNull(active, "active");
        this.destination = Objects.requireNonNull(destination, "destination");
    }

    @Override
    public void onMessageReceived(MessageReceivedEvent event) {
        if (!active.getAsBoolean()
            || !channelId.equals(event.getChannel().getId())
            || event.getAuthor().isBot()
            || event.isWebhookMessage()) {
            return;
        }

        Message message = event.getMessage();
        String content = message.getContentDisplay();
        int imageCount = Math.toIntExact(
            message.getAttachments().stream().filter(Message.Attachment::isImage).count()
        );
        int stickerCount = message.getStickers().size();
        DiscordInboundMessage inbound = new DiscordInboundMessage(
            event.getAuthor().getEffectiveName(),
            content,
            imageCount,
            stickerCount
        );
        if (inbound.hasVisibleContent()) {
            destination.accept(inbound);
        }
    }
}
