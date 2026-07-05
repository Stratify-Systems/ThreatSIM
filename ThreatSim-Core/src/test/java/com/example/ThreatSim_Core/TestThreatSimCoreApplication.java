package com.example.ThreatSim_Core;

import org.springframework.boot.SpringApplication;

public class TestThreatSimCoreApplication {

	public static void main(String[] args) {
		SpringApplication.from(ThreatSimCoreApplication::main).with(TestcontainersConfiguration.class).run(args);
	}

}
