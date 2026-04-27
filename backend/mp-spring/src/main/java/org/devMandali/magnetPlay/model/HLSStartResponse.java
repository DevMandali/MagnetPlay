package org.devMandali.magnetPlay.model;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.util.List;

public record HLSStartResponse(
    @NotBlank
    String manifestUrl,
    boolean success,
    double durationSec,
    @NotNull
    List<AudioTrackDto> audioTracks
) {
    public record AudioTrackDto(int index, String language, String codec, String title) {}
}
