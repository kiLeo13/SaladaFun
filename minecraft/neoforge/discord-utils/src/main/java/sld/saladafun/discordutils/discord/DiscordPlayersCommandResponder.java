package sld.saladafun.discordutils.discord;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.EnumSet;
import java.util.List;
import java.util.Objects;
import net.dv8tion.jda.api.components.container.Container;
import net.dv8tion.jda.api.components.textdisplay.TextDisplay;
import net.dv8tion.jda.api.entities.Message;
import net.dv8tion.jda.api.entities.channel.middleman.GuildMessageChannel;
import net.dv8tion.jda.api.utils.MarkdownSanitizer;
import net.dv8tion.jda.api.utils.messages.MessageCreateBuilder;
import net.dv8tion.jda.api.utils.messages.MessageCreateData;
import org.slf4j.Logger;

/** Renders and sends the registered Discord online-players command response. */
final class DiscordPlayersCommandResponder {
    static final int ACCENT_COLOR = 0x3C8527;
    static final String HEADING = "## <:mc_grass:1547842476193357885> Players Online";
    static final String NO_PLAYERS = "Nenhum jogador online";
    private static final int MAX_COMPONENT_TEXT_LENGTH = 4_000;

    private final GuildMessageChannel channel;
    private final Logger logger;

    /** Creates a responder for the configured Minecraft Discord channel. */
    DiscordPlayersCommandResponder(GuildMessageChannel channel, Logger logger) {
        this.channel = Objects.requireNonNull(channel, "channel");
        this.logger = Objects.requireNonNull(logger, "logger");
    }

    /** Queues the complete online-player list through Padinho's bot identity. */
    void send(List<String> playerNames) {
        for (MessageCreateData message : createMessages(playerNames)) {
            channel.sendMessage(message).queue(
                ignored -> { },
                failure -> logger.warn(
                    "Could not deliver the Discord online-players response ({})",
                    failure.getClass().getSimpleName()
                )
            );
        }
    }

    /** Creates minimal one-container messages while preserving every player name. */
    static List<MessageCreateData> createMessages(List<String> playerNames) {
        Objects.requireNonNull(playerNames, "playerNames");
        List<String> sortedNames = playerNames.stream()
            .map(playerName -> MarkdownSanitizer.escape(Objects.requireNonNull(playerName, "playerName")))
            .sorted(String.CASE_INSENSITIVE_ORDER.thenComparing(Comparator.naturalOrder()))
            .toList();

        if (sortedNames.isEmpty()) {
            return List.of(createMessage(HEADING + "\n- " + NO_PLAYERS));
        }

        List<MessageCreateData> messages = new ArrayList<>();
        StringBuilder content = new StringBuilder(HEADING);
        for (String playerName : sortedNames) {
            String listItem = "\n- " + playerName;
            if (HEADING.length() + listItem.length() > MAX_COMPONENT_TEXT_LENGTH) {
                throw new IllegalArgumentException("Minecraft player name exceeds Discord's component limit");
            }
            if (content.length() + listItem.length() > MAX_COMPONENT_TEXT_LENGTH) {
                messages.add(createMessage(content.toString()));
                content = new StringBuilder(HEADING);
            }
            content.append(listItem);
        }
        messages.add(createMessage(content.toString()));
        return List.copyOf(messages);
    }

    /** Wraps one text page in the requested Components V2 container. */
    private static MessageCreateData createMessage(String content) {
        Container container = Container.of(TextDisplay.of(content)).withAccentColor(ACCENT_COLOR);
        return new MessageCreateBuilder()
            .setComponents(container)
            .useComponentsV2()
            .setAllowedMentions(EnumSet.noneOf(Message.MentionType.class))
            .build();
    }
}
