package org.devMandali.magnetPlay.client;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.*;
import org.devMandali.magnetPlay.FileChunk;
import org.devMandali.magnetPlay.FileInfoRequest;
import org.devMandali.magnetPlay.FileInfoResponse;
import org.devMandali.magnetPlay.GetTorrentStatsRequest;
import org.devMandali.magnetPlay.GetTorrentStatsResponse;
import org.devMandali.magnetPlay.ListTorrentsRequest;
import org.devMandali.magnetPlay.ListTorrentsResponse;
import org.devMandali.magnetPlay.PauseTorrentRequest;
import org.devMandali.magnetPlay.PauseTorrentResponse;
import org.devMandali.magnetPlay.ResumeTorrentRequest;
import org.devMandali.magnetPlay.ResumeTorrentResponse;
import org.devMandali.magnetPlay.DeleteTorrentRequest;
import org.devMandali.magnetPlay.DeleteTorrentResponse;
import org.devMandali.magnetPlay.HLSRequest;
import org.devMandali.magnetPlay.HLSResponse;
import org.devMandali.magnetPlay.StreamRequest;
import org.devMandali.magnetPlay.TorrentRequest;
import org.devMandali.magnetPlay.TorrentResponse;
import org.devMandali.magnetPlay.TorrentServiceGrpc;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;
import reactor.core.scheduler.Scheduler;

import java.util.Iterator;
import java.util.concurrent.TimeUnit;
import java.util.function.Function;

@Component
public class TorrentGrpcClient {

    private final TorrentServiceGrpc.TorrentServiceBlockingStub torrentServiceBlockingStub;
    private final Scheduler grpcScheduler;

    public TorrentGrpcClient(
            @GrpcClient("TorrentService") TorrentServiceGrpc.TorrentServiceBlockingStub torrentServiceBlockingStub,
            @Qualifier("grpcScheduler") Scheduler grpcScheduler
    ) {
        this.torrentServiceBlockingStub = torrentServiceBlockingStub;
        this.grpcScheduler = grpcScheduler;
    }

    @Value("${grpc.client.torrent.add.deadline-seconds:90}")
    private long addDeadlineSeconds;

    @Value("${grpc.client.torrent.stream.deadline-hours:12}")
    private long streamDeadlineHours;

    private static final Logger logger = LoggerFactory.getLogger(TorrentGrpcClient.class);

    // ─── AddTorrent ──────────────────────────────────────────────────────────

    public Mono<TorrentAddResponse> addTorrent(TorrentRequest request, Function<TorrentResponse, TorrentAddResponse> prepResponseFn) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(addDeadlineSeconds, TimeUnit.SECONDS)
                .addTorrent(request))
                .map(prepResponseFn)
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("addTorrent gRPC error", e));
    }

    // ─── GetFileInfo ─────────────────────────────────────────────────────────

    public Mono<FileInfoResponse> getFileInfoResponse(FileInfoRequest request) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .getFileInfo(request))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("getFileInfo gRPC error for {}/{}", request.getInfoHash(), request.getFileId(), e));
    }

    // ─── GetTorrentStats ─────────────────────────────────────────────────────

    public Mono<GetTorrentStatsResponse> getTorrentStats(String infoHash, String fileId) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(5, TimeUnit.SECONDS)
                .getTorrentStats(GetTorrentStatsRequest.newBuilder()
                        .setInfoHash(infoHash)
                        .setFileId(fileId)
                        .build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("getTorrentStats gRPC error for {}/{}", infoHash, fileId, e));
    }

    // ─── ListTorrents ────────────────────────────────────────────────────────

    public Mono<ListTorrentsResponse> listTorrents() {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .listTorrents(ListTorrentsRequest.newBuilder().build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("listTorrents gRPC error", e));
    }

    // ─── PauseTorrent ────────────────────────────────────────────────────────

    public Mono<PauseTorrentResponse> pauseTorrent(String infoHash) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .pauseTorrent(PauseTorrentRequest.newBuilder()
                        .setInfoHash(infoHash).build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("pauseTorrent gRPC error for {}", infoHash, e));
    }

    // ─── ResumeTorrent ───────────────────────────────────────────────────────

    public Mono<ResumeTorrentResponse> resumeTorrent(String infoHash) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .resumeTorrent(ResumeTorrentRequest.newBuilder()
                        .setInfoHash(infoHash).build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("resumeTorrent gRPC error for {}", infoHash, e));
    }

    // ─── DeleteTorrent ───────────────────────────────────────────────────────

    public Mono<DeleteTorrentResponse> deleteTorrent(String infoHash) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .deleteTorrent(DeleteTorrentRequest.newBuilder()
                        .setInfoHash(infoHash)
                        .build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("deleteTorrent gRPC error for {}", infoHash, e));
    }

    // ─── StartHLS ────────────────────────────────────────────────────────────

    public Mono<HLSResponse> startHLS(String infoHash, String fileId, double seekTimeSec) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(35, TimeUnit.SECONDS)
                .startHLS(HLSRequest.newBuilder()
                        .setInfoHash(infoHash)
                        .setFileId(fileId)
                        .setSeekTimeSec(seekTimeSec)
                        .build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("startHLS gRPC error for {}/{}", infoHash, fileId, e));
    }

    // ─── StartRemux ──────────────────────────────────────────────────────────

    public Mono<HLSResponse> startRemux(String infoHash, String fileId, double seekTimeSec) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(15, TimeUnit.SECONDS)
                .startRemux(HLSRequest.newBuilder()
                        .setInfoHash(infoHash)
                        .setFileId(fileId)
                        .setSeekTimeSec(seekTimeSec)
                        .build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("startRemux gRPC error for {}/{}", infoHash, fileId, e));
    }

    // ─── StopHLS ─────────────────────────────────────────────────────────────

    public Mono<HLSResponse> stopHLS(String infoHash, String fileId) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(10, TimeUnit.SECONDS)
                .stopHLS(HLSRequest.newBuilder()
                        .setInfoHash(infoHash)
                        .setFileId(fileId)
                        .build()))
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("stopHLS gRPC error for {}/{}", infoHash, fileId, e));
    }

    // ─── StreamFile ──────────────────────────────────────────────────────────
    public Flux<FileChunk> streamFile(StreamRequest request) {
        return Flux.<FileChunk, Iterator<FileChunk>>generate(
                () -> torrentServiceBlockingStub
                        .withDeadlineAfter(streamDeadlineHours, TimeUnit.HOURS)
                        .streamFile(request),
                (iter, sink) -> {
                    try {
                        if (iter.hasNext()) {
                            sink.next(iter.next());
                        } else {
                            sink.complete();
                        }
                    } catch (StatusRuntimeException e) {
                        if (e.getStatus().getCode() == Status.Code.CANCELLED) {
                            logger.debug("gRPC stream cancelled for {}/{}", request.getTorrentId(), request.getFileId());
                            sink.complete();
                        } else {
                            logger.error("gRPC stream error for {}/{}", request.getTorrentId(), request.getFileId());
                            sink.error(e);
                        }
                    } catch (Exception e) {
                        sink.error(e);
                    }
                    return iter;
                })
                .subscribeOn(grpcScheduler);
    }
}
