package sld.saladafun.platform.purpur.config;

import org.bukkit.command.Command;
import org.bukkit.command.CommandSender;
import org.bukkit.configuration.file.YamlConfiguration;
import org.bukkit.plugin.java.JavaPlugin;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.logging.Logger;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.startsWith;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class SaladaFunCommandTest {
    @TempDir
    Path temporaryDirectory;

    @Test
    void reloadsValidatedConfigurationIntoTheSharedSettingsView() throws IOException {
        Files.writeString(
            temporaryDirectory.resolve("config.yml"),
            "batch-breaking:%n  sync-batching: SYNC%n".formatted()
        );
        JavaPlugin plugin = plugin();
        YamlConfiguration reloaded = new YamlConfiguration();
        reloaded.set("batch-breaking.sync-batching", "SYNC");
        when(plugin.getConfig()).thenReturn(reloaded);
        PluginSettings settings = new PluginSettings(new YamlConfiguration());
        CommandSender sender = mock(CommandSender.class);

        new SaladaFunCommand(plugin, settings).onCommand(
            sender,
            mock(Command.class),
            "saladafun",
            new String[]{"reloadconfig"}
        );

        verify(plugin).reloadConfig();
        verify(sender).sendMessage("SaladaFun configuration reloaded.");
        assertEquals(
            sld.saladafun.batchbreaking.BatchExecutionMode.SYNC,
            settings.batchExecutionMode()
        );
    }

    @Test
    void rejectsInvalidConfigurationWithoutReplacingLiveSettings() throws IOException {
        Files.writeString(
            temporaryDirectory.resolve("config.yml"),
            "batch-breaking:%n  sync-batching: EVENTUALLY%n".formatted()
        );
        JavaPlugin plugin = plugin();
        PluginSettings settings = new PluginSettings(new YamlConfiguration());
        CommandSender sender = mock(CommandSender.class);

        new SaladaFunCommand(plugin, settings).onCommand(
            sender,
            mock(Command.class),
            "saladafun",
            new String[]{"reloadconfig"}
        );

        verify(plugin, never()).reloadConfig();
        verify(sender).sendMessage(startsWith("SaladaFun configuration was not reloaded:"));
        assertEquals(
            sld.saladafun.batchbreaking.BatchExecutionMode.ASYNC,
            settings.batchExecutionMode()
        );
    }

    private JavaPlugin plugin() {
        JavaPlugin plugin = mock(JavaPlugin.class);
        when(plugin.getDataFolder()).thenReturn(temporaryDirectory.toFile());
        when(plugin.getLogger()).thenReturn(Logger.getAnonymousLogger());
        return plugin;
    }
}
