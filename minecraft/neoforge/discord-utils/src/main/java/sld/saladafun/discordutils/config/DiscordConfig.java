package sld.saladafun.discordutils.config;

import net.neoforged.fml.config.ModConfig;
import net.neoforged.neoforge.common.ModConfigSpec;

/** Defines the administrator-controlled Discord bridge configuration. */
public final class DiscordConfig {
    /** Configuration type keeps credentials local to the server instance. */
    public static final ModConfig.Type TYPE = ModConfig.Type.COMMON;
    /** NeoForge specification used to generate documented TOML defaults. */
    public static final ModConfigSpec SPEC;
    /** Whether the bridge should establish Discord connectivity. */
    public static final ModConfigSpec.BooleanValue ENABLED;
    /** Discord bot token; never emit this value to logs. */
    public static final ModConfigSpec.ConfigValue<String> TOKEN;
    /** Discord incoming webhook URL; never emit this value to logs. */
    public static final ModConfigSpec.ConfigValue<String> WEBHOOK_URL;
    /** Discord text-channel snowflake used for inbound traffic. */
    public static final ModConfigSpec.ConfigValue<String> CHANNEL_ID;

    static {
        ModConfigSpec.Builder builder = new ModConfigSpec.Builder();
        builder.push("discord-chat");
        ENABLED = builder.comment("Bridges accepted Minecraft chat and one Discord text channel.")
            .define("enabled", false);
        TOKEN = builder.comment("Discord bot token. Keep this secret and never commit it.")
            .define("token", "");
        WEBHOOK_URL = builder.comment("Incoming webhook for Minecraft-to-Discord traffic. Keep this secret.")
            .define("webhook-url", "");
        CHANNEL_ID = builder.comment("Discord text-channel snowflake for Discord-to-Minecraft traffic.")
            .define("channel-id", "");
        builder.pop();
        SPEC = builder.build();
    }

    private DiscordConfig() {
    }
}
