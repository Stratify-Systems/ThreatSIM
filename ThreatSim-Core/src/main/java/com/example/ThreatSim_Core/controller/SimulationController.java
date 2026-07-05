package com.example.ThreatSim_Core.controller;

import com.example.ThreatSim_Core.dto.request.SimulationRequest;
import com.example.ThreatSim_Core.dto.response.ApiResponse;
import com.example.ThreatSim_Core.dto.response.SimulationResponse;
import com.example.ThreatSim_Core.dto.response.SimulationResultResponse;
import com.example.ThreatSim_Core.service.SimulationService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/simulations")
@RequiredArgsConstructor
public class SimulationController {

    private final SimulationService simulationService;

    @PostMapping
    public ResponseEntity<ApiResponse<SimulationResponse>> create(
            @Valid @RequestBody SimulationRequest request) {
        SimulationResponse response = simulationService.create(request);
        return ResponseEntity
                .status(HttpStatus.CREATED)
                .body(ApiResponse.success("Simulation created", response));
    }

    @GetMapping
    public ResponseEntity<ApiResponse<List<SimulationResponse>>> getAll() {
        return ResponseEntity.ok(ApiResponse.success(simulationService.getAll()));
    }

    @GetMapping("/{id}")
    public ResponseEntity<ApiResponse<SimulationResponse>> getById(@PathVariable UUID id) {
        return ResponseEntity.ok(ApiResponse.success(simulationService.getById(id)));
    }

    @PutMapping("/{id}")
    public ResponseEntity<ApiResponse<SimulationResponse>> update(
            @PathVariable UUID id,
            @Valid @RequestBody SimulationRequest request) {
        return ResponseEntity.ok(
                ApiResponse.success("Simulation updated", simulationService.update(id, request)));
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<ApiResponse<Void>> delete(@PathVariable UUID id) {
        simulationService.delete(id);
        return ResponseEntity.ok(ApiResponse.success("Simulation deleted", null));
    }

    @PostMapping("/{id}/run")
    public ResponseEntity<ApiResponse<SimulationResultResponse>> run(@PathVariable UUID id) {
        SimulationResultResponse result = simulationService.run(id);
        return ResponseEntity.ok(ApiResponse.success("Simulation executed", result));
    }

    @GetMapping("/{id}/results")
    public ResponseEntity<ApiResponse<List<SimulationResultResponse>>> getResults(
            @PathVariable UUID id) {
        return ResponseEntity.ok(ApiResponse.success(simulationService.getResults(id)));
    }
}
