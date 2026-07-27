package sld.saladafun.platform.purpur.discord;

import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;
import org.bukkit.Server;
import org.bukkit.plugin.java.JavaPlugin;

import java.util.Objects;
import java.util.function.Consumer;

/**
 * Moves Discord callbacks onto the Minecraft server thread before broadcasting.
 */
public final class DiscordMinecraftBroadcaster implements Consumer<DiscordInboundMessage> {
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
        Component rendered = Component.text("[Discord] ", NamedTextColor.DARK_AQUA)
            .append(Component.text("<" + message.authorName() + ">", NamedTextColor.WHITE));
        if (!message.content().isBlank()) {
            rendered = rendered.append(Component.space())
                .append(Component.text(message.content(), NamedTextColor.WHITE));
        }
        if (message.imageCount() > 0 || message.stickerCount() > 0) {
            rendered = rendered.append(Component.newline());
            if (message.imageCount() > 0) {
                rendered = rendered.append(mediaCounter(message.imageCount(), "image"));
            }
            if (message.imageCount() > 0 && message.stickerCount() > 0) {
                rendered = rendered.append(Component.space());
            }
            if (message.stickerCount() > 0) {
                rendered = rendered.append(mediaCounter(message.stickerCount(), "sticker"));
            }
        }
        return rendered;
    }

    private static Component mediaCounter(int count, String singular) {
        String label = count == 1 ? singular : singular + "s";
        return Component.text("[+" + count + " " + label + "]", NamedTextColor.LIGHT_PURPLE);
    }
}
