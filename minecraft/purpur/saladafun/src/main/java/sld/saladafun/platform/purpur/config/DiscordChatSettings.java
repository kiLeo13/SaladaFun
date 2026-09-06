package sld.saladafun.platform.purpur.config;

import java.util.Objects;

/**
 * Immutable Discord bridge configuration. Its string representation deliberately
 * redacts both credentials.
 */
public record DiscordChatSettings(
    boolean enabled,
    String token,
    String webhookUrl,
    String channelId
) {
    private static final DiscordChatSettings DISABLED =
        new DiscordChatSettings(false, "", "", "");

    public DiscordChatSettings {
        Objects.requireNonNull(token, "token");
        Objects.requireNonNull(webhookUrl, "webhookUrl");
        Objects.requireNonNull(channelId, "channelId");
    }

    public static DiscordChatSettings disabled() {
        return DISABLED;
    }

    @Override
    public String toString() {
        return "DiscordChatSettings[enabled=%s, token=<redacted>, webhookUrl=<redacted>, channelId=%s]"
            .formatted(enabled, channelId);
    }
}
