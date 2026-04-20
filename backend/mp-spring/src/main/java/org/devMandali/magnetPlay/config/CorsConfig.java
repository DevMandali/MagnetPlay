package org.devMandali.magnetPlay.config;

import com.google.common.net.HttpHeaders;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpMethod;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.reactive.CorsWebFilter;
import org.springframework.web.cors.reactive.UrlBasedCorsConfigurationSource;

import java.util.List;

@Configuration
public class CorsConfig {
    @Value("${cors.allowed-origins:http://localhost:3000}")
    private List<String> allowedOrigins;

    /**
     * CORS filter for WebFlux.
     *
     * Critical headers exposed to the browser:
     * - Content-Range: tells video.js the byte range and total file size
     * - Accept-Ranges: signals that this endpoint supports seeking
     * - Content-Length: lets the browser show accurate progress
     *
     * Without exposing these, the browser silently blocks seeking.
     */
    @Bean
    public CorsWebFilter corsWebFilter() {
        CorsConfiguration config = new CorsConfiguration();

        config.setAllowedOrigins(allowedOrigins);
        config.setAllowedMethods(List.of(
                HttpMethod.GET.name(),
                HttpMethod.POST.name(),
                HttpMethod.DELETE.name(),
                HttpMethod.OPTIONS.name(),
                HttpMethod.HEAD.name()
        ));
        config.setAllowedHeaders(List.of(
                HttpHeaders.RANGE,
                HttpHeaders.CONTENT_TYPE,
                HttpHeaders.AUTHORIZATION
        ));
        config.setExposedHeaders(List.of(
                HttpHeaders.CONTENT_TYPE,
                HttpHeaders.ACCEPT_RANGES,
                HttpHeaders.CONTENT_LENGTH,
                HttpHeaders.CONTENT_RANGE
        ));
        // TODO Remove once frontend server is configured
        // This allows the browser's "null" origin (local file://)
        config.addAllowedOrigin("null");

        config.setAllowCredentials(false);
        config.setMaxAge(3600L);

        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/v1/torrent/**", config);

        return new CorsWebFilter(source);
    }
}
