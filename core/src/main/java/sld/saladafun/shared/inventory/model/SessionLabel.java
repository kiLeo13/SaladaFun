package sld.saladafun.shared.inventory.model;

import java.util.Objects;
import java.util.regex.Pattern;

/**
 * Human-readable session label in {@code yyyyMMdd_nn} form.
 */
public record SessionLabel(String value) {
    private static final Pattern FORMAT = Pattern.compile("\\d{8}_\\d+");

    public SessionLabel {
        Objects.requireNonNull(value, "value");
        if (!FORMAT.matcher(value).matches()) {
            throw new IllegalArgumentException("Invalid session label: " + value);
        }
    }
}
