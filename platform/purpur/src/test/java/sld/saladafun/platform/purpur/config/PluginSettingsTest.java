package sld.saladafun.platform.purpur.config;

import org.bukkit.configuration.file.YamlConfiguration;
import org.junit.jupiter.api.Test;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.ToolDurabilityMode;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

class PluginSettingsTest {

    @Test
    void defaultsToPlayerToolAndOneOriginalToolUse() {
        PluginSettings settings = new PluginSettings(new YamlConfiguration());

        assertEquals(BatchBlockAction.PLAYER_TOOL, settings.batchBlockAction());
        assertEquals(ToolDurabilityMode.SINGLE_USE, settings.toolDurabilityMode());
    }

    @Test
    void parsesBatchPoliciesCaseInsensitively() {
        YamlConfiguration configuration = new YamlConfiguration();
        configuration.set("batch-breaking.additional-block-action", "natural_drops");
        configuration.set("batch-breaking.tool-durability", "per_block");

        PluginSettings settings = new PluginSettings(configuration);

        assertEquals(BatchBlockAction.NATURAL_DROPS, settings.batchBlockAction());
        assertEquals(ToolDurabilityMode.PER_BLOCK, settings.toolDurabilityMode());
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
    }
}
