# Graph Report - .  (2026-04-27)

## Corpus Check
- 65 files · ~25,640 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 453 nodes · 656 edges · 52 communities detected
- Extraction: 74% EXTRACTED · 26% INFERRED · 0% AMBIGUOUS · INFERRED: 171 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Spring REST Service Layer|Spring REST Service Layer]]
- [[_COMMUNITY_Go gRPC Server Core|Go gRPC Server Core]]
- [[_COMMUNITY_App Root and Config|App Root and Config]]
- [[_COMMUNITY_HLS Transcoding Pipeline|HLS Transcoding Pipeline]]
- [[_COMMUNITY_Torrent State Repository|Torrent State Repository]]
- [[_COMMUNITY_gRPC Wiring and Prowlarr Start|gRPC Wiring and Prowlarr Start]]
- [[_COMMUNITY_FFmpeg Auto-Download|FFmpeg Auto-Download]]
- [[_COMMUNITY_Frontend App Handlers|Frontend App Handlers]]
- [[_COMMUNITY_gRPC Client and Search|gRPC Client and Search]]
- [[_COMMUNITY_HLS File Server|HLS File Server]]
- [[_COMMUNITY_Prowlarr HTTP Client|Prowlarr HTTP Client]]
- [[_COMMUNITY_Prowlarr Binary Downloader|Prowlarr Binary Downloader]]
- [[_COMMUNITY_FFmpeg Binary Functions|FFmpeg Binary Functions]]
- [[_COMMUNITY_Torrents Page UI|Torrents Page UI]]
- [[_COMMUNITY_HLS HTTP Handler|HLS HTTP Handler]]
- [[_COMMUNITY_Search Panel UI|Search Panel UI]]
- [[_COMMUNITY_Status Panel UI|Status Panel UI]]
- [[_COMMUNITY_Stop HLS Lifecycle|Stop HLS Lifecycle]]
- [[_COMMUNITY_Test Repository Helper|Test Repository Helper]]
- [[_COMMUNITY_Streaming Session Model|Streaming Session Model]]
- [[_COMMUNITY_List Torrents RPC|List Torrents RPC]]
- [[_COMMUNITY_File ID Contract|File ID Contract]]
- [[_COMMUNITY_HLS Start Response|HLS Start Response]]
- [[_COMMUNITY_Frontend Entry Point|Frontend Entry Point]]
- [[_COMMUNITY_Frontend Type Definitions|Frontend Type Definitions]]
- [[_COMMUNITY_Prowlarr Manager|Prowlarr Manager]]
- [[_COMMUNITY_Torrent Info Struct|Torrent Info Struct]]
- [[_COMMUNITY_Pause Torrent RPC|Pause Torrent RPC]]
- [[_COMMUNITY_Resume Torrent RPC|Resume Torrent RPC]]
- [[_COMMUNITY_Delete Torrent RPC|Delete Torrent RPC]]
- [[_COMMUNITY_Get Sessions Endpoint|Get Sessions Endpoint]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]

## God Nodes (most connected - your core abstractions)
1. `StartServer()` - 18 edges
2. `TorrentController` - 17 edges
3. `TorrentGrpcClient` - 16 edges
4. `TorrentService` - 16 edges
5. `grpc_server.StartServer` - 16 edges
6. `ProwlarrClient` - 13 edges
7. `Spring Boot Backend (mp-spring)` - 13 edges
8. `Manager` - 11 edges
9. `TorrentService` - 11 edges
10. `Go Sidecar (go-server)` - 11 edges

## Surprising Connections (you probably didn't know these)
- `Go sidecar owns all torrent state (design decision)` --rationale_for--> `TorrentGrpcClient`  [EXTRACTED]
  CLAUDE.md → backend\mp-spring\src\main\java\org\devMandali\magnetPlay\client\TorrentGrpcClient.java
- `Spring Boot as thin translation layer (design decision)` --rationale_for--> `TorrentService`  [EXTRACTED]
  CLAUDE.md → backend\mp-spring\src\main\java\org\devMandali\magnetPlay\service\TorrentService.java
- `ProwlarrClient.resolveApiKey()` --conceptually_related_to--> `Go sidecar owns all torrent state (design decision)`  [INFERRED]
  backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/ProwlarrClient.java → CLAUDE.md
