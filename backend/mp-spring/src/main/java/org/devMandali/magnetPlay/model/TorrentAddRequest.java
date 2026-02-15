package org.devMandali.magnetPlay.model;

import jakarta.validation.constraints.NotNull;
import org.springframework.util.StringUtils;

public record TorrentAddRequest(
        @NotNull(message = "Torrent URL/Magnet Link is mandatory") String magnet
) {
    public TorrentAddRequest {
        String errorMsg = isValidMagnetLink(magnet);
        if(StringUtils.hasText(errorMsg)) {
            throw new IllegalArgumentException(errorMsg);
        }
    }

    public String isValidMagnetLink(String link) {
        // 1. Checking format
        if(!link.startsWith("magnet:?xt=urn:btih:")) {
            return "invalid magnet link format";
        }

        // 2. Extract info hash
//        String infoHash = extractInfoHash(link);
//        if(StringUtils.hasText(infoHash) && infoHash.length() != 40 && infoHash.length() != 32) {
//            return "invalid info hash length";
//        }
        return null;
    }

    public String extractInfoHash(String link) {
        return link.substring(link.indexOf("magnet:?xt=urn:btih:"), link.indexOf("&"));
    }
}
