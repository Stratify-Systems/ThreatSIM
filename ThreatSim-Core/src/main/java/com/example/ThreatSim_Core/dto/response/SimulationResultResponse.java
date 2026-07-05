package com.example.ThreatSim_Core.dto.response;

import com.example.ThreatSim_Core.enums.SimulationStatus;

import java.time.LocalDateTime;
import java.util.UUID;

public record SimulationResultResponse(
        UUID id,
        UUID simulationId,
        SimulationStatus status,
        int expectedStatus,
        int actualStatus,
        String responseBody,
        long duration,
        LocalDateTime executedAt
) {
}
