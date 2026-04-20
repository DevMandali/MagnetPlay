package org.devMandali.magnetPlay.model;

import java.util.List;

public record TorrentListResponse(List<TorrentListItem> torrents) {
    public record TorrentListItem(
        String torrentId,
        String name,
        String state,
        long   totalSize,
        long   downloadedBytes,
        double completionPct,
        double downloadSpeedBps,
        List<FileItem> files
    ) {}

    public record FileItem(String id, String name, long size) {}
}
