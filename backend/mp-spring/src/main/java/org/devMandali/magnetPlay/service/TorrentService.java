package org.devMandali.magnetPlay.service;

import org.devMandali.magnetPlay.*;
import org.devMandali.magnetPlay.client.TorrentGrpcClient;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.devMandali.magnetPlay.util.ByteUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Mono;

import java.util.LinkedHashMap;

import java.util.function.Function;
import java.util.stream.Collectors;

@Service
public class TorrentService {
    private final Logger logger = LoggerFactory.getLogger(TorrentService.class);
    private final TorrentGrpcClient grpcClient;

    public TorrentService(TorrentGrpcClient grpcClient) {
        this.grpcClient = grpcClient;
    }

    private final Function<TorrentResponse, TorrentAddResponse> prepareTorrentAddResponseFn = (grpcResponse) -> new TorrentAddResponse(
            grpcResponse.getTorrentId(),
            grpcResponse.getName(),
            grpcResponse.getStatus().name(),
            grpcResponse.getFilesList().stream().collect(Collectors.toMap(
                    FileInfo::getId,
                    file -> String.format("%s => %s", file.getName(), ByteUtil.formatSize(file.getSize())),
                    (existing, replacement) -> existing,
                    LinkedHashMap::new
            ))
    );

    public Mono<TorrentAddResponse> addTorrentToSession(TorrentAddRequest request) {
        TorrentRequest grpcRequest = TorrentRequest.newBuilder().setMagnetUrl(request.magnet()).build();

        return grpcClient.addTorrent(grpcRequest, prepareTorrentAddResponseFn)
                .doOnSuccess(resp -> logger.debug("Torrent added: id={}, name={}, files={}", resp.torrentId(), resp.name(), resp.files()));
    }
}
