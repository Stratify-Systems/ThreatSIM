package com.example.ThreatSim_Core.entity;

import com.example.ThreatSim_Core.enums.AttackType;
import com.example.ThreatSim_Core.enums.HttpMethodType;
import jakarta.persistence.*;
import lombok.*;

import java.util.UUID;

@Entity
@Table(name = "simulations")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Simulation {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "application_id", nullable = false)
    private Application application;

    @Enumerated(EnumType.STRING)
    @Column(name = "attack_type", nullable = false)
    private AttackType attackType;

    @Enumerated(EnumType.STRING)
    @Column(name = "http_method", nullable = false)
    private HttpMethodType httpMethod;

    @Column(nullable = false)
    private String endpoint;

    @Column(columnDefinition = "text")
    private String headers;

    @Column(name = "request_body", columnDefinition = "text")
    private String requestBody;

    @Column(name = "expected_status", nullable = false)
    private int expectedStatus;

    @Builder.Default
    @Column(nullable = false)
    private boolean enabled = true;
}
