package sld.saladafun.batchbreaking;

import java.util.Locale;
import java.util.Objects;

/**
 * Parses the command/config representations {@code disabled}, {@code all}, or a positive integer.
 */
public final class BatchBreakingSettingParser {
    public BatchBreakingSetting parse(String value) {
        Objects.requireNonNull(value, "value");
        String normalized = value.strip().toLowerCase(Locale.ROOT);
        return switch (normalized) {
            case "disabled" -> new BatchBreakingSetting.Disabled();
            case "all" -> new BatchBreakingSetting.AllSentChunks();
            default -> parseRadius(normalized);
        };
    }

    public String format(BatchBreakingSetting setting) {
        Objects.requireNonNull(setting, "setting");
        return switch (setting) {
            case BatchBreakingSetting.Disabled ignored -> "disabled";
            case BatchBreakingSetting.AllSentChunks ignored -> "all";
            case BatchBreakingSetting.CubicRadius radius ->
                Integer.toString(radius.blocks());
        };
    }

    private BatchBreakingSetting parseRadius(String value) {
        final long radius;
        try {
            radius = Long.parseLong(value);
        } catch (NumberFormatException exception) {
            throw new IllegalArgumentException(
                "Expected disabled, all, or a positive integer", exception
            );
        }
        if (radius < 1 || radius > Integer.MAX_VALUE) {
            throw new IllegalArgumentException("Batch-breaking radius must be between 1 and "
                + Integer.MAX_VALUE);
        }
        return new BatchBreakingSetting.CubicRadius((int) radius);
    }
}
