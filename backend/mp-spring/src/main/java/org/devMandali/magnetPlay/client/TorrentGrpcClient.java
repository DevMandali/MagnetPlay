package org.devMandali.magnetPlay.client;

import net.devh.boot.grpc.client.inject.GrpcClient;
import org.devMandali.magnetPlay.TorrentRequest;
import org.devMandali.magnetPlay.TorrentResponse;
import org.devMandali.magnetPlay.TorrentServiceGrpc;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;
import reactor.core.scheduler.Scheduler;

import java.util.concurrent.TimeUnit;
import java.util.function.Function;

@Component
public class TorrentGrpcClient {

    @GrpcClient("TorrentService")
    private TorrentServiceGrpc.TorrentServiceBlockingStub torrentServiceBlockingStub;

    @Autowired
    @Qualifier("grpcScheduler")
    private Scheduler grpcScheduler;

    @Value("${grpc.client.torrent.deadline-seconds:90}")
    private long deadlineSeconds;

    private static final Logger logger = LoggerFactory.getLogger(TorrentGrpcClient.class);

    // ─── AddTorrent ──────────────────────────────────────────────────────────

    public Mono<TorrentAddResponse> addTorrent(TorrentRequest request, Function<TorrentResponse, TorrentAddResponse> prepResponseFn) {
        return Mono.fromCallable(() -> torrentServiceBlockingStub
                .withDeadlineAfter(deadlineSeconds, TimeUnit.SECONDS)
                .addTorrent(request))
                .map(prepResponseFn)
                .subscribeOn(grpcScheduler)
                .doOnError(e -> logger.error("addTorrent gRPC error", e));
    }
}
