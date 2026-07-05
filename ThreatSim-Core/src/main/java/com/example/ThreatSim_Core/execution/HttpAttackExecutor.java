package com.example.ThreatSim_Core.execution;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

import java.time.Duration;

/**
 * HTTP-based attack executor that sends real HTTP requests to target applications
 * using Spring's RestClient.
 */
@Component
public class HttpAttackExecutor implements AttackExecutor {

    private static final Logger log = LoggerFactory.getLogger(HttpAttackExecutor.class);

    private final RestClient restClient;

    public HttpAttackExecutor() {
        this.restClient = RestClient.builder()
                .defaultHeader("User-Agent", "ThreatSIM/1.0")
                .build();
    }

    @Override
    public AttackResponse execute(AttackRequest request) {
        log.info("Executing attack: {} {}", request.method(), request.url());

        long startTime = System.currentTimeMillis();

        try {
            var requestSpec = restClient.method(HttpMethod.valueOf(request.method()))
                    .uri(request.url());

            // Add custom headers
            if (request.headers() != null) {
                request.headers().forEach(requestSpec::header);
            }

            // Add request body if present
            if (request.body() != null && !request.body().isBlank()) {
                requestSpec.contentType(MediaType.APPLICATION_JSON)
                        .body(request.body());
            }

            var response = requestSpec.exchange((req, res) -> {
                String body = new String(res.getBody().readAllBytes());
                return new AttackResponse(
                        res.getStatusCode().value(),
                        body,
                        0 // will be set below
                );
            });

            long duration = System.currentTimeMillis() - startTime;

            log.info("Response received: status={}, duration={}ms", response.statusCode(), duration);

            return new AttackResponse(response.statusCode(), response.body(), duration);

        } catch (Exception ex) {
            long duration = System.currentTimeMillis() - startTime;
            log.error("Attack execution failed: {}", ex.getMessage());

            // Return a special response for connection failures / timeouts
            return new AttackResponse(-1, "Error: " + ex.getMessage(), duration);
        }
    }
}
