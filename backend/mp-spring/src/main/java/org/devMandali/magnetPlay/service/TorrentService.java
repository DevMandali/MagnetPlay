package org.devMandali.magnetPlay.service;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.SessionInfo;
import org.devMandali.magnetPlay.TorrentRequest;
import org.devMandali.magnetPlay.TorrentServiceGrpc;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.util.Map;

@Service
public class TorrentService {

    @GrpcClient("TorrentService")
    private TorrentServiceGrpc.TorrentServiceBlockingStub torrentServiceBlockingStub;

    private final Logger logger = LoggerFactory.getLogger(TorrentService.class);

    public Map<String, String> addTorrentToSession(TorrentAddRequest request) {
        TorrentRequest grpcRequest = TorrentRequest.newBuilder().setMagnetURL(request.magnet()).build();
        SessionInfo response = torrentServiceBlockingStub.addTorrent(grpcRequest);
        logger.info(response.getMessage());
        return Map.of(
                "name", response.getMessage(),
                "status", response.getStatus().name()
        );
    }
}
