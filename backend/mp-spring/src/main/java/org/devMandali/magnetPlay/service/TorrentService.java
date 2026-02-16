package org.devMandali.magnetPlay.service;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.TorrentRequest;
import org.devMandali.magnetPlay.TorrentResponse;
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

    public Map<String, Object> addTorrentToSession(TorrentAddRequest request) {
        TorrentRequest grpcRequest = TorrentRequest.newBuilder().setMagnetURL(request.magnet()).build();
        TorrentResponse response = torrentServiceBlockingStub.addTorrent(grpcRequest);
        logger.info(response.getName());
        return Map.of(
                "name", response.getName(),
                "status", response.getStatus().name(),
                "files", response.getFilesList().stream().toString()
        );
    }
}
