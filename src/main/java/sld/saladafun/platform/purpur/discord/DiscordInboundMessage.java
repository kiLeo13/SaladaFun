package sld.saladafun.platform.purpur.discord;

import java.util.Objects;

/**
 * Cache-independent snapshot of a Discord message for delivery to Minecraft.
 */
public record DiscordInboundMessage(
    String authorName,
    String content,
    int imageCount,
    int stickerCount
) {
    public DiscordInboundMessage {
        Objects.requireNonNull(authorName, "authorName");
        Objects.requireNonNull(content, "content");
        if (imageCount < 0 || stickerCount < 0) {
            throw new IllegalArgumentException("Media counts cannot be negative");
        }
    }

    public boolean hasVisibleContent() {
        return !content.isBlank() || imageCount > 0 || stickerCount > 0;
    }
}
