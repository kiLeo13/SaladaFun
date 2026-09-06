package sld.saladafun.platform.purpur.discord;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import net.kyori.adventure.text.format.TextColor;
import org.bukkit.Server;
import org.bukkit.plugin.java.JavaPlugin;

import java.util.Objects;
import java.util.function.Consumer;

/**
 * Moves Discord callbacks onto the Minecraft server thread before broadcasting.
 */
public final class DiscordMinecraftBroadcaster implements Consumer<DiscordInboundMessage> {
    private static final String DISCORD_PREFIX = "[Discord]";
    private static final int DISCORD_BLURPLE_RGB = 0x5865F2;
    private static final TextColor DISCORD_PREFIX_COLOR =
        TextColor.color(DISCORD_BLURPLE_RGB);
    private static final String AUTHOR_SUFFIX = ":";
    private static final TextColor AUTHOR_COLOR = NamedTextColor.WHITE;
    private static final TextColor MESSAGE_CONTENT_COLOR = NamedTextColor.GRAY;
    private static final TextColor MEDIA_COUNTER_COLOR = NamedTextColor.LIGHT_PURPLE;
    private static final String IMAGE_LABEL = "image";
    private static final String STICKER_LABEL = "sticker";

    private final JavaPlugin plugin;
    private final Server server;

    public DiscordMinecraftBroadcaster(JavaPlugin plugin) {
        this.plugin = Objects.requireNonNull(plugin, "plugin");
        this.server = Objects.requireNonNull(plugin.getServer(), "plugin server");
    }

    @Override
    public void accept(DiscordInboundMessage message) {
        Objects.requireNonNull(message, "message");
        if (!plugin.isEnabled()) {
            return;
        }
        Component rendered = render(message);
        server.getScheduler().runTask(plugin, () -> server.broadcast(rendered));
    }

    static Component render(DiscordInboundMessage message) {
        Component rendered = Component.text(DISCORD_PREFIX, DISCORD_PREFIX_COLOR)
            .append(Component.space())
            .append(Component.text(message.authorName(), AUTHOR_COLOR))
            .append(Component.text(AUTHOR_SUFFIX, AUTHOR_COLOR));
        if (!message.content().isBlank()) {
            rendered = rendered.append(Component.space())
                .append(Component.text(message.content(), MESSAGE_CONTENT_COLOR));
        }
        if (message.imageCount() > 0 || message.stickerCount() > 0) {
            rendered = rendered.append(Component.newline());
            if (message.imageCount() > 0) {
                rendered = rendered.append(
                    mediaCounter(message.imageCount(), IMAGE_LABEL)
                );
            }
            if (message.imageCount() > 0 && message.stickerCount() > 0) {
                rendered = rendered.append(Component.space());
            }
            if (message.stickerCount() > 0) {
                rendered = rendered.append(
                    mediaCounter(message.stickerCount(), STICKER_LABEL)
                );
            }
        }
        return rendered;
    }

    private static Component mediaCounter(int count, String singular) {
        String label = count == 1 ? singular : singular + "s";
        return Component.text(
            "[+" + count + " " + label + "]",
            MEDIA_COUNTER_COLOR
        );
    }
}
