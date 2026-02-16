package org.devMandali.magnetPlay.util;

public class TorrentUtil {
    public static final String TORRENT_LINK_START = "magnet:?xt=urn:btih:";

    public static String extractInfoHash(String link) {
        if (link == null || !link.startsWith(TORRENT_LINK_START)) {
            throw new IllegalArgumentException("Invalid magnet link format");
        }
        int start = TORRENT_LINK_START.length();
        int end = link.indexOf("&", start);
        return end == -1 ? link.substring(start) : link.substring(start, end);
    }
}
