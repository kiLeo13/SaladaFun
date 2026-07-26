package sld.saladafun.platform.purpur.config;

import org.bukkit.configuration.file.FileConfiguration;
import sld.saladafun.batchbreaking.BatchBlockAction;
import sld.saladafun.batchbreaking.BatchExecutionMode;
import sld.saladafun.batchbreaking.BatchBreakingSetting;
import sld.saladafun.batchbreaking.BatchBreakingSettingParser;
import sld.saladafun.batchbreaking.ToolDurabilityMode;

import java.util.Locale;
import java.util.Objects;

/**
 * Validated view of administrator-controlled plugin configuration.
 */
public final class PluginSettings {
    private FileConfiguration configuration;
    private final BatchBreakingSettingParser batchParser = new BatchBreakingSettingParser();

    public PluginSettings(FileConfiguration configuration) {
        this.configuration = Objects.requireNonNull(configuration, "configuration");
    }

    public void validate() {
        deathBehavior();
        respectItemsToKeep();
        batchBreakingSetting();
        batchBlockAction();
        toolDurabilityMode();
        batchExecutionMode();
        includeAnimals();
    }

    public void replace(FileConfiguration candidate) {
        PluginSettings validated = new PluginSettings(candidate);
        validated.validate();
        configuration = Objects.requireNonNull(candidate, "candidate");
    }

    public DeathBehavior deathBehavior() {
        String configured = configuration.getString(
            "shared-inventory.death-behavior",
            DeathBehavior.FOLLOW_GAMERULE.name()
        );
        try {
            return DeathBehavior.valueOf(configured.toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException exception) {
            throw new IllegalArgumentException(
                "Invalid shared-inventory.death-behavior: " + configured,
                exception
            );
        }
    }

    public boolean respectItemsToKeep() {
        return configuration.getBoolean("shared-inventory.respect-items-to-keep", true);
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
}
