package org.devMandali.magnetPlay.model;

import jakarta.validation.constraints.NotNull;
import org.devMandali.magnetPlay.util.TorrentUtil;
import org.springframework.util.StringUtils;

public record TorrentAddRequest(
        @NotNull(message = "Torrent URL/Magnet Link is mandatory") String magnet
) {
    public TorrentAddRequest {
        if (magnet == null || magnet.isBlank()) {
            throw new IllegalArgumentException("Torrent URL/Magnet Link is mandatory");
        }
        String errorMsg = validateMagnetLink(magnet);
        if(StringUtils.hasText(errorMsg)) {
            throw new IllegalArgumentException(errorMsg);
        }
    }

    public String validateMagnetLink(String link) {
        // 1. Checking format
        if(!link.startsWith(TorrentUtil.TORRENT_LINK_START)) {
            return "invalid magnet link format";
        }

        // 2. Extract info hash
        String infoHash = TorrentUtil.extractInfoHash(link);
        if(StringUtils.hasText(infoHash) && infoHash.length() != 40 && infoHash.length() != 32) {
            return "invalid info hash length";
        }
        return null;
    }
}
