package sld.saladafun.platform.purpur.shared;

import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.command.TabCompleter;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;

/** Root dispatcher for independently managed shared modules. */
public final class SharedCommand implements CommandExecutor, TabCompleter {
    private final Map<String, SharedModuleCommand> modules;

    public SharedCommand(List<SharedModuleCommand> modules) {
        var indexed = new LinkedHashMap<String, SharedModuleCommand>();
        for (SharedModuleCommand module : modules) {
            SharedModuleCommand duplicate = indexed.put(
                module.moduleName().toLowerCase(Locale.ROOT),
                Objects.requireNonNull(module, "module")
            );
            if (duplicate != null) {
                throw new IllegalArgumentException(
                    "Duplicate shared module: " + module.moduleName()
                );
            }
        }
        this.modules = Map.copyOf(indexed);
    }

    @Override
    public boolean onCommand(
        CommandSender sender,
        Command command,
        String label,
        String[] arguments
    ) {
        if (arguments.length < 2) {
            sender.sendMessage(
                "Usage: /shared <health|food> <enable|disable|status|resume>"
            );
            return true;
        }
        SharedModuleCommand module = modules.get(
            arguments[0].toLowerCase(Locale.ROOT)
        );
        if (module == null) {
            sender.sendMessage("Unknown shared module: " + arguments[0]);
            return true;
        }
        return module.execute(
            sender,
            java.util.Arrays.copyOfRange(arguments, 1, arguments.length)
        );
    }

    @Override
    public List<String> onTabComplete(
        CommandSender sender,
        Command command,
        String alias,
        String[] arguments
    ) {
        if (arguments.length == 1) {
            return filter(modules.keySet().stream().sorted().toList(), arguments[0]);
        }
        SharedModuleCommand module = modules.get(
            arguments[0].toLowerCase(Locale.ROOT)
        );
        if (module == null) {
            return List.of();
        }
        return module.complete(
            sender,
            java.util.Arrays.copyOfRange(arguments, 1, arguments.length)
        );
    }

    public static List<String> filter(List<String> values, String prefix) {
        String normalized = prefix.toLowerCase(Locale.ROOT);
        return values.stream()
            .filter(value -> value.toLowerCase(Locale.ROOT).startsWith(normalized))
            .toList();
    }
}
