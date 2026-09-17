package sld.saladafun.discordutils.discord;

import java.util.Objects;
import java.util.Optional;

/** JDA-free representation of a Discord message supported by the bridge. */
public record DiscordInboundMessage(
    String authorName,
    String content,
    int imageCount,
    int stickerCount,
    Optional<DiscordReplyReference> replyReference
) {
    /** Creates an ordinary inbound message without a Discord reply reference. */
    public DiscordInboundMessage(
        String authorName,
        String content,
        int imageCount,
        int stickerCount
    ) {
        this(authorName, content, imageCount, stickerCount, Optional.empty());
    }

    /** Validates copied message content and supported-media counters. */
    public DiscordInboundMessage {
        Objects.requireNonNull(authorName, "authorName");
        Objects.requireNonNull(content, "content");
        Objects.requireNonNull(replyReference, "replyReference");

        if (imageCount < 0 || stickerCount < 0) {
            throw new IllegalArgumentException("media counts cannot be negative");
        }
    }

    /** Returns whether Minecraft has any supported content to render. */
    public boolean hasVisibleContent() {
        return !content.isBlank() || imageCount > 0 || stickerCount > 0;
    }
}
