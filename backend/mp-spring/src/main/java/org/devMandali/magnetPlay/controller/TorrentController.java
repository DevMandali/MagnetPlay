package org.devMandali.magnetPlay.controller;
import jakarta.validation.Valid;
import org.devMandali.magnetPlay.model.TorrentAddRequest;
import org.devMandali.magnetPlay.service.TorrentService;
import org.devMandali.magnetPlay.util.TorrentUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/torrent")
public class TorrentController {

    private final Logger logger = LoggerFactory.getLogger(TorrentController.class);
    private final TorrentService service;

    public TorrentController(TorrentService service) {
        this.service = service;
    }

    @PostMapping(consumes = {"application/json"})
    public ResponseEntity<?> addTorrent(@Valid @RequestBody TorrentAddRequest request){
        logger.info("Got new torrent request");

        if(logger.isDebugEnabled())  {
            logger.debug("Torrent Magnet Link: {}", request.magnet());

            String infoHash = TorrentUtil.extractInfoHash(request.magnet());
            logger.debug("Extracted infoHash: {}", infoHash);
        }
        return ResponseEntity.ok(service.addTorrentToSession(request));
    }
}
