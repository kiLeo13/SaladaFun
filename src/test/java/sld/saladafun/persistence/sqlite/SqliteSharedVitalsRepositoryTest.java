package sld.saladafun.persistence.sqlite;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import sld.saladafun.shared.food.FoodState;
import sld.saladafun.shared.food.SharedFoodManager;
import sld.saladafun.shared.health.HealthContribution;
import sld.saladafun.shared.health.HealthPhase;
import sld.saladafun.shared.health.HealthState;
import sld.saladafun.shared.health.SharedHealthManager;
import sld.saladafun.shared.model.RestoreStatus;
import sld.saladafun.shared.model.SessionLabel;

import java.nio.file.Path;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.time.ZoneId;
import java.time.ZoneOffset;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

class SqliteSharedVitalsRepositoryTest {
    private static final Clock CLOCK = Clock.fixed(
        Instant.parse("2026-08-02T18:00:00Z"),
        ZoneOffset.UTC
    );
    private static final ZoneId LABEL_ZONE = ZoneOffset.UTC;

    @TempDir
    Path temporaryDirectory;

    @Test
    void persistsHealthLifecycleAgainstRealSQLite() {
        Path databaseFile = temporaryDirectory.resolve("health.db");
        UUID playerId = UUID.randomUUID();

        try (SqliteDatabase database = new SqliteDatabase(databaseFile)) {
            var repository = new SqliteSharedHealthRepository(database.context(), CLOCK);
            var manager = new SharedHealthManager(repository, CLOCK, LABEL_ZONE);
            HealthState personal = new HealthState(
                12.0, 20.0, 2.0, 8.0, HealthPhase.ALIVE, 0
            );

            var session = manager.enableFresh(Map.of(playerId, personal));
            assertEquals("20260802_01", session.label().value());

            HealthState changed = manager.applyTick(List.of(
                new HealthContribution(
                    playerId, -3.0, 1.0, true, 16.0, 6.0
                )
            ), false);
            assertEquals(16.0, changed.maximumHealth());
            assertEquals(16.0, changed.health());
            assertEquals(1, changed.revision());

            manager.disable();
            var backup = manager.pendingRestore(playerId).orElseThrow();
            assertEquals(personal, backup.state());
            assertEquals(RestoreStatus.RESTORE_PENDING, backup.restoreStatus());
            manager.markRestored(backup);
            assertTrue(manager.pendingRestore(playerId).isEmpty());

            manager.resume(new SessionLabel("20260802_01"), Map.of(playerId, personal));
            assertEquals(changed, manager.current().orElseThrow());
        }

        try (SqliteDatabase reopened = new SqliteDatabase(databaseFile)) {
            var repository = new SqliteSharedHealthRepository(reopened.context(), CLOCK);
            assertEquals(1, repository.loadActive().orElseThrow().state().revision());
        }
    }

    @Test
    void persistsFoodLifecycleAndModuleScopedLabelsAgainstRealSQLite() {
        Path databaseFile = temporaryDirectory.resolve("food.db");
        UUID playerId = UUID.randomUUID();

        try (SqliteDatabase database = new SqliteDatabase(databaseFile)) {
            var foodRepository = new SqliteSharedFoodRepository(database.context(), CLOCK);
            var foodManager = new SharedFoodManager(foodRepository, CLOCK, LABEL_ZONE);
            FoodState personal = new FoodState(9, 2.0F, 1.0F, 0);

            var first = foodManager.enableFresh(Map.of(playerId, personal));
            assertEquals("20260802_01", first.label().value());
            foodManager.applyTick(List.of(
                new sld.saladafun.shared.food.FoodContribution(
                    playerId, -1, -1.0F, 0.5F
                )
            ));
            foodManager.disable();
            assertEquals(
                RestoreStatus.RESTORE_PENDING,
                foodManager.pendingRestore(playerId).orElseThrow().restoreStatus()
            );

            var second = foodManager.enableFresh(Map.of());
            assertEquals("20260802_02", second.label().value());
            foodManager.disable();

            var healthRepository = new SqliteSharedHealthRepository(
                database.context(), CLOCK
            );
            var healthManager = new SharedHealthManager(
                healthRepository, CLOCK, LABEL_ZONE
            );
            assertEquals(
                "20260802_01",
                healthManager.enableFresh(Map.of()).label().value()
            );
        }
    }

    @Test
    void managersLoadOnlyTheirOwnActiveCanonicalState() {
        Path databaseFile = temporaryDirectory.resolve("both.db");

        try (SqliteDatabase database = new SqliteDatabase(databaseFile)) {
            var healthRepository = new SqliteSharedHealthRepository(
                database.context(), CLOCK
            );
            var foodRepository = new SqliteSharedFoodRepository(database.context(), CLOCK);
            new SharedHealthManager(healthRepository, CLOCK, LABEL_ZONE).enableFresh(Map.of());
            new SharedFoodManager(foodRepository, CLOCK, LABEL_ZONE).enableFresh(Map.of());

            var loadedHealth = new SharedHealthManager(
                healthRepository, CLOCK, LABEL_ZONE
            );
            var loadedFood = new SharedFoodManager(foodRepository, CLOCK, LABEL_ZONE);
            loadedHealth.load();
            loadedFood.load();

            assertTrue(loadedHealth.isEnabled());
            assertTrue(loadedFood.isEnabled());
            assertFalse(loadedHealth.archivedSessions().stream().findAny().isPresent());
            assertEquals(FoodState.fresh(), loadedFood.current().orElseThrow());
        }
    }

    @Test
    void coalescesCanonicalRevisionsAgainstRealSQLite() {
        Path databaseFile = temporaryDirectory.resolve("async.db");
        UUID playerId = UUID.randomUUID();

        try (SqliteDatabase database = new SqliteDatabase(databaseFile);
             CoalescingPersistenceWriter writer =
                 new CoalescingPersistenceWriter(Duration.ofDays(1))) {
            var synchronous = new SqliteSharedHealthRepository(
                database.context(), CLOCK
            );
            var asynchronous = new AsyncSharedHealthRepository(
                synchronous, writer
            );
            var manager = new SharedHealthManager(
                asynchronous, CLOCK, LABEL_ZONE
            );
            manager.enableFresh(Map.of());

            manager.applyTick(List.of(new HealthContribution(
                playerId, -1.0, 0.0, false, 20.0, 0.0
            )), false);
            HealthState latest = manager.applyTick(List.of(new HealthContribution(
                playerId, -2.0, 0.0, false, 20.0, 0.0
            )), false);
            writer.flush();

            assertEquals(latest, synchronous.loadActive().orElseThrow().state());
        }
    }
}
