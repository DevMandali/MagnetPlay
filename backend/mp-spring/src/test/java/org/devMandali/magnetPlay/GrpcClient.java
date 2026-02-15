package org.devMandali.magnetPlay;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;


@Service
public class GrpcClient {

    private static final Logger logger = LoggerFactory.getLogger(GrpcClient.class);

    private ManagedChannel channel;
    private GreeterGrpc.GreeterBlockingStub blockingStub;

    @PostConstruct
    void init(){
        channel = ManagedChannelBuilder.forAddress("localhost",50051).usePlaintext().build();
        blockingStub = GreeterGrpc.newBlockingStub(channel);
        logger.info("gRPC channel initialized to localhost:50051");
    }

    public String SayHello(String name){
        try{
            HelloRequest helloRequest = HelloRequest.newBuilder().setName(name).build();
            HelloResponse helloResponse = blockingStub.sayHello(helloRequest);
            String message = helloResponse.getMessage();
            logger.info("Received response: {}", message);
            return message;
        } catch(Exception e){
            logger.error(e.getMessage());
            throw new RuntimeException();
        }
    }
    /**
     * Shutdown the channel when the bean is destroyed
     */
    @PreDestroy
    public void cleanup() {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdown();
            logger.info("gRPC channel shutdown");
        }
    }
}
