package org.devMandali.magnetPlay.session;

import org.junit.jupiter.api.Test;
import static org.assertj.core.api.Assertions.assertThat;

class SessionManagerTest {

    @Test
    void openSession_appearsInActive() {
        var mgr = new SessionManager();
        var s = mgr.openSession("hash1", "hash1:0", "127.0.0.1", "TestAgent", 0L);
        assertThat(mgr.listActive()).hasSize(1);
        assertThat(mgr.listActive().get(0).sessionId()).isEqualTo(s.sessionId());
    }

    @Test
    void closeSession_movesToClosed() {
        var mgr = new SessionManager();
        var s = mgr.openSession("hash1", "hash1:0", "127.0.0.1", "TestAgent", 0L);
        mgr.closeSession(s.sessionId(), StreamingSession.CloseReason.COMPLETED);
        assertThat(mgr.listActive()).isEmpty();
        assertThat(mgr.listAll()).hasSize(1);
        assertThat(mgr.listAll().get(0).status()).isEqualTo(StreamingSession.SessionStatus.COMPLETED);
    }

    @Test
    void closedSessionsCappedAt100() {
        var mgr = new SessionManager();
        for (int i = 0; i < 110; i++) {
            var s = mgr.openSession("h", "h:0", "ip", "ua", 0);
            mgr.closeSession(s.sessionId(), StreamingSession.CloseReason.COMPLETED);
        }
        assertThat(mgr.listAll()).hasSizeLessThanOrEqualTo(100);
    }
}
