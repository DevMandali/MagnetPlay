package org.devMandali.magnetPlay.config;

import com.google.gson.Gson;
import net.devh.boot.grpc.client.channelfactory.GrpcChannelConfigurer;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import reactor.core.scheduler.Scheduler;
import reactor.core.scheduler.Schedulers;

import java.util.Map;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

@Configuration
public class GrpcConfig {

    /**
     * Custom Channel Configure is for the Go sidecar gRPC connection.
     *
     * Key settings:
     * - maxInboundMessageSize: raised to 512 MB since video chunks can be large
     * - keepAlive: prevents the connection from being torn down during long video streams
     * - flowControl: gRPC HTTP/2 flow control window sized to avoid head-of-line blocking
     */
    @Bean
    public GrpcChannelConfigurer customChannelConfigurer() {
        return (channelBuilder, name) -> {
            // service_config.json for retries.
            String serviceConfigJson = "{\n" +
                    "  \"methodConfig\": [\n" +
                    "    {\n" +
                    "      \"name\": [\n" +
                    "        { \"service\": \"\" } \n" +
                    "      ],\n" +
                    "      \"retryPolicy\": {\n" +
                    "        \"maxAttempts\": 3,\n" +
                    "        \"initialBackoff\": \"0.1s\",\n" +
                    "        \"maxBackoff\": \"1s\",\n" +
                    "        \"backoffMultiplier\": 2,\n" +
                    "        \"retryableStatusCodes\": [\n" +
                    "          \"UNAVAILABLE\",\n" +
                    "          \"DEADLINE_EXCEEDED\"\n" +
                    "        ]\n" +
                    "      }\n" +
                    "    }\n" +
                    "  ]\n" +
                    "}\n";

            // Convert JSON to Map for gRPC
            Map<String, ?> serviceConfig = new Gson().fromJson(serviceConfigJson, Map.class);

            channelBuilder
                    .maxInboundMessageSize(512 * 1024 * 1024)
                    .maxInboundMetadataSize(64 * 1024)
                    .keepAliveTime(30, TimeUnit.SECONDS)        // TODO Should be SYNC With Server side permitKeepAliveTime setting
                    .keepAliveTimeout(10, TimeUnit.SECONDS)
                    .keepAliveWithoutCalls(false)
                    .defaultServiceConfig(serviceConfig)
                    .enableRetry()
                    .maxRetryAttempts(3);
        };
    }

    /**
     * Dedicated thread pool for gRPC blocking stub calls bridged into WebFlux.
     * gRPC blocking stubs block threads — we isolate them from the Netty event loop
     * to prevent starving the reactive pipeline.
     */
    @Bean(name = "grpcScheduler")
    public Scheduler grpcScheduler() {
        AtomicInteger counter = new AtomicInteger(0);
        return Schedulers.fromExecutorService(
                Executors.newFixedThreadPool(
                    Runtime.getRuntime().availableProcessors() * 2,
                    r -> {
                    Thread t = new Thread(r, "grpc-stream" + counter.getAndIncrement());
                    t.setDaemon(true);
                    return t;
                }),
                "grpcScheduler"
        );
    }
}
