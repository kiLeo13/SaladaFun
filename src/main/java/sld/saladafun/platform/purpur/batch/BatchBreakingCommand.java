package sld.saladafun.platform.purpur.batch;

import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.command.TabCompleter;
import org.bukkit.plugin.java.JavaPlugin;
import sld.saladafun.batchbreaking.BatchBreakingSettingParser;
import sld.saladafun.platform.purpur.config.PluginSettings;

import java.util.List;
import java.util.Locale;
import java.util.Objects;

/**
 * Human-readable configuration command for batch breaking.
 */
public final class BatchBreakingCommand implements CommandExecutor, TabCompleter {
    private final JavaPlugin plugin;
    private final PluginSettings settings;
    private final BatchBreakingSettingParser parser = new BatchBreakingSettingParser();

    public BatchBreakingCommand(JavaPlugin plugin, PluginSettings settings) {
        this.plugin = Objects.requireNonNull(plugin, "plugin");
        this.settings = Objects.requireNonNull(settings, "settings");
    }

    @Override
    public boolean onCommand(
        CommandSender sender,
        Command command,
        String label,
        String[] arguments
    ) {
        if (arguments.length == 1 && arguments[0].equalsIgnoreCase("get")) {
            sender.sendMessage(
                "Batch breaking is set to "
                    + parser.format(settings.batchBreakingSetting())
                    + " with " + settings.batchBlockAction()
                    + ", " + settings.toolDurabilityMode() + " durability, "
                    + settings.batchExecutionMode() + " batching, and animals "
                    + (settings.includeAnimals() ? "included." : "excluded.")
            );
            return true;
        }
        if (arguments.length == 2 && arguments[0].equalsIgnoreCase("set")) {
            try {
                var parsed = parser.parse(arguments[1]);
                settings.batchBreakingSetting(parsed);
                plugin.saveConfig();
                sender.sendMessage(
                    "Batch breaking set to " + parser.format(parsed) + "."
                );
            } catch (IllegalArgumentException exception) {
                sender.sendMessage(exception.getMessage());
            }
            return true;
        }
        sender.sendMessage(
            "Usage: /batchbreaking get or /batchbreaking set <disabled|all|positive integer>"
        );
        return true;
    }

    @Override
    public List<String> onTabComplete(
        CommandSender sender,
        Command command,
        String alias,
        String[] arguments
    ) {
        if (arguments.length == 1) {
            return filter(List.of("get", "set"), arguments[0]);
        }
        if (arguments.length == 2 && arguments[0].equalsIgnoreCase("set")) {
            return filter(List.of("disabled", "all", "1", "2", "5"), arguments[1]);
        }
        return List.of();
    }

    private List<String> filter(List<String> options, String prefix) {
        String normalized = prefix.toLowerCase(Locale.ROOT);
        return options.stream()
            .filter(option -> option.startsWith(normalized))
            .toList();
    }
}
