package sld.saladafun.persistence.sqlite;

import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;

class CoalescingPersistenceWriterTest {

    @Test
    void writesOnlyTheLatestSnapshotForAKeyOffTheCallingThread() {
        AtomicInteger writes = new AtomicInteger();
        AtomicInteger value = new AtomicInteger();
        AtomicReference<String> writerThread = new AtomicReference<>();
        String callingThread = Thread.currentThread().getName();

        try (var writer = new CoalescingPersistenceWriter(Duration.ofDays(1))) {
            writer.submitLatest("health", () -> {
                writes.incrementAndGet();
                value.set(1);
            });
            writer.submitLatest("health", () -> {
                writes.incrementAndGet();
                value.set(2);
                writerThread.set(Thread.currentThread().getName());
            });

            writer.flush();
        }

        assertEquals(1, writes.get());
        assertEquals(2, value.get());
        assertNotEquals(callingThread, writerThread.get());
        assertEquals("saladafun-sqlite-writer", writerThread.get());
    }

    @Test
    void flushPersistsIndependentModuleKeys() {
        AtomicInteger writes = new AtomicInteger();
        try (var writer = new CoalescingPersistenceWriter(Duration.ofDays(1))) {
            writer.submitLatest("health", writes::incrementAndGet);
            writer.submitLatest("food", writes::incrementAndGet);

            writer.flush();
        }

        assertEquals(2, writes.get());
    }

    @Test
    void surfacesBackgroundWriteFailureAtFlushAndStillCloses() {
        var writer = new CoalescingPersistenceWriter(Duration.ofDays(1));
        writer.submitLatest("effects", () -> {
            throw new IllegalStateException("database unavailable");
        });

        assertThrows(PersistenceException.class, writer::flush);
        assertThrows(PersistenceException.class, writer::close);
    }
}
