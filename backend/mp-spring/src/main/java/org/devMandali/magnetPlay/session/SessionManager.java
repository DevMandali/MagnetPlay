package org.devMandali.magnetPlay.session;

import org.springframework.stereotype.Component;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;

@Component
public class SessionManager {
    private final ConcurrentHashMap<String, StreamingSession> active = new ConcurrentHashMap<>();
    private final Deque<StreamingSession> closed = new ArrayDeque<>();
    private static final int MAX_CLOSED = 100;

    public StreamingSession openSession(String infoHash, String fileId,
                                        String clientIp, String userAgent, long startByte) {
        var session = new StreamingSession(
            UUID.randomUUID().toString(), infoHash, fileId, clientIp, userAgent,
            Instant.now(), null, startByte, new AtomicLong(0),
            StreamingSession.SessionStatus.ACTIVE, null
        );
        active.put(session.sessionId(), session);
        return session;
    }

    public void closeSession(String sessionId, StreamingSession.CloseReason reason) {
        var session = active.remove(sessionId);
        if (session == null) return;
        var status = switch (reason) {
            case COMPLETED -> StreamingSession.SessionStatus.COMPLETED;
            case CANCELLED -> StreamingSession.SessionStatus.CANCELLED;
            default        -> StreamingSession.SessionStatus.ERROR;
        };
        var closed_ = session.withClosed(Instant.now(), status, reason);
        synchronized (closed) {
            closed.addLast(closed_);
            while (closed.size() > MAX_CLOSED) closed.removeFirst();
        }
    }

    public List<StreamingSession> listActive() {
        return new ArrayList<>(active.values());
    }

    public List<StreamingSession> listAll() {
        var result = new ArrayList<StreamingSession>();
        result.addAll(active.values());
        synchronized (closed) { result.addAll(closed); }
        return result;
    }
}
