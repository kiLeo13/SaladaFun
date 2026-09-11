package sld.saladafun.discordutils.discord;

import java.util.List;
import java.util.Objects;

/** Contains normalized, JDA-free input for one registered Discord message command. */
record DiscordMessageCommandRequest(
    String trigger,
    String content,
    String rawArguments,
    List<String> arguments
) {
    /** Validates strings and defensively copies parsed arguments. */
    DiscordMessageCommandRequest {
        Objects.requireNonNull(trigger, "trigger");
        Objects.requireNonNull(content, "content");
        Objects.requireNonNull(rawArguments, "rawArguments");
        arguments = List.copyOf(Objects.requireNonNull(arguments, "arguments"));
    }
}
