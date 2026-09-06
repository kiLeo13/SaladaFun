package sld.saladafun.discordutils;

import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.fml.ModContainer;
import net.neoforged.fml.common.Mod;
import net.neoforged.fml.event.config.ModConfigEvent;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.event.server.ServerStartedEvent;
import net.neoforged.neoforge.event.server.ServerStoppingEvent;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import sld.saladafun.discordutils.config.DiscordChatSettings;
import sld.saladafun.discordutils.config.DiscordConfig;
import sld.saladafun.discordutils.discord.DiscordChatBridge;
import sld.saladafun.discordutils.platform.neoforge.DiscordMinecraftBroadcaster;
import sld.saladafun.discordutils.platform.neoforge.MinecraftChatListener;

/** Boots the dedicated-server Discord chat bridge and applies configuration reloads. */
@Mod(value = DiscordUtilsMod.MOD_ID, dist = Dist.DEDICATED_SERVER)
public final class DiscordUtilsMod {
    /** Stable NeoForge mod identifier. */
    public static final String MOD_ID = "discord_utils";

    private static final Logger LOGGER = LoggerFactory.getLogger(DiscordUtilsMod.class);

    private DiscordChatBridge chatBridge;

    /** Registers configuration and dedicated-server lifecycle listeners. */
    public DiscordUtilsMod(IEventBus modEventBus, ModContainer container) {
        container.registerConfig(DiscordConfig.TYPE, DiscordConfig.SPEC, "discord-utils.toml");
        modEventBus.addListener(this::onConfigReloaded);
        NeoForge.EVENT_BUS.addListener(this::onServerStarted);
        NeoForge.EVENT_BUS.addListener(this::onServerStopping);
    }

    private void onConfigReloaded(ModConfigEvent event) {
        if (!event.getConfig().getSpec().equals(DiscordConfig.SPEC) || chatBridge == null) {
            return;
        }

        applyConfiguration();
    }

    private void onServerStarted(ServerStartedEvent event) {
        chatBridge = new DiscordChatBridge(new DiscordMinecraftBroadcaster(event.getServer()), LOGGER);
        MinecraftChatListener chatListener = new MinecraftChatListener(chatBridge);
        NeoForge.EVENT_BUS.addListener(chatListener::onServerChat);
        applyConfiguration();
    }

    private void onServerStopping(ServerStoppingEvent event) {
        if (chatBridge != null) {
            chatBridge.close();
            chatBridge = null;
        }
    }

    private void applyConfiguration() {
        try {
            chatBridge.reconfigure(DiscordChatSettings.fromConfig());
        } catch (IllegalArgumentException exception) {
            LOGGER.warn(
                "Discord configuration was rejected; the previous bridge remains active: {}",
                exception.getMessage()
            );
        }
    }
}
