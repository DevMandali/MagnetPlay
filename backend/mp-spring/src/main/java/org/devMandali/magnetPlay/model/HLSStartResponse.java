package org.devMandali.magnetPlay.model;

import java.util.List;

public record HLSStartResponse(
    String manifestUrl,
    boolean success,
    double durationSec,
    List<AudioTrackDto> audioTracks
) {
    public record AudioTrackDto(int index, String language, String codec, String title) {}
}
