package sld.saladafun.discordutils.discord;

import java.util.Objects;

/** JDA-free snapshot of the Discord message referenced by an inbound reply. */
public record DiscordReplyReference(
    String authorName,
    String content,
    int imageCount,
    int stickerCount
) {
    /** Validates copied reference content and supported-media counters. */
    public DiscordReplyReference {
        Objects.requireNonNull(authorName, "authorName");
        Objects.requireNonNull(content, "content");

        if (imageCount < 0 || stickerCount < 0) {
            throw new IllegalArgumentException("media counts cannot be negative");
        }
    }
}
