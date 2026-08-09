package sld.saladafun.platform.purpur.shared;

import org.bukkit.command.CommandSender;

import java.util.List;

/** Command delegate for one independently managed shared module. */
public interface SharedModuleCommand {
    String moduleName();

    boolean execute(CommandSender sender, String[] arguments);

    List<String> complete(CommandSender sender, String[] arguments);
}
