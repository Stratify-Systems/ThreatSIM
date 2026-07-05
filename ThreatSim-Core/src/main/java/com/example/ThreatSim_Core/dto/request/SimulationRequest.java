package com.example.ThreatSim_Core.dto.request;

import com.example.ThreatSim_Core.enums.AttackType;
import com.example.ThreatSim_Core.enums.HttpMethodType;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;

import java.util.Map;
import java.util.UUID;

public record SimulationRequest(

        @NotNull(message = "Application ID is required")
        UUID applicationId,

        @NotNull(message = "Attack type is required")
        AttackType attackType,

        @NotNull(message = "HTTP method is required")
        HttpMethodType httpMethod,

        @NotBlank(message = "Endpoint is required")
        String endpoint,

        Map<String, String> headers,

        String requestBody,

        @NotNull(message = "Expected HTTP status is required")
        Integer expectedStatus
) {
}
