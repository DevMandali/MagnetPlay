package org.devMandali.magnetPlay.model;

import io.swagger.v3.oas.annotations.media.Schema;

import java.io.Serializable;
import java.util.Map;

public record TorrentAddResponse(
        @Schema(description = "Torrent id to track added Torrent entry")
        String torrentId,

        @Schema(description = "Name of added torrent Magnet URL")
        String name,

        @Schema(description = "Status of added torrent in Magnetplay")
        String status,

        @Schema(
                type = "object",
                description = "List of video files present under added torrent entry. Present through map of file id as key and file name with its size as value {\"b7efcb48193a4a9e11497d00930d754c0bf1c65b:0\": \"Fallout.2024.S01E01.720p.AMZN.WEBRip.x264-GalaxyTV.mkv => 437.17 M\"}",
                example = "{\"TORRENT_HASH_INFO:INT_ID\": \"FILE_NAME => FILE_SIZE\"}",
                additionalPropertiesSchema = String.class
        )
        Map<String, String> files
) implements Serializable {}
