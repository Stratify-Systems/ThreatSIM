package com.example.ThreatSim_Core.service;

import com.example.ThreatSim_Core.dto.request.ApplicationRequest;
import com.example.ThreatSim_Core.dto.response.ApplicationResponse;
import com.example.ThreatSim_Core.entity.Application;
import com.example.ThreatSim_Core.exception.ResourceNotFoundException;
import com.example.ThreatSim_Core.repository.ApplicationRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class ApplicationService {

    private final ApplicationRepository applicationRepository;

    @Transactional
    public ApplicationResponse create(ApplicationRequest request) {
        Application app = Application.builder()
                .name(request.name())
                .baseUrl(normalizeUrl(request.baseUrl()))
                .build();

        return toResponse(applicationRepository.save(app));
    }

    @Transactional(readOnly = true)
    public List<ApplicationResponse> getAll() {
        return applicationRepository.findAll().stream()
                .map(this::toResponse)
                .toList();
    }

    @Transactional(readOnly = true)
    public ApplicationResponse getById(UUID id) {
        return toResponse(findOrThrow(id));
    }

    @Transactional
    public ApplicationResponse update(UUID id, ApplicationRequest request) {
        Application app = findOrThrow(id);
        app.setName(request.name());
        app.setBaseUrl(normalizeUrl(request.baseUrl()));

        return toResponse(applicationRepository.save(app));
    }

    @Transactional
    public void delete(UUID id) {
        if (!applicationRepository.existsById(id)) {
            throw new ResourceNotFoundException("Application", id);
        }
        applicationRepository.deleteById(id);
    }

    // --- Internal helpers ---

    Application findOrThrow(UUID id) {
        return applicationRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Application", id));
    }

    private String normalizeUrl(String url) {
        // Strip trailing slash for consistent URL building
        return url.endsWith("/") ? url.substring(0, url.length() - 1) : url;
    }

    private ApplicationResponse toResponse(Application app) {
        return new ApplicationResponse(
                app.getId(),
                app.getName(),
                app.getBaseUrl(),
                app.getCreatedAt()
        );
    }
}
