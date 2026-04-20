package org.devMandali.magnetPlay.controller;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.WebFluxTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.reactive.server.WebTestClient;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.model.TorrentStatsResponse;
import org.devMandali.magnetPlay.session.SessionManager;
import reactor.core.publisher.Mono;
import static org.mockito.Mockito.when;

@WebFluxTest(TorrentController.class)
class TorrentStatsControllerTest {

    @Autowired WebTestClient client;
    @MockBean  TorrentService torrentService;
    @MockBean  SessionManager sessionManager;

    @Test
    void stats_returns200() {
        var stats = new TorrentStatsResponse("abc:0", 1000L, 500L, 50.0, 1024.0);
        when(torrentService.getTorrentStats("abc", "abc:0")).thenReturn(Mono.just(stats));

        client.get()
              .uri("/v1/torrent/stats/abc?fileId=abc:0")
              .exchange()
              .expectStatus().isOk()
              .expectBody()
              .jsonPath("$.completionPct").isEqualTo(50.0);
    }
}
