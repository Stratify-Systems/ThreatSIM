package com.example.ThreatSim_Core.repository;

import com.example.ThreatSim_Core.entity.Simulation;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.UUID;

public interface SimulationRepository extends JpaRepository<Simulation, UUID> {

    List<Simulation> findByApplicationId(UUID applicationId);
}
