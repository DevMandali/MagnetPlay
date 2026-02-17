package org.devMandali.magnetPlay.controller;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.media.Content;
import io.swagger.v3.oas.annotations.media.Schema;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.responses.ApiResponses;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.devMandali.magnetPlay.model.TorrentAddResponse;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.util.TorrentUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/v1/torrent")
public class TorrentController {

    private final Logger logger = LoggerFactory.getLogger(TorrentController.class);
    private final TorrentService service;

    public TorrentController(TorrentService service) {
        this.service = service;
    }

    @Operation(summary = "Add Torrent URL")
    @Tag(name = "Add-Torrent")
    @ApiResponses(value = {
            @ApiResponse(responseCode = "201", description = "Added the torrent",
                    content = { @Content(mediaType = "application/json",
                            schema = @Schema(implementation = TorrentAddResponse.class)) }),
            @ApiResponse(responseCode = "400", description = "Invalid URL supplied",
                    content = @Content),
    })
    @PostMapping(consumes = {"application/json"})
    @io.swagger.v3.oas.annotations.parameters.RequestBody(required = true)
    public ResponseEntity<TorrentAddResponse> addTorrent(@Valid @RequestBody TorrentAddRequest request){
        logger.info("Got new torrent request");

        if(logger.isDebugEnabled())  {
            logger.debug("Torrent Magnet Link: {}", request.magnet());

            String infoHash = TorrentUtil.extractInfoHash(request.magnet());
            logger.debug("Extracted infoHash: {}", infoHash);
        }
        return ResponseEntity.status(HttpStatus.CREATED).body(service.addTorrentToSession(request));
    }
}
