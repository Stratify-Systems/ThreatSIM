package com.example.ThreatSim_Core.dto.request;

import jakarta.validation.constraints.NotBlank;

public record ApplicationRequest(

        @NotBlank(message = "Application name is required")
        String name,

        @NotBlank(message = "Base URL is required")
        String baseUrl
) {
}
