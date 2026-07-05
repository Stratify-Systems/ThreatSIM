package com.example.ThreatSim_Core.entity;

import com.example.ThreatSim_Core.enums.SimulationStatus;
import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.CreationTimestamp;

import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "simulation_results")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class SimulationResult {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "simulation_id", nullable = false)
    private Simulation simulation;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    private SimulationStatus status;

    @Column(name = "expected_status")
    private int expectedStatus;

    @Column(name = "actual_status")
    private int actualStatus;

    @Column(name = "response_body", columnDefinition = "text")
    private String responseBody;

    @Column(nullable = false)
    private long duration;

    @CreationTimestamp
    @Column(name = "executed_at", updatable = false)
    private LocalDateTime executedAt;
}
