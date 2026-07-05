package com.example.ThreatSim_Core.execution;

import com.example.ThreatSim_Core.enums.SimulationStatus;
import org.springframework.stereotype.Component;

/**
 * Analyzes the response from an attack execution and determines PASS / FAIL / ERROR.
 * <p>
 * For MVP, comparison is based on HTTP status code only.
 * Future versions can add response body matching, header checks, and timing analysis.
 */
@Component
public class ResultAnalyzer {

    /**
     * Compares the expected status with the actual response.
     *
     * @param expectedStatus the HTTP status code the simulation expects
     * @param response       the actual response from the target
     * @return PASS if match, ERROR if request failed, FAIL otherwise
     */
    public SimulationStatus analyze(int expectedStatus, AttackResponse response) {
        // Connection failure or timeout
        if (response.statusCode() < 0) {
            return SimulationStatus.ERROR;
        }

        return response.statusCode() == expectedStatus
                ? SimulationStatus.PASS
                : SimulationStatus.FAIL;
    }
}
