package org.devMandali.magnetPlay.controller;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.GreeterGrpc;
import org.devMandali.magnetPlay.HelloResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.devMandali.magnetPlay.HelloRequest;

@RestController
@RequestMapping("/torrent")
public class AddTorrent {

    private final Logger logger = LoggerFactory.getLogger(AddTorrent.class);

    @GrpcClient("Greeter")
    private GreeterGrpc.GreeterBlockingStub greeterBlockingStub;

    @PostMapping
    public ResponseEntity<?> addTorrent(@RequestBody String torrentURL){
//        Handle Torrent URL
        HelloRequest request = HelloRequest.newBuilder().setName(torrentURL).build();

        HelloResponse response = greeterBlockingStub.sayHello(request);

        logger.info(response.getMessage());
        return ResponseEntity.ok(response.getMessage());
    }
}
