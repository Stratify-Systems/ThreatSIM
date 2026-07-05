package com.example.ThreatSim_Core.controller;

import com.example.ThreatSim_Core.dto.response.ApiResponse;
import com.example.ThreatSim_Core.dto.response.SimulationResultResponse;
import com.example.ThreatSim_Core.service.SimulationService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/api/v1/results")
@RequiredArgsConstructor
public class ResultController {

    private final SimulationService simulationService;

    @GetMapping("/{id}")
    public ResponseEntity<ApiResponse<SimulationResultResponse>> getById(@PathVariable UUID id) {
        return ResponseEntity.ok(ApiResponse.success(simulationService.getResultById(id)));
    }
}
