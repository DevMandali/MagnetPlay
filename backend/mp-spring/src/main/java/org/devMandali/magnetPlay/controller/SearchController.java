package org.devMandali.magnetPlay.controller;

import org.devMandali.magnetPlay.client.ProwlarrClient;
import org.devMandali.magnetPlay.model.SearchResultsResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import reactor.core.publisher.Mono;

@RestController
@RequestMapping("/v1/search")
public class SearchController {

    private static final Logger log = LoggerFactory.getLogger(SearchController.class);
    private final ProwlarrClient prowlarrClient;

    public SearchController(ProwlarrClient prowlarrClient) {
        this.prowlarrClient = prowlarrClient;
    }

    @GetMapping
    public Mono<ResponseEntity<SearchResultsResponse>> search(@RequestParam String q) {
        if (q == null || q.isBlank()) {
            return Mono.just(ResponseEntity.badRequest().<SearchResultsResponse>build());
        }
        return prowlarrClient.search(q.trim())
            .map(results -> ResponseEntity.ok(new SearchResultsResponse(results, results.size())))
            .doOnError(e -> log.error("Search failed: {}", e.getMessage()))
            .onErrorReturn(ResponseEntity.status(503).<SearchResultsResponse>build());
    }
}
