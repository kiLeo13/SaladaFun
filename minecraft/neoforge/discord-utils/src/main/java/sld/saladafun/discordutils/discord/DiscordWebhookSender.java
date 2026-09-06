package sld.saladafun.discordutils.discord;

import java.util.EnumSet;
import java.util.Objects;
import java.util.UUID;
import net.dv8tion.jda.api.entities.IncomingWebhookClient;
import net.dv8tion.jda.api.entities.Message;
import org.slf4j.Logger;

/** Delivers Minecraft messages through an incoming Discord webhook. */
final class DiscordWebhookSender {
    private static final String HEAD_AVATAR_BASE_URL = "https://api.mcheads.org/head/";
    private static final int HEAD_AVATAR_SIZE_PIXELS = 128;
    private static final String HEAD_OUTER_LAYER_PATH = "hat";

    private final IncomingWebhookClient webhook;
    private final Logger logger;

    /** Creates a sender around one JDA webhook client. */
    DiscordWebhookSender(IncomingWebhookClient webhook, Logger logger) {
        this.webhook = Objects.requireNonNull(webhook, "webhook");
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    /** Queues a mention-safe outbound message with the player's head avatar. */
    void send(String playerName, UUID playerId, String content) {
        if (content.isBlank()) {
            return;
        }

        webhook.sendMessage(limitToDiscordContent(content))
            .setUsername(playerName)
            .setAvatarUrl(headAvatarUrl(playerId))
            .setAllowedMentions(EnumSet.noneOf(Message.MentionType.class))
            .queue(
                ignored -> { },
                failure -> logger.warn(
                    "Could not deliver Minecraft chat through the Discord webhook ({})",
                    failure.getClass().getSimpleName()
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

    /** Limits content to Discord's code-point limit without splitting surrogate pairs. */
    static String limitToDiscordContent(String content) {
        int codePointCount = content.codePointCount(0, content.length());
        if (codePointCount <= Message.MAX_CONTENT_LENGTH) {
            return content;
        }

        int end = content.offsetByCodePoints(0, Message.MAX_CONTENT_LENGTH - 1);
        return content.substring(0, end) + "…";
    }
}
