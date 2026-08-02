package sld.saladafun.platform.purpur;

import org.bukkit.command.PluginCommand;
import org.bukkit.plugin.java.JavaPlugin;
import sld.saladafun.platform.purpur.batch.BatchBreakingCommand;
import sld.saladafun.platform.purpur.batch.BatchBreakingHandler;
import sld.saladafun.platform.purpur.config.PluginSettings;
import sld.saladafun.platform.purpur.config.SaladaFunCommand;
import sld.saladafun.platform.purpur.discord.DiscordChatBridge;
import sld.saladafun.platform.purpur.discord.DiscordMinecraftBroadcaster;
import sld.saladafun.platform.purpur.discord.MinecraftChatListener;

import java.io.IOException;
import java.nio.file.Files;
import java.util.Objects;

/**
 * Purpur composition root. Platform-independent services are constructed here and injected
 * into the server adapters; no static service locator is used.
 */
public final class SaladaFunPlugin extends JavaPlugin {
    private BatchBreakingHandler batchBreakingHandler;
    private DiscordChatBridge discordChatBridge;

    @Override
    public void onEnable() {
        saveDefaultConfig();
        try {
            Files.createDirectories(getDataFolder().toPath());
            PluginSettings settings = new PluginSettings(getConfig());
            // Validate configuration before registering any event listeners.
            settings.validate();

            batchBreakingHandler = new BatchBreakingHandler(this, settings);
            discordChatBridge = new DiscordChatBridge(
                new DiscordMinecraftBroadcaster(this),
                getLogger()
            );

            getServer().getPluginManager().registerEvents(batchBreakingHandler, this);
            getServer().getPluginManager().registerEvents(
                new MinecraftChatListener(discordChatBridge),
                this
            );
            discordChatBridge.reconfigure(settings.discordChatSettings());
            registerCommands(settings, discordChatBridge);
        } catch (IOException | RuntimeException exception) {
            getLogger().severe("SaladaFun could not start safely: " + exception.getMessage());
            getServer().getPluginManager().disablePlugin(this);
        }
    }

    @Override
    public void onDisable() {
        if (discordChatBridge != null) {
            discordChatBridge.close();
        }
        if (batchBreakingHandler != null) {
            batchBreakingHandler.close();
        }
    }

    private void registerCommands(
        PluginSettings settings,
        DiscordChatBridge discordBridge
    ) {
        var batch = new BatchBreakingCommand(this, settings);
        PluginCommand batchCommand = Objects.requireNonNull(
            getCommand("batchbreaking"), "batchbreaking command missing from plugin.yml"
        );
        batchCommand.setExecutor(batch);
        batchCommand.setTabCompleter(batch);

        PluginCommand saladaFunCommand = Objects.requireNonNull(
            getCommand("saladafun"), "saladafun command missing from plugin.yml"
        );
        saladaFunCommand.setExecutor(
            new SaladaFunCommand(this, settings, discordBridge::reconfigure)
        );
    }
}
