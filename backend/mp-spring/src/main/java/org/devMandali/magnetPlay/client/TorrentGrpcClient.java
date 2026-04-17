package org.devMandali.magnetPlay.client;

import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.*;
import org.devMandali.magnetPlay.FileChunk;
import org.devMandali.magnetPlay.FileInfoRequest;
import org.devMandali.magnetPlay.FileInfoResponse;
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
