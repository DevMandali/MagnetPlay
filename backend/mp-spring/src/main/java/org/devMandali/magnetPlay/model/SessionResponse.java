package org.devMandali.magnetPlay.model;

import java.time.Instant;
import java.util.List;

public record SessionResponse(List<SessionItem> sessions) {
    public record SessionItem(
        String  sessionId,
        String  infoHash,
        String  fileId,
        String  clientIp,
        Instant startTime,
        Instant endTime,
        long    startByte,
        long    bytesServed,
        String  status,
        String  closeReason
    ) {}
}