- `Piece prioritization strategy (first 5 + last 5 + range)` --rationale_for--> `TorrentService.streamFile()`  [INFERRED]
  CLAUDE.md → backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java
- `File ID format {infoHash}:{fileIndex}` --rationale_for--> `TorrentFile interface`  [INFERRED]
  CLAUDE.md → frontend/src/types/index.ts

## Hyperedges (group relationships)
- **gRPC Server Startup Wiring Flow** — grpc_server_startserver, prowlarr_manager_newmanager, ffmpeg_binary_ensureffmpeg, ffmpeg_binary_ensureffprobe, torrent_repository_newrepository, hls_transcoder_newremuxhandler, hls_server_newhlsserver, torrent_service_newtorrentservice [EXTRACTED 0.95]
- **MKV Streaming Pipeline (rawfile â†’ FFmpeg remux â†’ fMP4)** — hls_server_serverawfile, hls_transcoder_remuxhandler, hls_transcoder_buildremuxargs, torrent_service_startremux, torrent_service_getorprobe, torrent_service_runffprobe [INFERRED 0.90]
- **Piece Prioritization for Streaming** — torrent_service_streamfile, torrent_service_prioritize, hls_server_serverawfile, hls_server_fileprioritizer, grpc_server_startserver [INFERRED 0.85]
- **FFmpeg Binary Auto-Download and Resolution** — ffmpeg_binary_ensureffmpeg, ffmpeg_binary_ensureffprobe, ffmpeg_binary_downloadffmpeg, ffmpeg_binary_fetchlatestrelease, ffmpeg_binary_extractzip, ffmpeg_binary_ffmpegbinaryname, ffmpeg_binary_ffprobebinaryname [EXTRACTED 0.95]
- **Prowlarr Lifecycle: Download, Start, Seed Indexers** — prowlarr_manager_manager, prowlarr_manager_start, prowlarr_manager_stop, prowlarr_downloader_ensurebinary, prowlarr_manager_seeddefaultindexers, prowlarr_manager_fetchindexerschemas [EXTRACTED 0.95]
- **Torrent gRPC Service Methods** — torrent_service_addtorrent, torrent_service_getfileinfo, torrent_service_streamfile, torrent_service_gettorrentstats, torrent_service_listtorrents, torrent_service_pausetorrent, torrent_service_resumetorrent, torrent_service_deletetorrent, torrent_service_startremux [EXTRACTED 1.00]
- **gRPC Translation Pipeline: Controller â†’ Service â†’ GrpcClient** — torrentcontroller_torrentcontroller, torrentservice_torrentservice, torrentgrpcclient_torrentgrpcclient [EXTRACTED 0.98]
- **MKV Remux Streaming Flow: App â†’ REST â†’ Service â†’ gRPC â†’ VideoPlayer** — app_handleplay, torrentcontroller_startremux, torrentservice_startremux, torrentgrpcclient_startremux, videoplayer_videoplayer [EXTRACTED 0.95]
- **Live Stats Polling: StatusPanel + VideoPlayer StatsButton â†’ /v1/torrent/stats â†’ TorrentService** — statuspanel_statuspanel, videosetup_registerstatsbutton, torrentservice_gettorrentstats [EXTRACTED 0.92]

## Communities

### Community 0 - "Spring REST Service Layer"
Cohesion: 0.06
Nodes (41): grpc_server.StartServer, hls.FileOpener (type), hls.FilePrioritizer (type), hls.HLSServer, hls.NewHLSServer, hls.HLSServer.serveRawFile, hls server tests, hls.BuildRemuxArgs (+33 more)

### Community 1 - "Go gRPC Server Core"
Cohesion: 0.06
Nodes (7): Spring Boot as thin translation layer (design decision), SessionManager, SessionManagerTest, withClosed(), TorrentController, TorrentHLSControllerTest, TorrentService

### Community 2 - "App Root and Config"
Cohesion: 0.06
Nodes (43): MagnetPlay CLAUDE.md, Frontend (React SPA), Go Sidecar (go-server), Peer Network (DHT + Trackers), Spring Boot Backend (mp-spring), BitTorrent Protocol, DHT (Distributed Hash Table), FileChunk (gRPC Message, 256KB) (+35 more)

### Community 3 - "HLS Transcoding Pipeline"
Cohesion: 0.07
Nodes (26): App (React root), Piece prioritization strategy (first 5 + last 5 + range), Config, Default(), ProwlarrConfig, main(), SearchPanel, StatusPanel (+18 more)

