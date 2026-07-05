package com.example.ThreatSim_Core.repository;

import com.example.ThreatSim_Core.entity.Application;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.UUID;

public interface ApplicationRepository extends JpaRepository<Application, UUID> {
}
