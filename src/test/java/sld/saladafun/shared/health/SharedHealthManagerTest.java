package sld.saladafun.shared.health;

import org.junit.jupiter.api.Test;
import sld.saladafun.persistence.sqlite.PersistenceException;
import sld.saladafun.shared.model.InitialStateMode;
import sld.saladafun.shared.model.SessionId;
import sld.saladafun.shared.model.SessionLabel;
import sld.saladafun.shared.model.SessionStatus;

import java.time.Clock;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneOffset;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class SharedHealthManagerTest {

    @Test
    void rollsBackTheInMemoryCanonicalStateWhenPersistenceFails() {
        SharedHealthRepository repository = mock(SharedHealthRepository.class);
        Clock clock = Clock.fixed(
            Instant.parse("2026-08-02T18:00:00Z"), ZoneOffset.UTC
        );
        HealthState initial = HealthState.full(20.0, 0.0);
        HealthSession session = new HealthSession(
            SessionId.create(),
            new SessionLabel("20260802_01"),
            LocalDate.of(2026, 8, 2),
            1,
            SessionStatus.ACTIVE,
            InitialStateMode.FRESH,
            null,
            initial,
            Instant.now(clock),
            null
        );
        when(repository.create(any(), any(), any(), any(), any()))
            .thenReturn(session);
        doThrow(new PersistenceException("write failed"))
            .when(repository).saveCanonical(any(), any());
        SharedHealthManager manager = new SharedHealthManager(
            repository, clock, ZoneOffset.UTC
        );
        manager.enableFresh(Map.of());

        assertThrows(
            PersistenceException.class,
            () -> manager.applyTick(List.of(new HealthContribution(
                UUID.randomUUID(), -2.0, 0.0, false, 20.0, 0.0
            )), false)
        );

        assertEquals(initial, manager.current().orElseThrow());
    }
}
