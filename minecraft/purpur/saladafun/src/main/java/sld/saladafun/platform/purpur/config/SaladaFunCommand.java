package sld.saladafun.platform.purpur.config;

import org.bukkit.command.Command;
import org.bukkit.command.CommandExecutor;
import org.bukkit.command.CommandSender;
import org.bukkit.configuration.InvalidConfigurationException;
import org.bukkit.configuration.file.YamlConfiguration;
import org.bukkit.plugin.java.JavaPlugin;

import java.io.File;
import java.io.IOException;
import java.util.Objects;
import java.util.function.Consumer;

/**
 * Administrative entry point for validated runtime configuration reloads.
 */
public final class SaladaFunCommand implements CommandExecutor {
    private final JavaPlugin plugin;
    private final PluginSettings settings;
    private final Consumer<DiscordChatSettings> discordChatConfigurator;

    public SaladaFunCommand(JavaPlugin plugin, PluginSettings settings) {
        this(plugin, settings, ignored -> {
            // Compatibility constructor for contexts without a Discord bridge.
        });
    }

    public SaladaFunCommand(
        JavaPlugin plugin,
        PluginSettings settings,
        Consumer<DiscordChatSettings> discordChatConfigurator
    ) {
        this.plugin = Objects.requireNonNull(plugin, "plugin");
        this.settings = Objects.requireNonNull(settings, "settings");
        this.discordChatConfigurator = Objects.requireNonNull(
            discordChatConfigurator,
            "discordChatConfigurator"
        );
    }

    @Override
    public boolean onCommand(
        CommandSender sender,
        Command command,
        String label,
        String[] arguments
    ) {
        if (arguments.length != 1
            || !arguments[0].equalsIgnoreCase("reloadconfig")) {
            sender.sendMessage("Usage: /saladafun reloadconfig");
            return true;
        }

        try {
            File file = new File(plugin.getDataFolder(), "config.yml");
            YamlConfiguration candidate = new YamlConfiguration();
            candidate.load(file);
            new PluginSettings(candidate).validate();

            plugin.reloadConfig();
            settings.replace(plugin.getConfig());
            discordChatConfigurator.accept(settings.discordChatSettings());
            sender.sendMessage("SaladaFun configuration reloaded.");
        } catch (IOException | InvalidConfigurationException | RuntimeException exception) {
            plugin.getLogger().warning(
                "Could not reload SaladaFun configuration: " + exception.getMessage()
            );
            sender.sendMessage(
                "SaladaFun configuration was not reloaded: " + exception.getMessage()
            );
        }
        return true;
    }
}
