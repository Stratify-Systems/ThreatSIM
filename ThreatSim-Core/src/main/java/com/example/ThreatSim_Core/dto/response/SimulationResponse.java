package com.example.ThreatSim_Core.dto.response;

import com.example.ThreatSim_Core.enums.AttackType;
import com.example.ThreatSim_Core.enums.HttpMethodType;

import java.util.Map;
import java.util.UUID;

public record SimulationResponse(
        UUID id,
        UUID applicationId,
        String applicationName,
        AttackType attackType,
        HttpMethodType httpMethod,
        String endpoint,
        Map<String, String> headers,
        String requestBody,
        int expectedStatus,
        boolean enabled
) {
}
