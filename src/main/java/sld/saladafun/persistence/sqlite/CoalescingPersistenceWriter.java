package sld.saladafun.persistence.sqlite;

import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

/** Serial background writer that retains only the latest task for each state key. */
public final class CoalescingPersistenceWriter implements AutoCloseable {
    private final ScheduledExecutorService executor;
    private final ScheduledFuture<?> scheduledFlush;
    private final Map<String, Runnable> pending = new LinkedHashMap<>();
    private final AtomicReference<RuntimeException> failure = new AtomicReference<>();
    private boolean closed;

    public CoalescingPersistenceWriter(Duration flushInterval) {
        Objects.requireNonNull(flushInterval, "flushInterval");
        if (flushInterval.isZero() || flushInterval.isNegative()) {
            throw new IllegalArgumentException("flushInterval must be positive");
        }
        executor = java.util.concurrent.Executors.newSingleThreadScheduledExecutor(
            Thread.ofPlatform()
                .name("saladafun-sqlite-writer")
                .daemon(true)
                .factory()
        );
        long delayMillis = Math.max(1L, flushInterval.toMillis());
        scheduledFlush = executor.scheduleWithFixedDelay(
            this::drainSafely,
            delayMillis,
            delayMillis,
            TimeUnit.MILLISECONDS
        );
    }

    public synchronized void submitLatest(String key, Runnable write) {
        Objects.requireNonNull(key, "key");
        Objects.requireNonNull(write, "write");
        requireOpen();
        throwIfFailed();
        pending.put(key, write);
    }

    /** Waits until every write submitted before this call has completed. */
    public void flush() {
        throwIfFailed();
        CompletableFuture<Void> barrier = new CompletableFuture<>();
        synchronized (this) {
            requireOpen();
            executor.execute(() -> {
                drainSafely();
                RuntimeException asynchronousFailure = failure.get();
                if (asynchronousFailure == null) {
                    barrier.complete(null);
                } else {
                    barrier.completeExceptionally(asynchronousFailure);
                }
            });
        }
        await(barrier);
    }

    public void throwIfFailed() {
        RuntimeException asynchronousFailure = failure.get();
        if (asynchronousFailure != null) {
            throw asynchronousFailure;
        }
    }

    @Override
    public void close() {
        synchronized (this) {
            if (closed) {
                return;
            }
        }
        RuntimeException closeFailure = null;
        try {
            flush();
        } catch (RuntimeException exception) {
            closeFailure = exception;
        }
        synchronized (this) {
            closed = true;
        }
        scheduledFlush.cancel(false);
        executor.shutdown();
        try {
            if (!executor.awaitTermination(5, TimeUnit.SECONDS)) {
                executor.shutdownNow();
            }
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            executor.shutdownNow();
            if (closeFailure == null) {
                closeFailure = new PersistenceException(
                    "Interrupted while closing the persistence writer",
                    exception
                );
            }
        }
        if (closeFailure != null) {
            throw closeFailure;
        }
    }

    private void drainSafely() {
        Map<String, Runnable> writes;
        synchronized (this) {
            if (pending.isEmpty() || failure.get() != null) {
                return;
            }
            writes = Map.copyOf(pending);
            pending.clear();
        }
        try {
            for (Runnable write : writes.values()) {
                write.run();
            }
        } catch (RuntimeException exception) {
            failure.compareAndSet(
                null,
                new PersistenceException("Asynchronous SQLite write failed", exception)
            );
        }
    }

    private synchronized void requireOpen() {
        if (closed) {
            throw new IllegalStateException("Persistence writer is closed");
        }
    }

    private void await(CompletableFuture<Void> barrier) {
        try {
            barrier.get();
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            throw new PersistenceException(
                "Interrupted while flushing asynchronous SQLite writes",
                exception
            );
        } catch (ExecutionException exception) {
            Throwable cause = exception.getCause();
            if (cause instanceof RuntimeException runtimeException) {
                throw runtimeException;
            }
            throw new PersistenceException(
                "Could not flush asynchronous SQLite writes",
                cause
            );
        }
    }
}
