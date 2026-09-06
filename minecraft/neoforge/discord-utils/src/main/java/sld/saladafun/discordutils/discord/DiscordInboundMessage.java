package sld.saladafun.discordutils.discord;

import java.util.Objects;

/** JDA-free representation of a Discord message supported by the bridge. */
public record DiscordInboundMessage(
    String authorName,
    String content,
    int imageCount,
    int stickerCount
) {
    /** Validates copied message content and supported-media counters. */
    public DiscordInboundMessage {
        Objects.requireNonNull(authorName, "authorName");
        Objects.requireNonNull(content, "content");

        if (imageCount < 0 || stickerCount < 0) {
            throw new IllegalArgumentException("media counts cannot be negative");
        }
    }

    /** Returns whether Minecraft has any supported content to render. */
    public boolean hasVisibleContent() {
        return !content.isBlank() || imageCount > 0 || stickerCount > 0;
    }
}
