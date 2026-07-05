package com.example.ThreatSim_Core.dto.response;

import java.time.LocalDateTime;
import java.util.UUID;

public record ApplicationResponse(
        UUID id,
        String name,
        String baseUrl,
        LocalDateTime createdAt
) {
}
