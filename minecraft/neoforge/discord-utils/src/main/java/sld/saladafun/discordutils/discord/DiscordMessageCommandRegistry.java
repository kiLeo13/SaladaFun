package sld.saladafun.discordutils.discord;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.function.Consumer;
import java.util.regex.Pattern;

/** Registers literal Discord message commands and dispatches exact first-token matches. */
final class DiscordMessageCommandRegistry {
    private static final Pattern ARGUMENT_SEPARATOR = Pattern.compile("(?U)\\s+");

    private final List<Declaration> declarations = new ArrayList<>();

    private Map<String, Consumer<DiscordMessageCommandRequest>> handlers;

    /** Registers one complete literal trigger, including its prefix. */
    void register(String trigger, Consumer<DiscordMessageCommandRequest> handler) {
        if (handlers != null) {
            throw new IllegalStateException("Discord message command registry is frozen");
        }
        declarations.add(new Declaration(trigger, handler));
    }

    /** Validates declarations and freezes the registry for concurrent reads. */
    void freeze() {
        if (handlers != null) {
            return;
        }

        Map<String, Consumer<DiscordMessageCommandRequest>> dispatch = new HashMap<>();
        for (Declaration declaration : declarations) {
            String trigger = Objects.requireNonNull(declaration.trigger(), "trigger");
            Consumer<DiscordMessageCommandRequest> handler = Objects.requireNonNull(
                declaration.handler(),
                "handler"
            );
            if (trigger.isBlank() || !trigger.equals(trigger.strip()) || containsWhitespace(trigger)) {
                throw new IllegalArgumentException("Invalid Discord message command trigger: " + trigger);
            }
            if (dispatch.putIfAbsent(normalize(trigger), handler) != null) {
                throw new IllegalArgumentException("Duplicate Discord message command trigger: " + trigger);
            }
        }
        handlers = Map.copyOf(dispatch);
        declarations.clear();
    }

    /** Dispatches a registered first token and reports whether the message was consumed. */
    boolean dispatch(String content) {
        Objects.requireNonNull(content, "content");
        if (handlers == null) {
            throw new IllegalStateException("Discord message command registry is not frozen");
        }

        String stripped = content.strip();
        if (stripped.isEmpty()) {
            return false;
        }

        int argumentStart = firstWhitespace(stripped);
        String trigger = argumentStart < 0 ? stripped : stripped.substring(0, argumentStart);
        Consumer<DiscordMessageCommandRequest> handler = handlers.get(normalize(trigger));
        if (handler == null) {
            return false;
        }

        String rawArguments = argumentStart < 0 ? "" : stripped.substring(argumentStart).strip();
        List<String> arguments = rawArguments.isEmpty()
            ? List.of()
            : List.of(ARGUMENT_SEPARATOR.split(rawArguments));
        handler.accept(new DiscordMessageCommandRequest(trigger, stripped, rawArguments, arguments));
        return true;
    }

    /** Returns whether a trigger contains any Unicode whitespace. */
    private static boolean containsWhitespace(String trigger) {
        return trigger.codePoints().anyMatch(
            codePoint -> Character.isWhitespace(codePoint) || Character.isSpaceChar(codePoint)
        );
    }

    /** Finds the first Unicode whitespace boundary or reports its absence. */
    private static int firstWhitespace(String content) {
        for (int index = 0; index < content.length();) {
            int codePoint = content.codePointAt(index);
            if (Character.isWhitespace(codePoint) || Character.isSpaceChar(codePoint)) {
                return index;
            }
            index += Character.charCount(codePoint);
        }
        return -1;
    }

    /** Normalizes triggers for case-insensitive lookup. */
    private static String normalize(String trigger) {
        return trigger.toLowerCase(Locale.ROOT);
    }

    /** Stores one mutable-startup declaration before the registry is frozen. */
    private record Declaration(String trigger, Consumer<DiscordMessageCommandRequest> handler) {
    }
}
