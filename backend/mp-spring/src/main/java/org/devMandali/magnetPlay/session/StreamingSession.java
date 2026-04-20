package org.devMandali.magnetPlay.session;

import java.time.Instant;
import java.util.concurrent.atomic.AtomicLong;

public record StreamingSession(
    String        sessionId,
    String        infoHash,
    String        fileId,
    String        clientIp,
    String        userAgent,
    Instant       startTime,
    Instant       endTime,
    long          startByte,
    AtomicLong    bytesServed,
    SessionStatus status,
    CloseReason   closeReason
) {
    public enum SessionStatus { ACTIVE, COMPLETED, CANCELLED, ERROR }
    public enum CloseReason   { COMPLETED, CANCELLED, ERROR, UNKNOWN }

    public StreamingSession withClosed(Instant end, SessionStatus s, CloseReason r) {
        return new StreamingSession(sessionId, infoHash, fileId, clientIp, userAgent,
            startTime, end, startByte, bytesServed, s, r);
    }
}
