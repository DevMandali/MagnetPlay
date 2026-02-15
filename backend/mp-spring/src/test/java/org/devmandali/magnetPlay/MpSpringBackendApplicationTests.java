package org.devmandali.magnetPlay;

import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.context.annotation.Bean;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

@SpringBootTest
class MpSpringBackendApplicationTests {

    private static final Logger logger = LoggerFactory.getLogger(MpSpringBackendApplicationTests.class);

    @Test
    void contextLoads() {
    }
    /**
     * Run gRPC client calls on application startup
     */
//    @Bean
//    public CommandLineRunner run(GrpcClient greeterClient) {
//        return args -> {
//            logger.info("=== Starting gRPC Client Demo ===");
//
//            // Test with different names
//            String[] names = {"Alice", "Bob", "World"};
//            ExecutorService executorService = Executors.newFixedThreadPool(5);
//            for (String name : names) {
//                executorService.submit(() -> {
//                    try {
//                        String response = greeterClient.SayHello(name);
//                        logger.info("✓ Success: {}", response);
//                    } catch (Exception e) {
//                        logger.error("✗ Failed for {}: {}", name, e.getMessage());
//                    }
//                });
//            }
//
//            logger.info("=== gRPC Client Demo Complete ===");
//        };
//    }
}
