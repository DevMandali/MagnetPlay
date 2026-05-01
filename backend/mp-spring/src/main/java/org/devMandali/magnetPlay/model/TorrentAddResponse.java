package org.devMandali.magnetPlay.model;

import io.swagger.v3.oas.annotations.media.Schema;
import java.io.Serializable;
import java.util.List;

public record TorrentAddResponse(
        @Schema(description = "Torrent id")
        String torrentId,

        @Schema(description = "Name of added torrent")
        String name,

        @Schema(description = "Status of added torrent")
        String status,

        @Schema(description = "Files present in torrent (video + subtitle)")
        List<FileItem> files
) implements Serializable {

    public record FileItem(
            String id,
            String name,
            String sizeLabel,
            String fileType   // "VIDEO" or "SUBTITLE"
    ) {}
}
