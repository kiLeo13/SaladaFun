package sld.saladafun.shared.model;

import java.util.Objects;
import java.util.regex.Pattern;

/** Human-facing session identifier such as {@code 20260802_01}. */
public record SessionLabel(String value) {
    private static final Pattern FORMAT = Pattern.compile("\\d{8}_\\d{2,}");

    public SessionLabel {
        Objects.requireNonNull(value, "value");
        if (!FORMAT.matcher(value).matches()) {
            throw new IllegalArgumentException("Invalid session label: " + value);
        }
    }
}
