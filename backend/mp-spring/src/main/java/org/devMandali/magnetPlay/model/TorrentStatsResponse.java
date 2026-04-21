package org.devMandali.magnetPlay.model;

public record TorrentStatsResponse(
        String fileId,
        long   totalSize,
        long   downloadedBytes,
        double completionPct,
        double downloadSpeedBps,
        int    seeders,
        int    peers,
        int    trackers
) {}
