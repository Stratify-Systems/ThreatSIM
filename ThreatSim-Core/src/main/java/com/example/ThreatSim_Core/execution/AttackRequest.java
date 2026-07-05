package com.example.ThreatSim_Core.execution;

/**
 * Data object representing the HTTP request to be sent to the target application.
 */
public record AttackRequest(
        String url,
        String method,
        java.util.Map<String, String> headers,
        String body
) {
}
