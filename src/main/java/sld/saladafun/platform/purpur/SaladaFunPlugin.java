package sld.saladafun.platform.purpur;

import org.bukkit.command.PluginCommand;
import org.bukkit.plugin.java.JavaPlugin;
import sld.saladafun.persistence.sqlite.SqliteSharedInventoryStore;
import sld.saladafun.platform.purpur.batch.BatchBreakingCommand;
import sld.saladafun.platform.purpur.batch.BatchBreakingHandler;
import sld.saladafun.platform.purpur.config.PluginSettings;
import sld.saladafun.platform.purpur.config.SaladaFunCommand;
import sld.saladafun.platform.purpur.discord.DiscordChatBridge;
import sld.saladafun.platform.purpur.discord.DiscordMinecraftBroadcaster;
import sld.saladafun.platform.purpur.discord.MinecraftChatListener;
import sld.saladafun.platform.purpur.shared.PlayerInventorySynchronizer;
import sld.saladafun.platform.purpur.shared.PurpurInventoryMapper;
import sld.saladafun.platform.purpur.shared.SharedInventoryCommand;
import sld.saladafun.platform.purpur.shared.SharedInventoryHandler;
import sld.saladafun.shared.inventory.SharedInventoryManager;

import java.io.IOException;
import java.nio.file.Files;
import java.time.Clock;
import java.time.ZoneId;
import java.util.Objects;

/**
 * Purpur composition root. Platform-independent services are constructed here and injected
 * into the server adapters; no static service locator is used.
 */
public final class SaladaFunPlugin extends JavaPlugin {
    private SharedInventoryManager sharedInventoryManager;
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

            var store = new SqliteSharedInventoryStore(
                getDataFolder().toPath().resolve("shared-inventory.db")
            );
            sharedInventoryManager = new SharedInventoryManager(
                store, Clock.systemUTC(), ZoneId.systemDefault()
            );
            sharedInventoryManager.load();

            var mapper = new PurpurInventoryMapper();
            var synchronizer = new PlayerInventorySynchronizer(
                getServer(), sharedInventoryManager, mapper
            );
            var sharedHandler = new SharedInventoryHandler(
                this, sharedInventoryManager, mapper, synchronizer, settings
            );
            batchBreakingHandler = new BatchBreakingHandler(this, settings);
            discordChatBridge = new DiscordChatBridge(
                new DiscordMinecraftBroadcaster(this),
                getLogger()
            );

            getServer().getPluginManager().registerEvents(sharedHandler, this);
            getServer().getPluginManager().registerEvents(batchBreakingHandler, this);
            getServer().getPluginManager().registerEvents(
                new MinecraftChatListener(discordChatBridge),
                this
            );
            discordChatBridge.reconfigure(settings.discordChatSettings());
            registerCommands(mapper, synchronizer, settings, discordChatBridge);

            // Handles plugin reloads, where players may already be online.
            getServer().getOnlinePlayers().forEach(sharedHandler::reconcilePlayer);
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
        if (sharedInventoryManager != null) {
            sharedInventoryManager.close();
        }
    }

    private void registerCommands(
        PurpurInventoryMapper mapper,
        PlayerInventorySynchronizer synchronizer,
        PluginSettings settings,
        DiscordChatBridge discordBridge
    ) {
        var shared = new SharedInventoryCommand(
            getServer(), sharedInventoryManager, mapper, synchronizer
        );
        PluginCommand sharedCommand = Objects.requireNonNull(
            getCommand("shared"), "shared command missing from plugin.yml"
        );
        sharedCommand.setExecutor(shared);
        sharedCommand.setTabCompleter(shared);

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
