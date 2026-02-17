package org.devMandali.magnetPlay.service;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.*;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.devMandali.magnetPlay.util.ByteUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.util.LinkedHashMap;

import java.util.stream.Collectors;

@Service
public class TorrentService {

    @GrpcClient("TorrentService")
    private TorrentServiceGrpc.TorrentServiceBlockingStub torrentServiceBlockingStub;

    private final Logger logger = LoggerFactory.getLogger(TorrentService.class);

    public TorrentAddResponse addTorrentToSession(TorrentAddRequest request) {
        TorrentRequest grpcRequest = TorrentRequest.newBuilder().setMagnetUrl(request.magnet()).build();
        TorrentResponse response = torrentServiceBlockingStub.addTorrent(grpcRequest);
        logger.debug("Torrent added: id={}, name={}", response.getTorrentId(), response.getName());

        return new TorrentAddResponse(
                response.getTorrentId(),
                response.getName(),
                response.getStatus().name(),
                response.getFilesList().stream().collect(Collectors.toMap(
                        FileInfo::getId,
                        file -> String.format("%s => %s", file.getName(), ByteUtil.formatSize(file.getSize())),
                        (existing, replacement) -> existing,
                        LinkedHashMap::new
                ))
        );
    }
}
