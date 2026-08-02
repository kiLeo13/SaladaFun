package sld.saladafun.platform.purpur.config;

import org.bukkit.configuration.file.FileConfiguration;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.BatchExecutionMode;
import sld.saladafun.batchbreaking.BatchBreakingSetting;
import sld.saladafun.batchbreaking.BatchBreakingSettingParser;
import sld.saladafun.batchbreaking.ToolDurabilityMode;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.Locale;
import java.util.Objects;
import java.util.Set;

/**
 * Validated view of administrator-controlled plugin configuration.
 */
public final class PluginSettings {
    private static final Set<String> DISCORD_WEBHOOK_HOSTS = Set.of(
        "discord.com",
        "ptb.discord.com",
        "canary.discord.com",
        "discordapp.com",
        "ptb.discordapp.com",
        "canary.discordapp.com"
    );

    private FileConfiguration configuration;
    private final BatchBreakingSettingParser batchParser = new BatchBreakingSettingParser();

    public PluginSettings(FileConfiguration configuration) {
        this.configuration = Objects.requireNonNull(configuration, "configuration");
    }

    public void validate() {
        batchBreakingSetting();
        batchBlockAction();
        toolDurabilityMode();
        batchExecutionMode();
        includeAnimals();
        discordChatSettings();
    }

    public void replace(FileConfiguration candidate) {
        PluginSettings validated = new PluginSettings(candidate);
        validated.validate();
        configuration = Objects.requireNonNull(candidate, "candidate");
    }

    public BatchBreakingSetting batchBreakingSetting() {
        return batchParser.parse(
            configuration.getString("batch-breaking.setting", "disabled")
        );
    }

    public BatchBlockAction batchBlockAction() {
        return enumSetting(
            "batch-breaking.additional-block-action",
            BatchBlockAction.PLAYER_TOOL
        );
    }

    public ToolDurabilityMode toolDurabilityMode() {
        return enumSetting(
            "batch-breaking.tool-durability",
            ToolDurabilityMode.SINGLE_USE
        );
    }

    public BatchExecutionMode batchExecutionMode() {
        return enumSetting(
            "batch-breaking.sync-batching",
            BatchExecutionMode.ASYNC
        );
    }

    public boolean includeAnimals() {
        return configuration.getBoolean("batch-breaking.include-animals", false);
    }

    public DiscordChatSettings discordChatSettings() {
        if (!configuration.getBoolean("discord-chat.enabled", false)) {
            return DiscordChatSettings.disabled();
        }

        String token = requiredString("discord-chat.token");
        String webhookUrl = requiredString("discord-chat.webhook-url");
        String channelId = requiredString("discord-chat.channel-id");
        validateDiscordWebhookUrl(webhookUrl);
        validateDiscordSnowflake(channelId);
        return new DiscordChatSettings(true, token, webhookUrl, channelId);
    }

    public void batchBreakingSetting(BatchBreakingSetting setting) {
        configuration.set("batch-breaking.setting", batchParser.format(setting));
    }

    private <E extends Enum<E>> E enumSetting(String path, E defaultValue) {
        String configured = configuration.getString(path, defaultValue.name());
        try {
            return Enum.valueOf(
                defaultValue.getDeclaringClass(),
                configured.toUpperCase(Locale.ROOT)
            );
        } catch (IllegalArgumentException exception) {
            throw new IllegalArgumentException(
                "Invalid " + path + ": " + configured,
                exception
            );
        }
    }

    private String requiredString(String path) {
        String configured = configuration.getString(path, "");
        if (configured == null || configured.isBlank()) {
            throw new IllegalArgumentException("Missing required " + path);
        }
        return configured.strip();
    }

    private void validateDiscordWebhookUrl(String configured) {
        try {
            URI uri = new URI(configured);
            String host = uri.getHost();
            String path = uri.getPath();
            if (!"https".equalsIgnoreCase(uri.getScheme())
                || host == null
                || !DISCORD_WEBHOOK_HOSTS.contains(host.toLowerCase(Locale.ROOT))
                || path == null
                || !path.matches("/api(?:/v\\d+)?/webhooks/\\d+/[^/]+/?")) {
                throw invalidDiscordWebhookUrl();
            }
        } catch (URISyntaxException exception) {
            throw invalidDiscordWebhookUrl();
        }
    }

    private void validateDiscordSnowflake(String configured) {
        try {
            if (Long.parseUnsignedLong(configured) == 0L) {
                throw new NumberFormatException("zero");
            }
        } catch (NumberFormatException exception) {
            throw new IllegalArgumentException(
                "Invalid discord-chat.channel-id: expected a non-zero Discord snowflake",
                exception
            );
        }
    }

    private IllegalArgumentException invalidDiscordWebhookUrl() {
        return new IllegalArgumentException(
            "Invalid discord-chat.webhook-url: expected an HTTPS Discord incoming webhook URL"
        );
    }
}
