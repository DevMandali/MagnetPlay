package org.devMandali.magnetPlay.controller;
import com.google.common.net.HttpHeaders;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.media.Content;
import io.swagger.v3.oas.annotations.media.Schema;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import org.devMandali.magnetPlay.FileInfoResponse;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.util.TorrentUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.io.buffer.DataBuffer;
import org.springframework.core.io.buffer.DataBufferFactory;
import org.springframework.core.io.buffer.DefaultDataBufferFactory;
import org.springframework.http.HttpRange;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.http.server.reactive.ServerHttpRequest;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

import java.util.List;

@RestController
@RequestMapping("/v1/torrent")
public class TorrentController {

    private final Logger logger = LoggerFactory.getLogger(TorrentController.class);
    private final DataBufferFactory bufferFactory = new DefaultDataBufferFactory();
    private final TorrentService service;

    public TorrentController(TorrentService service) {
        this.service = service;
    }

    @Operation(summary = "Add Torrent URL")
    @Tag(name = "Add-Torrent")
    @ApiResponses(value = {
            @ApiResponse(responseCode = "201", description = "Added the torrent",
                    content = { @Content(mediaType = "application/json",
                            schema = @Schema(implementation = TorrentAddResponse.class)) }),
            @ApiResponse(responseCode = "400", description = "Invalid URL supplied",
                    content = @Content),
    })
    @PostMapping(value = "/add", consumes = {"application/json"})
    @io.swagger.v3.oas.annotations.parameters.RequestBody(required = true)
    public Mono<ResponseEntity<TorrentAddResponse>> addTorrent(@Valid @RequestBody TorrentAddRequest request){
        logger.info("Got new torrent request");

        if(logger.isDebugEnabled())  {
            logger.debug("Torrent Magnet Link: {}", request.magnet());

            String infoHash = TorrentUtil.extractInfoHash(request.magnet());
            logger.debug("Extracted infoHash: {}", infoHash);
        }

        return service.addTorrentToSession(request)
                .map(resp -> ResponseEntity.status(HttpStatus.CREATED).body(resp))
                .doOnError(e -> logger.error("Failed to add torrent: {}", e.getMessage(), e))
                .onErrorResume(e -> {
                    if (e instanceof IllegalArgumentException) {
                        return Mono.just(ResponseEntity.badRequest().build());
                    }
                    return Mono.just(ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build());
                });
    }

    @GetMapping("/stream/{infoHash}")
    public Mono<ResponseEntity<Flux<DataBuffer>>> stream(
            @PathVariable("infoHash") String infoHash,
            @RequestParam("fileId") String fileId,
            ServerHttpRequest request) {
        return service.getFileInfo(infoHash, fileId)
                .map(info -> buildStreamResponse(info, infoHash, fileId, request))
                .doOnError(e -> logger.error("stream error for {}/{}", infoHash, fileId, e))
                .onErrorReturn(ResponseEntity.status(HttpStatus.NOT_FOUND).build());
    }

    private ResponseEntity<Flux<DataBuffer>> buildStreamResponse(
            FileInfoResponse info,
            String infoHash,
            String fileId,
            ServerHttpRequest request) {
        logger.info("total file size {}", info.getTotalSize());
        long fileSize = info.getTotalSize();
        String mimeType = info.getMimeType();

        // ── Parse Range header ─────────────────────────────────────────────
        //
        // The browser sends Range: bytes=N-M or Range: bytes=N-
        // video.js computes N from the MP4 moov atom's sample table.
        // We never need to convert seconds to bytes — the browser does that.
        //
        List<HttpRange> ranges = request.getHeaders().getRange();

        long startByte;
        long endByte;
        HttpStatus responseStatus;

        if(ranges.isEmpty()) {
            // No Range header — full file request (initial load or non-seek-capable client)
            startByte = 0;
            endByte = fileSize - 1;
            responseStatus = HttpStatus.OK;
        } else {
            // Partial content request — browser is seeking or resuming
            HttpRange range = ranges.get(0);
            startByte = range.getRangeStart(fileSize);
            endByte = range.getRangeEnd(fileSize);
            responseStatus = HttpStatus.PARTIAL_CONTENT;

            logger.debug("range request: bytes={}-{}/{} for {}/{}",
                    startByte, endByte, fileSize, infoHash, fileId);
        }

        long contentLength = endByte - startByte + 1;

        // ── Build gRPC stream → DataBuffer Flux ───────────────────────────
        //
        // endByte in our gRPC contract is exclusive, but HTTP Range is inclusive.
        // So we pass endByte + 1 to the gRPC call.
        //
        Flux<DataBuffer> dataStream = service
                .streamFile(infoHash, fileId, startByte, endByte + 1)
                .map(chunk -> {
                    logger.info("[{}] chunk retrieved from sidecar with offset : {} and byte length: {}", Thread.currentThread().getName(), chunk.getOffset(), chunk.getData().toByteArray().length);
                    byte[] bytes = chunk.getData().toByteArray();
                    DataBuffer buffer = bufferFactory.allocateBuffer(bytes.length);
                    buffer.write(bytes);
                    return buffer;
                })
                // onBackpressureBuffer: if the HTTP client is slower than gRPC delivery,
                // buffer up to 32 chunks (32 × 256KB = 8 MB max memory per stream)
                // before applying flow control upstream.
                .onBackpressureBuffer(32)
                .doOnComplete(() -> logger.debug("stream complete: {}/{} bytes={}-{}", infoHash, fileId, startByte, endByte))
                .doOnCancel(() -> logger.debug("stream cancelled: {}/{} at startByte={}", infoHash, fileId, startByte))
                .doOnError(e -> logger.error("stream error: {}/{}", infoHash, fileId, e));

        // ── Build HTTP response ────────────────────────────────────────────
        //
        // These headers are what make video seeking work:
        //   Accept-Ranges: bytes → tells video.js this endpoint supports seeking
        //   Content-Range: bytes N-M/Total → tells browser where this chunk sits
        //   Content-Length → lets browser show accurate buffering progress
        //
        return ResponseEntity
                .status(responseStatus)
                .header(HttpHeaders.CONTENT_TYPE, mimeType)
                .header(HttpHeaders.ACCEPT_RANGES)
                .header(HttpHeaders.CONTENT_LENGTH, String.valueOf(contentLength))
                .header(HttpHeaders.CONTENT_RANGE, "bytes " + startByte + "-" + endByte + "/" + fileSize)
                .header(HttpHeaders.CACHE_CONTROL, "no-cache, no-store")
                .body(dataStream);

    }
}
