package org.devMandali.magnetPlay.service;

import org.devMandali.magnetPlay.*;
import org.devMandali.magnetPlay.FileChunk;
import org.devMandali.magnetPlay.FileInfo;
import org.devMandali.magnetPlay.FileInfoRequest;
import org.devMandali.magnetPlay.FileInfoResponse;
import org.devMandali.magnetPlay.StreamRequest;
import org.devMandali.magnetPlay.TorrentRequest;
import org.devMandali.magnetPlay.TorrentResponse;
import org.devMandali.magnetPlay.client.TorrentGrpcClient;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.devMandali.magnetPlay.model.TorrentListResponse;
import org.devMandali.magnetPlay.model.HLSStartResponse;
import org.devMandali.magnetPlay.model.TorrentStatsResponse;
import org.devMandali.magnetPlay.util.ByteUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.io.buffer.DataBuffer;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Flux;
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

    public Mono<FileInfoResponse> getFileInfo(String torrentId, String fileId) {
        FileInfoRequest request = FileInfoRequest.newBuilder()
                .setInfoHash(torrentId)
                .setFileId(fileId)
                .build();

        return grpcClient.getFileInfoResponse(request);
    }


    /**
     * Opens a gRPC server-streaming call and wraps the chunk iterator as a Flux.
     *
     * How backpressure works here:
     *  - Flux.create with BUFFER sink allows the gRPC iterator to produce slightly
     *    ahead of HTTP consumption, which is fine for streaming video.
     *  - The subscribeOn(grpcScheduler) ensures Iterator.next() blocks on a dedicated
     *    thread, not the WebFlux event loop.
     *  - When the HTTP client cancels (user seeks or closes), the Flux is disposed,
     *    which interrupts the blocking thread, triggering CANCELLED on the gRPC stream.
     *
     * @param infoHash  torrent info hash (hex)
     * @param fileId  file ID (Mapped against FilePath) within the torrent
     * @param startByte inclusive start byte (for HTTP Range seek support)
     * @param endByte   exclusive end byte; -1 to stream to EOF
     */
    public Mono<TorrentStatsResponse> getTorrentStats(String infoHash, String fileId) {
        return grpcClient.getTorrentStats(infoHash, fileId)
                .map(r -> {
                    var s = r.getStats();
                    return new TorrentStatsResponse(
                            s.getFileId(),
                            s.getTotalSize(),
                            s.getDownloadedBytes(),
                            s.getCompletionPct(),
                            s.getDownloadSpeedBps(),
                            s.getSeeders(),
                            s.getPeers(),
                            s.getTrackers()
                    );
                })
                .doOnError(e -> logger.error("getTorrentStats error for {}/{}", infoHash, fileId, e));
    }

    public Mono<TorrentListResponse> listTorrents() {
        return grpcClient.listTorrents()
                .map(r -> {
                    var items = r.getTorrentsList().stream().map(t ->
                        new TorrentListResponse.TorrentListItem(
                            t.getTorrentId(), t.getName(), t.getState().name(),
                            t.getTotalSize(), t.getDownloadedBytes(),
                            t.getCompletionPct(), t.getDownloadSpeedBps(),
                            t.getFilesList().stream().map(f ->
                                new TorrentListResponse.FileItem(f.getId(), f.getName(), f.getSize())
                            ).toList()
                        )
                    ).toList();
                    return new TorrentListResponse(items);
                })
                .doOnError(e -> logger.error("listTorrents error", e));
    }

    public Mono<String> pauseTorrent(String infoHash) {
        return grpcClient.pauseTorrent(infoHash)
                .map(r -> r.getMessage())
                .doOnError(e -> logger.error("pauseTorrent error for {}", infoHash, e));
    }

    public Mono<String> resumeTorrent(String infoHash) {
        return grpcClient.resumeTorrent(infoHash)
                .map(r -> r.getMessage())
                .doOnError(e -> logger.error("resumeTorrent error for {}", infoHash, e));
    }

    public Mono<String> deleteTorrent(String infoHash, boolean deleteFiles) {
        return grpcClient.deleteTorrent(infoHash, deleteFiles)
                .map(r -> r.getMessage())
                .doOnError(e -> logger.error("deleteTorrent error for {}", infoHash, e));
    }

    public Flux<FileChunk> streamFile(String infoHash, String fileId, long startByte, long endByte) {
        StreamRequest request = StreamRequest.newBuilder()
                .setTorrentId(infoHash)
                .setFileId(fileId)
                .setStartByte(startByte)
                .setEndByte(endByte)
                .build();

        return grpcClient.streamFile(request);
    }

    public Mono<HLSStartResponse> startHLS(String infoHash, String fileId, double seekTimeSec) {
        return grpcClient.startHLS(infoHash, fileId, seekTimeSec)
                .map(r -> new HLSStartResponse(
                        r.getManifestUrl(),
                        r.getSuccess(),
                        r.getDurationSec(),
                        r.getAudioTracksList().stream()
                                .map(t -> new HLSStartResponse.AudioTrackDto(
                                        t.getIndex(), t.getLanguage(), t.getCodec(), t.getTitle()))
                                .toList()
                ))
                .doOnError(e -> logger.error("startHLS error for {}/{}", infoHash, fileId, e));
    }

    public Mono<HLSStartResponse> startRemux(String infoHash, String fileId, double seekTimeSec) {
        return grpcClient.startRemux(infoHash, fileId, seekTimeSec)
                .map(r -> new HLSStartResponse(
                        r.getManifestUrl(),
                        r.getSuccess(),
                        r.getDurationSec(),
                        r.getAudioTracksList().stream()
                                .map(t -> new HLSStartResponse.AudioTrackDto(
                                        t.getIndex(), t.getLanguage(), t.getCodec(), t.getTitle()))
                                .toList()
                ))
                .doOnError(e -> logger.error("startRemux error for {}/{}", infoHash, fileId, e));
    }

    public Mono<String> stopHLS(String infoHash, String fileId) {
        return grpcClient.stopHLS(infoHash, fileId)
                .map(r -> r.getSuccess() ? "stopped" : "not found")
                .doOnError(e -> logger.error("stopHLS error for {}/{}", infoHash, fileId, e));
    }
}