### Community 4 - "Torrent State Repository"
Cohesion: 0.12
Nodes (22): activeJob, FileOpener, FilePrioritizer, HLSServer, RemuxHandler, NewHLSServer(), TestRawFileHandler_CORS(), TestRawFileHandler_NotFound_NoOpener() (+14 more)

### Community 5 - "gRPC Wiring and Prowlarr Start"
Cohesion: 0.13
Nodes (11): NewClient(), cloneMap(), extractProwlarrError(), NewManager(), prowlarr.BinaryName, prowlarr.EnsureBinary, prowlarr.FindBinary, indexerDef (+3 more)

### Community 6 - "FFmpeg Auto-Download"
Cohesion: 0.14
Nodes (10): NewSpeedTracker(), TestSpeedTracker_OldSamplesDropped(), TestSpeedTracker_SingleSample(), TestSpeedTracker_ZeroAtStart(), sample, torrent.TorrentService.DeleteTorrent, torrent.TorrentService.PauseTorrent, torrent.TorrentService.ResumeTorrent (+2 more)

### Community 7 - "Frontend App Handlers"
Cohesion: 0.1
Nodes (4): Go sidecar owns all torrent state (design decision), ProwlarrClient.resolveApiKey(), ProwlarrClient.search(), TorrentGrpcClient

### Community 8 - "gRPC Client and Search"
Cohesion: 0.12
Nodes (13): handleFetchFiles(), handleFullReset(), handlePlay(), handleReset(), HLSStartResponse, TorrentController.addTorrent() POST /v1/torrent/add, TorrentController.startHLS() POST /v1/torrent/hls/start, TorrentController.startRemux() POST /v1/torrent/remux/start (+5 more)

### Community 9 - "HLS File Server"
Cohesion: 0.19
Nodes (18): downloadFFmpeg(), downloadToFile(), EnsureFFmpeg(), EnsureFFprobe(), extractZip(), fetchLatestRelease(), FFmpegBinaryName(), FFprobeBinaryName() (+10 more)

### Community 10 - "Prowlarr HTTP Client"
Cohesion: 0.22
Nodes (1): ProwlarrClient

### Community 11 - "Prowlarr Binary Downloader"
Cohesion: 0.25
Nodes (6): TestStartRemux_ReturnsNotFound_UnknownTorrent(), NewTorrentService(), newTestRepository(), TestGetTorrentStats_UnknownHash(), TestListTorrents_EmptyRepo(), TorrentStatsControllerTest

### Community 12 - "FFmpeg Binary Functions"
Cohesion: 0.33
Nodes (9): BinaryName(), downloadToFile(), EnsureBinary(), extractZip(), fetchLatestRelease(), FindBinary(), platformSuffix(), githubAsset (+1 more)

### Community 13 - "Torrents Page UI"
Cohesion: 0.24
Nodes (10): ffmpeg.downloadFFmpeg, ffmpeg.EnsureFFmpeg, ffmpeg.EnsureFFprobe, ffmpeg.extractZip, ffmpeg.fetchLatestRelease, ffmpeg.FFmpegBinaryName, ffmpeg.FFprobeBinaryName, ffmpeg.ResolveFFprobePath (+2 more)

### Community 14 - "HLS HTTP Handler"
Cohesion: 0.36
Nodes (7): deleteTorrent(), fetchData(), formatBytes(), formatSpeed(), pause(), resume(), withAction()

### Community 15 - "Search Panel UI"
Cohesion: 0.4
Nodes (2): validateMagnetLink(), TorrentUtil

### Community 16 - "Status Panel UI"
Cohesion: 0.4
Nodes (0): 

### Community 17 - "Stop HLS Lifecycle"
Cohesion: 0.5
Nodes (1): GrpcConfig

### Community 18 - "Test Repository Helper"
Cohesion: 0.5
Nodes (1): SearchController

### Community 19 - "Streaming Session Model"
Cohesion: 0.67
Nodes (2): fmtBytes(), fmtSpeed()

### Community 20 - "List Torrents RPC"
Cohesion: 0.5
Nodes (0): 

### Community 21 - "File ID Contract"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplication

### Community 22 - "HLS Start Response"
Cohesion: 0.67
Nodes (1): CorsConfig

### Community 23 - "Frontend Entry Point"
Cohesion: 0.67
Nodes (1): GlobalExceptionHandler

