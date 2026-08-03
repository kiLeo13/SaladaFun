package sld.saladafun.platform.purpur;

import org.bukkit.command.PluginCommand;
import org.bukkit.plugin.java.JavaPlugin;
import sld.saladafun.persistence.sqlite.SqliteDatabase;
import sld.saladafun.persistence.sqlite.SqliteSharedFoodRepository;
import sld.saladafun.persistence.sqlite.SqliteSharedHealthRepository;
import sld.saladafun.platform.purpur.batch.BatchBreakingCommand;
import sld.saladafun.platform.purpur.batch.BatchBreakingHandler;
import sld.saladafun.platform.purpur.config.PluginSettings;
import sld.saladafun.platform.purpur.config.SaladaFunCommand;
import sld.saladafun.platform.purpur.discord.DiscordChatBridge;
import sld.saladafun.platform.purpur.discord.DiscordMinecraftBroadcaster;
import sld.saladafun.platform.purpur.discord.MinecraftChatListener;
import sld.saladafun.platform.purpur.shared.SharedCommand;
import sld.saladafun.platform.purpur.shared.food.PlayerFoodSynchronizer;
import sld.saladafun.platform.purpur.shared.food.PurpurFoodMapper;
import sld.saladafun.platform.purpur.shared.food.SharedFoodCommand;
import sld.saladafun.platform.purpur.shared.food.SharedFoodHandler;
import sld.saladafun.platform.purpur.shared.health.PlayerHealthSynchronizer;
import sld.saladafun.platform.purpur.shared.health.PurpurHealthMapper;
import sld.saladafun.platform.purpur.shared.health.SharedDeathCoordinator;
import sld.saladafun.platform.purpur.shared.health.SharedHealthCommand;
import sld.saladafun.platform.purpur.shared.health.SharedHealthHandler;
import sld.saladafun.shared.food.SharedFoodManager;
import sld.saladafun.shared.health.SharedHealthManager;

import java.io.IOException;
import java.nio.file.Files;
import java.time.Clock;
import java.time.ZoneId;
import java.util.List;
import java.util.Objects;

/**
 * Purpur composition root. Platform-independent services are constructed here and injected
 * into the server adapters; no static service locator is used.
 */
public final class SaladaFunPlugin extends JavaPlugin {
    private BatchBreakingHandler batchBreakingHandler;
    private DiscordChatBridge discordChatBridge;
    private SqliteDatabase sharedStateDatabase;

    @Override
    public void onEnable() {
        saveDefaultConfig();
        try {
            Files.createDirectories(getDataFolder().toPath());
            PluginSettings settings = new PluginSettings(getConfig());
            // Validate configuration before registering any event listeners.
            settings.validate();
            var sharedVitalsSettings = settings.sharedVitalsSettings();

            Clock clock = Clock.systemUTC();
            sharedStateDatabase = new SqliteDatabase(
                getDataFolder().toPath().resolve("shared-state.db")
            );
            var healthManager = new SharedHealthManager(
                new SqliteSharedHealthRepository(sharedStateDatabase.context(), clock),
                clock,
                ZoneId.systemDefault()
            );
            var foodManager = new SharedFoodManager(
                new SqliteSharedFoodRepository(sharedStateDatabase.context(), clock),
                clock,
                ZoneId.systemDefault()
            );
            healthManager.load();
            foodManager.load();

            var healthMapper = new PurpurHealthMapper();
            var healthSynchronizer = new PlayerHealthSynchronizer(
                getServer(), healthMapper
            );
            var healthHandler = new SharedHealthHandler(
                healthManager,
                healthMapper,
                healthSynchronizer,
                new SharedDeathCoordinator(getServer()),
                sharedVitalsSettings.safetyAuditIntervalTicks()
            );
            var foodMapper = new PurpurFoodMapper();
            var foodSynchronizer = new PlayerFoodSynchronizer(getServer(), foodMapper);
            var foodHandler = new SharedFoodHandler(
                foodManager,
                foodMapper,
                foodSynchronizer,
                sharedVitalsSettings.safetyAuditIntervalTicks()
            );

            batchBreakingHandler = new BatchBreakingHandler(this, settings);
            discordChatBridge = new DiscordChatBridge(
                new DiscordMinecraftBroadcaster(this),
                getLogger()
            );

            getServer().getPluginManager().registerEvents(healthHandler, this);
            getServer().getPluginManager().registerEvents(foodHandler, this);
            getServer().getPluginManager().registerEvents(batchBreakingHandler, this);
            getServer().getPluginManager().registerEvents(
                new MinecraftChatListener(discordChatBridge),
                this
            );
            discordChatBridge.reconfigure(settings.discordChatSettings());
            registerCommands(
                settings,
                discordChatBridge,
                new SharedHealthCommand(
                    getServer(),
                    healthManager,
                    healthMapper,
                    healthSynchronizer,
                    healthHandler
                ),
                new SharedFoodCommand(
                    getServer(),
                    foodManager,
                    foodMapper,
                    foodSynchronizer,
                    foodHandler
                )
            );
            getServer().getOnlinePlayers().forEach(player -> {
                healthHandler.reconcilePlayer(player);
                foodHandler.reconcilePlayer(player);
            });
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
        if (sharedStateDatabase != null) {
            sharedStateDatabase.close();
        }
    }

    private void registerCommands(
        PluginSettings settings,
        DiscordChatBridge discordBridge,
        SharedHealthCommand healthCommand,
        SharedFoodCommand foodCommand
    ) {
        var shared = new SharedCommand(List.of(healthCommand, foodCommand));
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
