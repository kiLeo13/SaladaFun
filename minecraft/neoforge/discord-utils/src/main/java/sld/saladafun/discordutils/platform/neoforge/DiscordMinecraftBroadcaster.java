package sld.saladafun.discordutils.platform.neoforge;

import java.util.Objects;
import java.util.function.Consumer;
import net.minecraft.ChatFormatting;
import net.minecraft.network.chat.Component;
import net.minecraft.network.chat.MutableComponent;
import net.minecraft.server.MinecraftServer;
import sld.saladafun.discordutils.discord.DiscordInboundMessage;

/** Delivers accepted Discord messages to online Minecraft players on the server thread. */
public final class DiscordMinecraftBroadcaster implements Consumer<DiscordInboundMessage> {
    private static final ChatFormatting DISCORD_BLUE = ChatFormatting.BLUE;
    private static final ChatFormatting MESSAGE_GRAY = ChatFormatting.GRAY;
    private static final ChatFormatting MEDIA_PURPLE = ChatFormatting.LIGHT_PURPLE;

    private final MinecraftServer server;

    /** Creates a broadcaster for the currently running dedicated server. */
    public DiscordMinecraftBroadcaster(MinecraftServer server) {
        this.server = Objects.requireNonNull(server, "server");
    }

    /** Schedules one Discord message for delivery to every online player. */
    @Override
    public void accept(DiscordInboundMessage message) {
        Objects.requireNonNull(message, "message");
        Component renderedMessage = render(message);
        server.execute(() -> server.getPlayerList().broadcastSystemMessage(renderedMessage, false));
    }

    private Component render(DiscordInboundMessage message) {
        MutableComponent rendered = Component.literal("[Discord] ").withStyle(DISCORD_BLUE)
            .append(Component.literal(message.authorName()).withStyle(ChatFormatting.WHITE));

        if (message.hasVisibleContent()) {
            rendered.append(Component.literal(": ").withStyle(MESSAGE_GRAY));
            rendered.append(Component.literal(message.content()).withStyle(MESSAGE_GRAY));
        }

        appendMediaSummary(rendered, message);
        return rendered;
    }

    private void appendMediaSummary(MutableComponent rendered, DiscordInboundMessage message) {
        if (message.imageCount() == 0 && message.stickerCount() == 0) {
            return;
        }

        MutableComponent summary = Component.empty();
        if (message.imageCount() > 0) {
            summary.append(Component.literal(message.imageCount() + " image(s)"));
        }
        if (message.stickerCount() > 0) {
            if (!summary.getString().isEmpty()) {
                summary.append(Component.literal(", "));
            }
            summary.append(Component.literal(message.stickerCount() + " sticker(s)"));
        }

        rendered.append(Component.literal("\n["));
        rendered.append(summary.withStyle(MEDIA_PURPLE));
        rendered.append(Component.literal("]"));
    }
}