### Community 24 - "Frontend Type Definitions"
Cohesion: 0.67
Nodes (1): ByteUtil

### Community 25 - "Prowlarr Manager"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplicationTests

### Community 26 - "Torrent Info Struct"
Cohesion: 0.67
Nodes (0): 

### Community 27 - "Pause Torrent RPC"
Cohesion: 0.67
Nodes (2): TorrentController.stopHLS() DELETE /v1/torrent/hls/stop, TorrentService.stopHLS()

### Community 28 - "Resume Torrent RPC"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "Delete Torrent RPC"
Cohesion: 1.0
Nodes (1): GrpcClient

### Community 30 - "Get Sessions Endpoint"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "Community 31"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (0): 

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (2): File ID format {infoHash}:{fileIndex}, TorrentFile interface

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (0): 

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (0): 

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (0): 

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (0): 

### Community 38 - "Community 38"
Cohesion: 1.0
Nodes (0): 

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (0): 

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (0): 

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (0): 

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (0): 

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (0): 

### Community 44 - "Community 44"
Cohesion: 1.0
Nodes (1): Session Tracking (Per Range Request)

### Community 45 - "Community 45"
Cohesion: 1.0
Nodes (0): 

### Community 46 - "Community 46"
Cohesion: 1.0
Nodes (1): prowlarr.Manager

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (1): torrent.TorrentInfo

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (0): 

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (0): 

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (0): 

### Community 51 - "Community 51"
Cohesion: 1.0
Nodes (1): TorrentController.getSessions() GET /v1/torrent/sessions

## Knowledge Gaps
- **57 isolated node(s):** `githubAsset`, `githubRelease`, `indexerDef`, `sample`, `GrpcClient` (+52 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Resume Torrent RPC`** (2 nodes): `repository_test.go`, `TestTorrentInfo_PauseResume()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Delete Torrent RPC`** (2 nodes): `GrpcClient.java`, `GrpcClient`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Sessions Endpoint`** (2 nodes): `SubtitlePanel.tsx`, `handleFilePick()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (2 nodes): `repository_test_helper.go`, `NewTestRepository()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (2 nodes): `TorrentGrpcClient.listTorrents()`, `TorrentService.listTorrents()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (2 nodes): `File ID format {infoHash}:{fileIndex}`, `TorrentFile interface`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (1 nodes): `SearchResult.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (1 nodes): `SearchResultsResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (1 nodes): `SessionResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (1 nodes): `TorrentAddResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (1 nodes): `TorrentListResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (1 nodes): `TorrentStatsResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (1 nodes): `index.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (1 nodes): `Session Tracking (Per Range Request)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (1 nodes): `HLSStartResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (1 nodes): `prowlarr.Manager`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (1 nodes): `torrent.TorrentInfo`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (1 nodes): `TorrentGrpcClient.pauseTorrent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (1 nodes): `TorrentGrpcClient.resumeTorrent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (1 nodes): `TorrentGrpcClient.deleteTorrent()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (1 nodes): `TorrentController.getSessions() GET /v1/torrent/sessions`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `TorrentController` connect `Go gRPC Server Core` to `HLS Transcoding Pipeline`, `Prowlarr HTTP Client`, `Prowlarr Binary Downloader`?**
  _High betweenness centrality (0.128) - this node is a cross-community bridge._
- **Why does `StartServer()` connect `gRPC Wiring and Prowlarr Start` to `Spring REST Service Layer`, `Go gRPC Server Core`, `HLS Transcoding Pipeline`, `Torrent State Repository`, `HLS File Server`, `Prowlarr Binary Downloader`?**
  _High betweenness centrality (0.115) - this node is a cross-community bridge._
- **Why does `grpc_server.StartServer` connect `Spring REST Service Layer` to `Go gRPC Server Core`, `Torrents Page UI`, `HLS Transcoding Pipeline`, `gRPC Wiring and Prowlarr Start`?**
  _High betweenness centrality (0.095) - this node is a cross-community bridge._
- **Are the 17 inferred relationships involving `StartServer()` (e.g. with `main()` and `NewManager()`) actually correct?**
  _`StartServer()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **What connects `githubAsset`, `githubRelease`, `indexerDef` to the rest of the system?**
  _57 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Spring REST Service Layer` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._
- **Should `Go gRPC Server Core` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._