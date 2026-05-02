# Graph Report - .  (2026-05-02)

## Corpus Check
- 70 files · ~28,030 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 575 nodes · 870 edges · 30 communities detected
- Extraction: 73% EXTRACTED · 27% INFERRED · 0% AMBIGUOUS · INFERRED: 238 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Go Torrent Core Engine|Go Torrent Core Engine]]
- [[_COMMUNITY_Spring REST API Layer|Spring REST API Layer]]
- [[_COMMUNITY_Integration Tests & Prowlarr Client|Integration Tests & Prowlarr Client]]
- [[_COMMUNITY_HLS Transcoding Pipeline|HLS Transcoding Pipeline]]
- [[_COMMUNITY_Architecture Docs & Config|Architecture Docs & Config]]
- [[_COMMUNITY_System Architecture Overview|System Architecture Overview]]
- [[_COMMUNITY_App Entry Points & Piece Priority|App Entry Points & Piece Priority]]
- [[_COMMUNITY_Prowlarr Download Manager|Prowlarr Download Manager]]
- [[_COMMUNITY_gRPC Client Bridge|gRPC Client Bridge]]
- [[_COMMUNITY_React UI Event Handlers|React UI Event Handlers]]
- [[_COMMUNITY_FFmpeg Binary Manager|FFmpeg Binary Manager]]
- [[_COMMUNITY_Torrent Lifecycle Operations|Torrent Lifecycle Operations]]
- [[_COMMUNITY_Subtitle Handler|Subtitle Handler]]
- [[_COMMUNITY_FFmpeg Utilities|FFmpeg Utilities]]
- [[_COMMUNITY_HLS Server Interface|HLS Server Interface]]
- [[_COMMUNITY_gRPC Channel Config|gRPC Channel Config]]
- [[_COMMUNITY_Search REST Controller|Search REST Controller]]
- [[_COMMUNITY_Status UI Panel|Status UI Panel]]
- [[_COMMUNITY_Spring Boot Entry Point|Spring Boot Entry Point]]
- [[_COMMUNITY_CORS Configuration|CORS Configuration]]
- [[_COMMUNITY_Error Handling|Error Handling]]
- [[_COMMUNITY_Byte Formatting Util|Byte Formatting Util]]
- [[_COMMUNITY_Spring Boot Tests|Spring Boot Tests]]
- [[_COMMUNITY_HLS Stop Flow|HLS Stop Flow]]
- [[_COMMUNITY_gRPC Test Client|gRPC Test Client]]
- [[_COMMUNITY_Torrent File ID Model|Torrent File ID Model]]
- [[_COMMUNITY_Prowlarr Manager|Prowlarr Manager]]
- [[_COMMUNITY_Torrent Info Model|Torrent Info Model]]
- [[_COMMUNITY_Get Sessions Endpoint|Get Sessions Endpoint]]
- [[_COMMUNITY_BitTorrent DHT Discovery|BitTorrent DHT Discovery]]

## God Nodes (most connected - your core abstractions)
1. `MagnetPlay CLAUDE.md - Project Guidance` - 23 edges
2. `StartServer()` - 20 edges
3. `StartServer()` - 18 edges
4. `TorrentController` - 17 edges
5. `TorrentGrpcClient` - 16 edges
6. `TorrentService` - 16 edges
7. `TorrentController` - 15 edges
8. `prowlarr.Manager.Start` - 13 edges
9. `ProwlarrClient` - 13 edges
10. `Spring Boot Backend (mp-spring)` - 13 edges

