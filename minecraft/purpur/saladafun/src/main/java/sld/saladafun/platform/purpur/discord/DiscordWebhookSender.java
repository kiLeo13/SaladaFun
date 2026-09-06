package sld.saladafun.platform.purpur.discord;

import net.dv8tion.jda.api.entities.IncomingWebhookClient;
import net.dv8tion.jda.api.entities.Message;

import java.util.EnumSet;
import java.util.Objects;
import java.util.UUID;
import java.util.logging.Logger;

final class DiscordWebhookSender {
    private static final String HEAD_AVATAR_BASE_URL = "https://api.mcheads.org/head/";
    private static final int HEAD_AVATAR_SIZE_PIXELS = 128;
    private static final String HEAD_OUTER_LAYER_PATH = "hat";

    private final IncomingWebhookClient webhook;
    private final Logger logger;

    DiscordWebhookSender(IncomingWebhookClient webhook, Logger logger) {
        this.webhook = Objects.requireNonNull(webhook, "webhook");
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    void send(String playerName, UUID playerId, String content) {
        if (content.isBlank()) {
            return;
        }

        webhook.sendMessage(limitToDiscordContent(content))
            .setUsername(playerName)
            .setAvatarUrl(headAvatarUrl(playerId))
            .setAllowedMentions(EnumSet.noneOf(Message.MentionType.class))
            .queue(
                ignored -> {
                    // Delivery succeeded; there is nothing useful to retain.
                },
                failure -> logger.warning(
                    "Could not deliver Minecraft chat through the Discord webhook ("
                        + failure.getClass().getSimpleName() + ")"
                )
            );
    }

    private static String headAvatarUrl(UUID playerId) {
        return HEAD_AVATAR_BASE_URL
            + playerId
            + "/"
            + HEAD_AVATAR_SIZE_PIXELS
            + "/"
            + HEAD_OUTER_LAYER_PATH;
    }

    private String limitToDiscordContent(String content) {
        int codePoints = content.codePointCount(0, content.length());
        if (codePoints <= Message.MAX_CONTENT_LENGTH) {
            return content;
        }
        int end = content.offsetByCodePoints(0, Message.MAX_CONTENT_LENGTH - 1);
        return content.substring(0, end) + "…";
    }
}
