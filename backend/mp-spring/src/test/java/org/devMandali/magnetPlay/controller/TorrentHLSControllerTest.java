package org.devMandali.magnetPlay.controller;

import org.devMandali.magnetPlay.model.HLSStartResponse;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.session.SessionManager;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.WebFluxTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.reactive.server.WebTestClient;
import reactor.core.publisher.Mono;
import static org.assertj.core.api.Assertions.assertThat;

import java.util.List;

import static org.mockito.Mockito.when;

@WebFluxTest(TorrentController.class)
class TorrentHLSControllerTest {

    @Autowired
    private WebTestClient webTestClient;

    @MockBean
    private TorrentService torrentService;

    @MockBean
    private SessionManager sessionManager;

    @Test
    void startHLS_returns200_withManifestUrl() {
        HLSStartResponse mockResp = new HLSStartResponse(
                "http://localhost:8091/hls/abc123/abc123:0/playlist.m3u8",
                true, 7243.0, List.of());

        when(torrentService.startHLS("abc123", "abc123:0", 0.0))
                .thenReturn(Mono.just(mockResp));

        webTestClient.post()
                .uri("/v1/torrent/hls/abc123/start?fileId=abc123:0&t=0")
                .exchange()
                .expectStatus().isOk()
                .expectBody(HLSStartResponse.class)
                .value(resp -> {
                    assertThat(resp.manifestUrl()).contains("playlist.m3u8");
                    assertThat(resp.success()).isTrue();
                    assertThat(resp.durationSec()).isEqualTo(7243.0);
                });
    }

    @Test
    void startHLS_returns503_whenSuccessFalse() {
        HLSStartResponse failResp = new HLSStartResponse("", false, 0.0, List.of());

        when(torrentService.startHLS("abc123", "abc123:0", 0.0))
                .thenReturn(Mono.just(failResp));

        webTestClient.post()
                .uri("/v1/torrent/hls/abc123/start?fileId=abc123:0&t=0")
                .exchange()
                .expectStatus().is5xxServerError();
    }

    @Test
    void stopHLS_returns200() {
        when(torrentService.stopHLS("abc123", "abc123:0"))
                .thenReturn(Mono.just("stopped"));

        webTestClient.delete()
                .uri("/v1/torrent/hls/abc123/stop?fileId=abc123:0")
                .exchange()
                .expectStatus().isOk();
    }
}
