package sld.saladafun.discordutils.discord;

import java.util.EnumSet;
import java.util.Objects;
import net.dv8tion.jda.api.components.container.Container;
import net.dv8tion.jda.api.components.textdisplay.TextDisplay;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.entities.channel.middleman.GuildMessageChannel;
import net.dv8tion.jda.api.utils.MarkdownSanitizer;
import net.dv8tion.jda.api.utils.MarkdownUtil;
import net.dv8tion.jda.api.utils.messages.MessageCreateBuilder;
import net.dv8tion.jda.api.utils.messages.MessageCreateData;
import org.slf4j.Logger;

/** Sends player join and leave notifications as Padinho bot messages. */
final class DiscordPlayerNotificationSender {
    static final int JOINED_ACCENT_COLOR = 0x57F287;
    static final int LEFT_ACCENT_COLOR = 0xED4245;
    static final int MAX_COMPONENT_TEXT_LENGTH = 4_000;

    private final GuildMessageChannel channel;
    private final Logger logger;

    /** Creates a sender for the configured Discord guild channel. */
    DiscordPlayerNotificationSender(GuildMessageChannel channel, Logger logger) {
        this.channel = Objects.requireNonNull(channel, "channel");
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    /** Queues one mention-safe Components V2 player-presence notification. */
    void send(String playerName, PlayerPresence presence) {
        channel.sendMessage(createMessage(playerName, presence)).queue(
            ignored -> { },
            failure -> logger.warn(
                "Could not deliver a Minecraft player presence notification through Discord ({})",
                failure.getClass().getSimpleName()
            )
        );
    }

    /** Creates the complete Components V2 message for one presence transition. */
    static MessageCreateData createMessage(String playerName, PlayerPresence presence) {
        Objects.requireNonNull(playerName, "playerName");
        Objects.requireNonNull(presence, "presence");

        String boldPlayerName = MarkdownUtil.bold(playerName);
        String text = switch (presence.type()) {
            case JOINED -> "%s entrou no servidor".formatted(boldPlayerName);
            case LEFT -> presence.disconnectReason()
                .map(reason -> "%s desconectou: %s".formatted(boldPlayerName, MarkdownSanitizer.escape(reason)))
                .orElseGet(() -> "%s saiu do servidor".formatted(boldPlayerName));
        };
        int accentColor = switch (presence.type()) {
            case JOINED -> JOINED_ACCENT_COLOR;
            case LEFT -> LEFT_ACCENT_COLOR;
        };
        Container container = Container.of(TextDisplay.of(limitToComponentText(text))).withAccentColor(accentColor);

        return new MessageCreateBuilder()
            .setComponents(container)
            .useComponentsV2()
            .setAllowedMentions(EnumSet.noneOf(Message.MentionType.class))
            .build();
    }

    /** Limits a text display to Discord's UTF-16 length without splitting a surrogate pair. */
    static String limitToComponentText(String text) {
        if (text.length() <= MAX_COMPONENT_TEXT_LENGTH) {
            return text;
        }

        int end = MAX_COMPONENT_TEXT_LENGTH - 1;
        if (Character.isHighSurrogate(text.charAt(end - 1))) {
            end--;
        }
        return text.substring(0, end) + "…";
    }
}