## Surprising Connections (you probably didn't know these)
- `StartServer()` --calls--> `NewRepository()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\repository.go
- `Go sidecar owns all torrent state (design decision)` --rationale_for--> `TorrentGrpcClient`  [EXTRACTED]
  CLAUDE.md → backend\mp-spring\src\main\java\org\devMandali\magnetPlay\client\TorrentGrpcClient.java
- `Spring Boot as thin translation layer (design decision)` --rationale_for--> `TorrentService`  [EXTRACTED]
  CLAUDE.md → backend\mp-spring\src\main\java\org\devMandali\magnetPlay\service\TorrentService.java
- `ProwlarrClient.resolveApiKey()` --conceptually_related_to--> `Go sidecar owns all torrent state (design decision)`  [INFERRED]
  backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/ProwlarrClient.java → CLAUDE.md
- `Piece prioritization strategy (first 5 + last 5 + range)` --rationale_for--> `TorrentService.streamFile()`  [INFERRED]
  CLAUDE.md → backend/mp-spring/src/main/java/org/devMandali/magnetPlay/service/TorrentService.java

## Hyperedges (group relationships)
- **gRPC Streaming Pipeline: Spring→gRPC→Go→BitTorrent** — spring_grpc_client, proto_torrent_proto, go_service, concept_grpc_streaming [EXTRACTED 0.95]
- **React Frontend Component Tree** — frontend_main_tsx, frontend_app_tsx, frontend_video_player, frontend_subtitle_panel, frontend_netflix_overlay [INFERRED 0.85]
- **Video Seeking Flow: HTTP Range → CORS → Piece Prioritization** — concept_http_range_requests, spring_cors_config, concept_piece_prioritization [INFERRED 0.80]

## Communities

### Community 0 - "Go Torrent Core Engine"
Cohesion: 0.06
Nodes (38): StartServer(), hls.RemuxHandler.StopAll, toFileInfoList(), prowlarr.NewManager, NewRepository(), IsMKV(), Prioritize(), runFFprobe() (+30 more)

### Community 1 - "Spring REST API Layer"
Cohesion: 0.05
Nodes (10): TorrentController, TorrentHLSControllerTest, TorrentService, SessionManager, SessionManagerTest, withClosed(), torrent.TorrentService.DeleteTorrent, validateMagnetLink() (+2 more)

### Community 2 - "Integration Tests & Prowlarr Client"
Cohesion: 0.06
Nodes (11): Spring Boot as thin translation layer (design decision), ProwlarrClient, TestIsMKV(), TestStartRemux_ReturnsNotFound_UnknownTorrent(), NewTorrentService(), newTestRepository(), TestGetTorrentStats_UnknownHash(), TestListTorrents_EmptyRepo() (+3 more)

### Community 3 - "HLS Transcoding Pipeline"
Cohesion: 0.09
Nodes (34): activeJob, FileOpener, FilePrioritizer, HLSServer, RemuxHandler, NewHLSServer(), hls server tests, TestRawFileHandler_CORS() (+26 more)

### Community 4 - "Architecture Docs & Config"
Cohesion: 0.07
Nodes (42): Spring Boot application.yml, MagnetPlay CLAUDE.md - Project Guidance, MagnetPlay CLAUDE.md, anacrolix/torrent BitTorrent Engine Library, File ID Format: {infoHash}:{fileIndex}, Go Sidecar Owns All Torrent State, God Node: App.tsx (React graph center, 11+ edges), God Node: repository.go (mutex risk, 11+ edges) (+34 more)

### Community 5 - "System Architecture Overview"
Cohesion: 0.06
Nodes (39): Frontend (React SPA), Go Sidecar (go-server), Peer Network (DHT + Trackers), Spring Boot Backend (mp-spring), BitTorrent Protocol, CORS Headers Required for Browser Video Seeking, DHT (Distributed Hash Table), FileChunk (gRPC Message, 256KB) (+31 more)

### Community 6 - "App Entry Points & Piece Priority"
Cohesion: 0.07
Nodes (26): App (React root), Piece prioritization strategy (first 5 + last 5 + range), Config struct, Config.Default, ProwlarrConfig struct, main (entrypoint), SearchPanel, StatusPanel (+18 more)

### Community 7 - "Prowlarr Download Manager"
Cohesion: 0.11
Nodes (23): NewClient(), BinaryName(), downloadToFile(), EnsureBinary(), extractZip(), fetchLatestRelease(), FindBinary(), platformSuffix() (+15 more)

### Community 8 - "gRPC Client Bridge"
Cohesion: 0.06
Nodes (5): Go sidecar owns all torrent state (design decision), TorrentGrpcClient, ProwlarrClient.resolveApiKey(), ProwlarrClient.search(), TorrentGrpcClient

### Community 9 - "React UI Event Handlers"
Cohesion: 0.08
Nodes (17): handleFetchFiles(), handleFullReset(), handlePlay(), handleReset(), HLSStartResponse, encodeFileId(), handleFullReset(), handlePlay() (+9 more)

### Community 10 - "FFmpeg Binary Manager"
Cohesion: 0.19
Nodes (18): downloadFFmpeg(), downloadToFile(), EnsureFFmpeg(), EnsureFFprobe(), extractZip(), fetchLatestRelease(), FFmpegBinaryName(), FFprobeBinaryName() (+10 more)

### Community 11 - "Torrent Lifecycle Operations"
Cohesion: 0.2
Nodes (13): deleteTorrent(), formatBytes(), formatSpeed(), pause(), resume(), withAction(), deleteTorrent(), fetchData() (+5 more)

### Community 12 - "Subtitle Handler"
Cohesion: 0.23
Nodes (9): NewSubtitleHandler(), TestSubtitleEmbedded_BadPath(), TestSubtitleEmbedded_BadStreamIndex(), TestSubtitleEmbedded_CORSOnOptions(), TestSubtitleEmbedded_PathTraversalRejected(), TestSubtitleFile_BadPath(), TestSubtitleFile_ContentTypeHeader(), TestSubtitleFile_CORSOnOptions() (+1 more)

### Community 13 - "FFmpeg Utilities"
Cohesion: 0.24
Nodes (10): ffmpeg.downloadFFmpeg, ffmpeg.EnsureFFmpeg, ffmpeg.EnsureFFprobe, ffmpeg.extractZip, ffmpeg.fetchLatestRelease, ffmpeg.FFmpegBinaryName, ffmpeg.FFprobeBinaryName, ffmpeg.ResolveFFprobePath (+2 more)

### Community 14 - "HLS Server Interface"
Cohesion: 0.33
Nodes (7): hls.FileOpener (type), hls.FilePrioritizer (type), hls.HLSServer, hls.HLSServer.serveRawFile, hls.RemuxHandler, torrent.Repository, torrent.TorrentService

### Community 16 - "gRPC Channel Config"
Cohesion: 0.5
Nodes (1): GrpcConfig

### Community 17 - "Search REST Controller"
Cohesion: 0.5
Nodes (1): SearchController

### Community 18 - "Status UI Panel"
Cohesion: 0.67
Nodes (2): fmtBytes(), fmtSpeed()

### Community 20 - "Spring Boot Entry Point"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplication

### Community 21 - "CORS Configuration"
Cohesion: 0.67
Nodes (1): CorsConfig

### Community 22 - "Error Handling"
Cohesion: 0.67
Nodes (1): GlobalExceptionHandler

### Community 23 - "Byte Formatting Util"
Cohesion: 0.67
Nodes (1): ByteUtil

### Community 24 - "Spring Boot Tests"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplicationTests

### Community 26 - "HLS Stop Flow"
Cohesion: 0.67
Nodes (2): TorrentController.stopHLS() DELETE /v1/torrent/hls/stop, TorrentService.stopHLS()

### Community 28 - "gRPC Test Client"
Cohesion: 1.0
Nodes (1): GrpcClient

### Community 32 - "Torrent File ID Model"
Cohesion: 1.0
Nodes (2): File ID format {infoHash}:{fileIndex}, TorrentFile interface

### Community 44 - "Prowlarr Manager"
Cohesion: 1.0
Nodes (1): prowlarr.Manager

### Community 45 - "Torrent Info Model"
Cohesion: 1.0
Nodes (1): torrent.TorrentInfo

### Community 49 - "Get Sessions Endpoint"
Cohesion: 1.0
Nodes (1): TorrentController.getSessions() GET /v1/torrent/sessions

### Community 50 - "BitTorrent DHT Discovery"
Cohesion: 1.0
Nodes (1): BitTorrent DHT + Tracker Peer Discovery

## Knowledge Gaps
- **68 isolated node(s):** `githubAsset`, `githubRelease`, `indexerDef`, `sample`, `GrpcClient` (+63 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `gRPC Channel Config`** (4 nodes): `GrpcConfig.java`, `GrpcConfig`, `.customChannelConfigurer()`, `.grpcScheduler()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Search REST Controller`** (4 nodes): `SearchController.java`, `SearchController`, `.search()`, `.SearchController()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Status UI Panel`** (4 nodes): `StatusPanel.tsx`, `fmtBytes()`, `fmtSpeed()`, `poll()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring Boot Entry Point`** (3 nodes): `MpSpringBackendApplication.java`, `MpSpringBackendApplication`, `.main()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `CORS Configuration`** (3 nodes): `CorsConfig.java`, `CorsConfig`, `.corsWebFilter()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Error Handling`** (3 nodes): `GlobalExceptionHandler.java`, `GlobalExceptionHandler`, `.handleValidationException()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Byte Formatting Util`** (3 nodes): `ByteUtil.java`, `ByteUtil`, `.formatSize()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring Boot Tests`** (3 nodes): `MpSpringBackendApplicationTests.java`, `MpSpringBackendApplicationTests`, `.contextLoads()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `HLS Stop Flow`** (3 nodes): `TorrentController.stopHLS() DELETE /v1/torrent/hls/stop`, `TorrentGrpcClient.stopHLS()`, `TorrentService.stopHLS()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `gRPC Test Client`** (2 nodes): `GrpcClient.java`, `GrpcClient`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent File ID Model`** (2 nodes): `File ID format {infoHash}:{fileIndex}`, `TorrentFile interface`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Prowlarr Manager`** (1 nodes): `prowlarr.Manager`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Info Model`** (1 nodes): `torrent.TorrentInfo`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Sessions Endpoint`** (1 nodes): `TorrentController.getSessions() GET /v1/torrent/sessions`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `BitTorrent DHT Discovery`** (1 nodes): `BitTorrent DHT + Tracker Peer Discovery`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `StartServer()` connect `Go Torrent Core Engine` to `Spring REST API Layer`, `HLS Transcoding Pipeline`, `App Entry Points & Piece Priority`, `Prowlarr Download Manager`, `Subtitle Handler`, `FFmpeg Utilities`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **Why does `TorrentController` connect `Integration Tests & Prowlarr Client` to `Spring REST API Layer`, `App Entry Points & Piece Priority`?**
  _High betweenness centrality (0.101) - this node is a cross-community bridge._
- **Why does `TorrentService` connect `Integration Tests & Prowlarr Client` to `gRPC Client Bridge`, `Spring REST API Layer`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `StartServer()` (e.g. with `NewClient()` and `FileOpener`) actually correct?**
  _`StartServer()` has 8 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `StartServer()` (e.g. with `main (entrypoint)` and `NewManager()`) actually correct?**
  _`StartServer()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **What connects `githubAsset`, `githubRelease`, `indexerDef` to the rest of the system?**
  _68 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Go Torrent Core Engine` be split into smaller, more focused modules?**
  _Cohesion score 0.06 - nodes in this community are weakly interconnected._