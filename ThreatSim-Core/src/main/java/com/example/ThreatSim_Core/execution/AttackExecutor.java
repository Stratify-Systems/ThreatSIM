package com.example.ThreatSim_Core.execution;

/**
 * Strategy interface for executing attack requests against target applications.
 * <p>
 * The default implementation uses HTTP (RestClient). Future implementations
 * could execute via Docker containers, remote agents, or Kubernetes jobs.
 */
public interface AttackExecutor {

    /**
     * Sends the attack request to the target and returns the response.
     *
     * @param request the fully built attack request
     * @return the response from the target application
     */
    AttackResponse execute(AttackRequest request);
}
