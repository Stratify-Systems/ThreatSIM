package com.example.ThreatSim_Core.controller;

import com.example.ThreatSim_Core.dto.request.ApplicationRequest;
import com.example.ThreatSim_Core.dto.response.ApiResponse;
import com.example.ThreatSim_Core.dto.response.ApplicationResponse;
import com.example.ThreatSim_Core.service.ApplicationService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/applications")
@RequiredArgsConstructor
public class ApplicationController {

    private final ApplicationService applicationService;

    @PostMapping
    public ResponseEntity<ApiResponse<ApplicationResponse>> create(
            @Valid @RequestBody ApplicationRequest request) {
        ApplicationResponse response = applicationService.create(request);
        return ResponseEntity
                .status(HttpStatus.CREATED)
                .body(ApiResponse.success("Application registered", response));
    }

    @GetMapping
    public ResponseEntity<ApiResponse<List<ApplicationResponse>>> getAll() {
        return ResponseEntity.ok(ApiResponse.success(applicationService.getAll()));
    }

    @GetMapping("/{id}")
    public ResponseEntity<ApiResponse<ApplicationResponse>> getById(@PathVariable UUID id) {
        return ResponseEntity.ok(ApiResponse.success(applicationService.getById(id)));
    }

    @PutMapping("/{id}")
    public ResponseEntity<ApiResponse<ApplicationResponse>> update(
            @PathVariable UUID id,
            @Valid @RequestBody ApplicationRequest request) {
        return ResponseEntity.ok(
                ApiResponse.success("Application updated", applicationService.update(id, request)));
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<ApiResponse<Void>> delete(@PathVariable UUID id) {
        applicationService.delete(id);
        return ResponseEntity.ok(ApiResponse.success("Application deleted", null));
    }
}
