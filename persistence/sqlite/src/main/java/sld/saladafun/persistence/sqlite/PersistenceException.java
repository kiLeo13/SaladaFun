package sld.saladafun.persistence.sqlite;

/**
 * Unchecked boundary exception used to fail closed when durable state cannot be trusted.
 */
public final class PersistenceException extends RuntimeException {
    public PersistenceException(String message, Throwable cause) {
        super(message, cause);
    }

    public PersistenceException(String message) {
        super(message);
    }
}
