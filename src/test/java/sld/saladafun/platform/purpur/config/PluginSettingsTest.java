package sld.saladafun.platform.purpur.config;

import org.bukkit.configuration.file.YamlConfiguration;
import org.junit.jupiter.api.Test;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.BatchExecutionMode;
import sld.saladafun.batchbreaking.ToolDurabilityMode;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assertions.assertThrows;

class PluginSettingsTest {

    @Test
    void defaultsToSafeBatchPolicies() {
        PluginSettings settings = new PluginSettings(new YamlConfiguration());

        assertEquals(BatchBlockAction.PLAYER_TOOL, settings.batchBlockAction());
        assertEquals(ToolDurabilityMode.SINGLE_USE, settings.toolDurabilityMode());
        assertEquals(BatchExecutionMode.ASYNC, settings.batchExecutionMode());
        assertFalse(settings.includeAnimals());
    }

    @Test
    void parsesBatchPoliciesCaseInsensitively() {
        YamlConfiguration configuration = new YamlConfiguration();
        configuration.set("batch-breaking.additional-block-action", "natural_drops");
        configuration.set("batch-breaking.tool-durability", "per_block");
        configuration.set("batch-breaking.sync-batching", "sync");
        configuration.set("batch-breaking.include-animals", true);

        PluginSettings settings = new PluginSettings(configuration);

        assertEquals(BatchBlockAction.NATURAL_DROPS, settings.batchBlockAction());
        assertEquals(ToolDurabilityMode.PER_BLOCK, settings.toolDurabilityMode());
        assertEquals(BatchExecutionMode.SYNC, settings.batchExecutionMode());
        assertEquals(true, settings.includeAnimals());
    }

    @Test
    void rejectsUnknownBatchPolicies() {
        YamlConfiguration configuration = new YamlConfiguration();
        configuration.set("batch-breaking.additional-block-action", "explode");
        PluginSettings settings = new PluginSettings(configuration);

        assertThrows(IllegalArgumentException.class, settings::batchBlockAction);

        configuration.set("batch-breaking.additional-block-action", "PLAYER_TOOL");
        configuration.set("batch-breaking.tool-durability", "forever");

        assertThrows(IllegalArgumentException.class, settings::toolDurabilityMode);

        configuration.set("batch-breaking.tool-durability", "SINGLE_USE");
        configuration.set("batch-breaking.sync-batching", "sometimes");

        assertThrows(IllegalArgumentException.class, settings::batchExecutionMode);
    }

    @Test
    void keepsDiscordDisabledWithoutCredentialsByDefault() {
        DiscordChatSettings discord = new PluginSettings(
            new YamlConfiguration()
        ).discordChatSettings();

        assertFalse(discord.enabled());
        assertEquals("", discord.token());
        assertEquals("", discord.webhookUrl());
        assertEquals("", discord.channelId());
    }

    @Test
    void parsesEnabledDiscordSettingsWithoutExposingCredentials() {
        YamlConfiguration configuration = enabledDiscordConfiguration();
        PluginSettings settings = new PluginSettings(configuration);

        DiscordChatSettings discord = settings.discordChatSettings();

        assertTrue(discord.enabled());
        assertEquals("secret.bot.token", discord.token());
        assertEquals(
            "https://discord.com/api/webhooks/123456789012345678/webhook-secret",
            discord.webhookUrl()
        );
        assertEquals("987654321098765432", discord.channelId());
        assertFalse(discord.toString().contains("secret.bot.token"));
        assertFalse(discord.toString().contains("webhook-secret"));
    }

    @Test
    void rejectsIncompleteOrMalformedDiscordSettingsWithoutLeakingSecrets() {
        YamlConfiguration configuration = enabledDiscordConfiguration();
        configuration.set("discord-chat.channel-id", "");
        PluginSettings settings = new PluginSettings(configuration);

        assertThrows(IllegalArgumentException.class, settings::discordChatSettings);

        configuration.set("discord-chat.channel-id", "not-a-snowflake");
        assertThrows(IllegalArgumentException.class, settings::discordChatSettings);

        String maliciousUrl = "https://example.com/api/webhooks/123/leaked-secret";
        configuration.set("discord-chat.channel-id", "987654321098765432");
        configuration.set("discord-chat.webhook-url", maliciousUrl);
        IllegalArgumentException exception = assertThrows(
            IllegalArgumentException.class,
            settings::discordChatSettings
        );
        assertFalse(exception.getMessage().contains(maliciousUrl));
        assertFalse(exception.getMessage().contains("secret.bot.token"));
    }

    private YamlConfiguration enabledDiscordConfiguration() {
        YamlConfiguration configuration = new YamlConfiguration();
        configuration.set("discord-chat.enabled", true);
        configuration.set("discord-chat.token", "secret.bot.token");
        configuration.set(
            "discord-chat.webhook-url",
            "https://discord.com/api/webhooks/123456789012345678/webhook-secret"
        );
        configuration.set("discord-chat.channel-id", "987654321098765432");
        return configuration;
    }
}
