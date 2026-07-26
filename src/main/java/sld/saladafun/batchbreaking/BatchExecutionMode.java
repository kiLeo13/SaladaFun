package sld.saladafun.batchbreaking;

/**
 * Controls whether a batch is spread across ticks or completed in one blocking operation.
 */
public enum BatchExecutionMode {
    ASYNC,
    SYNC
}
