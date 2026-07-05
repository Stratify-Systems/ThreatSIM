package com.example.ThreatSim_Core.service;

import com.example.ThreatSim_Core.dto.request.SimulationRequest;
import com.example.ThreatSim_Core.dto.response.SimulationResponse;
import com.example.ThreatSim_Core.dto.response.SimulationResultResponse;
import com.example.ThreatSim_Core.entity.Application;
import com.example.ThreatSim_Core.entity.Simulation;
import com.example.ThreatSim_Core.entity.SimulationResult;
import com.example.ThreatSim_Core.enums.SimulationStatus;
import com.example.ThreatSim_Core.exception.ResourceNotFoundException;
import com.example.ThreatSim_Core.execution.AttackExecutor;
import com.example.ThreatSim_Core.execution.AttackRequest;
import com.example.ThreatSim_Core.execution.AttackResponse;
import com.example.ThreatSim_Core.execution.ResultAnalyzer;
import com.example.ThreatSim_Core.repository.SimulationRepository;
import com.example.ThreatSim_Core.repository.SimulationResultRepository;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class SimulationService {

    private static final Logger log = LoggerFactory.getLogger(SimulationService.class);

    private final SimulationRepository simulationRepository;
    private final SimulationResultRepository resultRepository;
    private final ApplicationService applicationService;
    private final AttackExecutor attackExecutor;
    private final ResultAnalyzer resultAnalyzer;
    private final ObjectMapper objectMapper;

    // --- CRUD ---

    @Transactional
    public SimulationResponse create(SimulationRequest request) {
        Application app = applicationService.findOrThrow(request.applicationId());

        Simulation sim = Simulation.builder()
                .application(app)
                .attackType(request.attackType())
                .httpMethod(request.httpMethod())
                .endpoint(request.endpoint())
                .headers(toJson(request.headers()))
                .requestBody(request.requestBody())
                .expectedStatus(request.expectedStatus())
                .enabled(true)
                .build();

        return toResponse(simulationRepository.save(sim));
    }

    @Transactional(readOnly = true)
    public List<SimulationResponse> getAll() {
        return simulationRepository.findAll().stream()
                .map(this::toResponse)
                .toList();
    }

    @Transactional(readOnly = true)
    public SimulationResponse getById(UUID id) {
        return toResponse(findOrThrow(id));
    }

    @Transactional
    public SimulationResponse update(UUID id, SimulationRequest request) {
        Simulation sim = findOrThrow(id);
        Application app = applicationService.findOrThrow(request.applicationId());

        sim.setApplication(app);
        sim.setAttackType(request.attackType());
        sim.setHttpMethod(request.httpMethod());
        sim.setEndpoint(request.endpoint());
        sim.setHeaders(toJson(request.headers()));
        sim.setRequestBody(request.requestBody());
        sim.setExpectedStatus(request.expectedStatus());

        return toResponse(simulationRepository.save(sim));
    }

    @Transactional
    public void delete(UUID id) {
        if (!simulationRepository.existsById(id)) {
            throw new ResourceNotFoundException("Simulation", id);
        }
        simulationRepository.deleteById(id);
    }

    // --- Execution ---

    @Transactional
    public SimulationResultResponse run(UUID simulationId) {
        Simulation sim = findOrThrow(simulationId);

        // 1. Build the full URL
        String fullUrl = sim.getApplication().getBaseUrl() + sim.getEndpoint();

        // 2. Build the attack request
        AttackRequest attackRequest = new AttackRequest(
                fullUrl,
                sim.getHttpMethod().name(),
                fromJson(sim.getHeaders()),
                sim.getRequestBody()
        );

        // 3. Execute the attack
        AttackResponse attackResponse = attackExecutor.execute(attackRequest);

        // 4. Analyze the result
        SimulationStatus status = resultAnalyzer.analyze(
                sim.getExpectedStatus(),
                attackResponse
        );

        // 5. Persist the result
        SimulationResult result = SimulationResult.builder()
                .simulation(sim)
                .status(status)
                .expectedStatus(sim.getExpectedStatus())
                .actualStatus(attackResponse.statusCode())
                .responseBody(truncate(attackResponse.body(), 5000))
                .duration(attackResponse.durationMs())
                .build();

        SimulationResult saved = resultRepository.save(result);

        log.info("Simulation {} completed: {} (expected={}, actual={}, duration={}ms)",
                simulationId, status, sim.getExpectedStatus(),
                attackResponse.statusCode(), attackResponse.durationMs());

        return toResultResponse(saved);
    }

    // --- Results ---

    @Transactional(readOnly = true)
    public List<SimulationResultResponse> getResults(UUID simulationId) {
        if (!simulationRepository.existsById(simulationId)) {
            throw new ResourceNotFoundException("Simulation", simulationId);
        }
        return resultRepository.findBySimulationIdOrderByExecutedAtDesc(simulationId).stream()
                .map(this::toResultResponse)
                .toList();
    }

    @Transactional(readOnly = true)
    public SimulationResultResponse getResultById(UUID resultId) {
        SimulationResult result = resultRepository.findById(resultId)
                .orElseThrow(() -> new ResourceNotFoundException("SimulationResult", resultId));
        return toResultResponse(result);
    }

    // --- Internal helpers ---

    private Simulation findOrThrow(UUID id) {
        return simulationRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Simulation", id));
    }

    private SimulationResponse toResponse(Simulation sim) {
        return new SimulationResponse(
                sim.getId(),
                sim.getApplication().getId(),
                sim.getApplication().getName(),
                sim.getAttackType(),
                sim.getHttpMethod(),
                sim.getEndpoint(),
                fromJson(sim.getHeaders()),
                sim.getRequestBody(),
                sim.getExpectedStatus(),
                sim.isEnabled()
        );
    }

    private SimulationResultResponse toResultResponse(SimulationResult result) {
        return new SimulationResultResponse(
                result.getId(),
                result.getSimulation().getId(),
                result.getStatus(),
                result.getExpectedStatus(),
                result.getActualStatus(),
                result.getResponseBody(),
                result.getDuration(),
                result.getExecutedAt()
        );
    }

    private String toJson(Map<String, String> map) {
        if (map == null || map.isEmpty()) return null;
        try {
            return objectMapper.writeValueAsString(map);
        } catch (JsonProcessingException e) {
            throw new IllegalArgumentException("Invalid headers format", e);
        }
    }

    private Map<String, String> fromJson(String json) {
        if (json == null || json.isBlank()) return Collections.emptyMap();
        try {
            return objectMapper.readValue(json, new TypeReference<>() {});
        } catch (JsonProcessingException e) {
            return Collections.emptyMap();
        }
    }

    private String truncate(String value, int maxLength) {
        if (value == null) return null;
        return value.length() > maxLength ? value.substring(0, maxLength) + "..." : value;
    }
}
