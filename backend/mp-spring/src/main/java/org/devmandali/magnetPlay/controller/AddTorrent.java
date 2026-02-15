package org.devmandali.magnetPlay.controller;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.GreeterGrpc;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/torrent")
public class AddTorrent {

    @GrpcClient("Greeter")
    private GreeterGrpc.GreeterBlockingStub greeterBlockingStub;

    @PostMapping("/")
    public ResponseEntity<?> addTorrent(@RequestBody String torrentURL){
//        Handle Torrent URL
        return ResponseEntity.ok().build();
    }
}
