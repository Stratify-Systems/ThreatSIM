package com.example.ThreatSim_Core.execution;

/**
 * Data object representing the response received from the target application.
 */
public record AttackResponse(
        int statusCode,
        String body,
        long durationMs
) {
}
