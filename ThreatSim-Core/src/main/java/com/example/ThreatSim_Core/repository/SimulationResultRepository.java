package com.example.ThreatSim_Core.repository;

import com.example.ThreatSim_Core.entity.SimulationResult;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.UUID;

public interface SimulationResultRepository extends JpaRepository<SimulationResult, UUID> {

    List<SimulationResult> findBySimulationIdOrderByExecutedAtDesc(UUID simulationId);
}
