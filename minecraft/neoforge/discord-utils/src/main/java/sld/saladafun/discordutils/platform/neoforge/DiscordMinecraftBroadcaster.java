package sld.saladafun.discordutils.platform.neoforge;

import java.util.Objects;
import java.util.function.Consumer;
import net.minecraft.ChatFormatting;
import net.minecraft.network.chat.Component;
import net.minecraft.network.chat.MutableComponent;
import net.minecraft.server.MinecraftServer;
import sld.saladafun.discordutils.discord.DiscordInboundMessage;
import sld.saladafun.discordutils.discord.DiscordReplyReference;

/** Delivers accepted Discord messages to online Minecraft players on the server thread. */
public final class DiscordMinecraftBroadcaster implements Consumer<DiscordInboundMessage> {
    private static final ChatFormatting DISCORD_BLUE = ChatFormatting.BLUE;
    private static final ChatFormatting MESSAGE_GRAY = ChatFormatting.GRAY;
    private static final ChatFormatting MEDIA_PURPLE = ChatFormatting.LIGHT_PURPLE;
    private static final ChatFormatting REPLY_DARK_GRAY = ChatFormatting.DARK_GRAY;

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

    /** Builds the complete Minecraft component for one Discord message. */
    static Component render(DiscordInboundMessage message) {
        MutableComponent rendered = Component.empty();
        message.replyReference().ifPresent(reference -> appendReplyReference(rendered, reference));

        rendered.append(Component.literal("[Discord] ").withStyle(DISCORD_BLUE)
            .append(Component.literal(message.authorName()).withStyle(ChatFormatting.WHITE)));

        if (message.hasVisibleContent()) {
            rendered.append(Component.literal(": ").withStyle(MESSAGE_GRAY));
            rendered.append(Component.literal(message.content()).withStyle(MESSAGE_GRAY));
        }

        appendMediaSummary(rendered, message);
        return rendered;
    }

    /** Appends one subdued line describing the referenced Discord message. */
    private static void appendReplyReference(
        MutableComponent rendered,
        DiscordReplyReference reference
    ) {
        String authorName = normalizeWhitespace(reference.authorName());
        String preview = normalizeWhitespace(reference.content());
        if (preview.isEmpty()) {
            preview = mediaSummary(reference.imageCount(), reference.stickerCount());
        }

        String suffix = preview.isEmpty() ? "" : ": " + preview;
        rendered.append(Component.literal(
            "┃ respondendo a " + authorName + suffix + "\n"
        ).withStyle(REPLY_DARK_GRAY));
    }

    /** Appends supported media counts beneath the main Discord message. */
    private static void appendMediaSummary(MutableComponent rendered, DiscordInboundMessage message) {
        if (message.imageCount() == 0 && message.stickerCount() == 0) {
            return;
        }

        rendered.append(Component.literal("\n["));
        rendered.append(Component.literal(
            mediaSummary(message.imageCount(), message.stickerCount())
        ).withStyle(MEDIA_PURPLE));
        rendered.append(Component.literal("]"));
    }

    /** Formats supported Discord media counts for chat display. */
    private static String mediaSummary(int imageCount, int stickerCount) {
        StringBuilder summary = new StringBuilder();
        if (imageCount > 0) {
            summary.append(imageCount).append(" image(s)");
        }
        if (stickerCount > 0) {
            if (!summary.isEmpty()) {
                summary.append(", ");
            }
            summary.append(stickerCount).append(" sticker(s)");
        }
        return summary.toString();
    }

    /** Collapses Discord text into a compact single-line reply preview. */
    private static String normalizeWhitespace(String value) {
        return value.strip().replaceAll("\\s+", " ");
    }
}
