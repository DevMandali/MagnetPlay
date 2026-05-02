# Graph Report - .  (2026-05-03)

## Corpus Check
- Corpus is ~31,020 words - fits in a single context window. You may not need a graph.

## Summary
- 716 nodes · 1069 edges · 41 communities detected
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 255 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Go Torrent gRPC Service|Go Torrent gRPC Service]]
- [[_COMMUNITY_Spring Session & HLS Layer|Spring Session & HLS Layer]]
- [[_COMMUNITY_App Entry & Config Bootstrap|App Entry & Config Bootstrap]]
- [[_COMMUNITY_Spring REST Controllers|Spring REST Controllers]]
- [[_COMMUNITY_Frontend App Wizard State|Frontend App Wizard State]]
- [[_COMMUNITY_HLS Transcoder & Server|HLS Transcoder & Server]]
- [[_COMMUNITY_Project Config & Architecture|Project Config & Architecture]]
- [[_COMMUNITY_System Architecture Concepts|System Architecture Concepts]]
- [[_COMMUNITY_Spring gRPC Client|Spring gRPC Client]]
- [[_COMMUNITY_Frontend App Event Handlers|Frontend App Event Handlers]]
- [[_COMMUNITY_Prowlarr Process Manager|Prowlarr Process Manager]]
- [[_COMMUNITY_Torrent Mapper & Speed Tracker|Torrent Mapper & Speed Tracker]]
- [[_COMMUNITY_Binary Downloader & Repository|Binary Downloader & Repository]]
- [[_COMMUNITY_FFmpeg Binary Manager|FFmpeg Binary Manager]]
- [[_COMMUNITY_Torrent List UI|Torrent List UI]]
- [[_COMMUNITY_HLS Subtitle Handler|HLS Subtitle Handler]]
- [[_COMMUNITY_Electron Service Manager|Electron Service Manager]]
- [[_COMMUNITY_FFmpeg Binary Functions|FFmpeg Binary Functions]]
- [[_COMMUNITY_Electron Main Process|Electron Main Process]]
- [[_COMMUNITY_Status Panel UI|Status Panel UI]]
- [[_COMMUNITY_HLS Server Core|HLS Server Core]]
- [[_COMMUNITY_HLS Remux & Subtitle Serve|HLS Remux & Subtitle Serve]]
- [[_COMMUNITY_Spring gRPC Config|Spring gRPC Config]]
- [[_COMMUNITY_Spring Search Controller|Spring Search Controller]]
- [[_COMMUNITY_Spring App Entry|Spring App Entry]]
- [[_COMMUNITY_Spring CORS Config|Spring CORS Config]]
- [[_COMMUNITY_Spring Exception Handler|Spring Exception Handler]]
- [[_COMMUNITY_Spring Byte Util|Spring Byte Util]]
- [[_COMMUNITY_Spring App Tests|Spring App Tests]]
- [[_COMMUNITY_Stop HLS Cross-Layer|Stop HLS Cross-Layer]]
- [[_COMMUNITY_Spring gRPC Test Client|Spring gRPC Test Client]]
- [[_COMMUNITY_File ID Types|File ID Types]]
- [[_COMMUNITY_VideoJS Skip Buttons|VideoJS Skip Buttons]]
- [[_COMMUNITY_VideoJS Audio Track|VideoJS Audio Track]]
- [[_COMMUNITY_Prowlarr Manager Instance|Prowlarr Manager Instance]]
- [[_COMMUNITY_Torrent Info Type|Torrent Info Type]]
- [[_COMMUNITY_Get Sessions Controller|Get Sessions Controller]]
- [[_COMMUNITY_BitTorrent DHT Concept|BitTorrent DHT Concept]]
- [[_COMMUNITY_Torrent Stats Service|Torrent Stats Service]]
- [[_COMMUNITY_Electron Window API|Electron Window API]]
- [[_COMMUNITY_VideoJS VHS Config|VideoJS VHS Config]]

## God Nodes (most connected - your core abstractions)
1. `StartServer()` - 32 edges
2. `MagnetPlay CLAUDE.md - Project Guidance` - 23 edges
3. `Repository` - 19 edges
4. `StartServer()` - 18 edges
5. `TorrentController` - 17 edges
6. `Manager` - 16 edges
7. `TorrentGrpcClient` - 16 edges
8. `TorrentService` - 16 edges
9. `TorrentController` - 15 edges
10. `ProwlarrClient` - 13 edges

## Surprising Connections (you probably didn't know these)
- `StartServer()` --calls--> `NewManager()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\prowlarr\manager.go
- `StartServer()` --calls--> `NewRepository()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\repository.go
- `Go sidecar owns all torrent state (design decision)` --rationale_for--> `TorrentGrpcClient`  [EXTRACTED]
  CLAUDE.md → backend\mp-spring\src\main\java\org\devMandali\magnetPlay\client\TorrentGrpcClient.java
- `Spring Boot as thin translation layer (design decision)` --rationale_for--> `TorrentService`  [EXTRACTED]
  CLAUDE.md → backend\mp-spring\src\main\java\org\devMandali\magnetPlay\service\TorrentService.java
- `StartServer()` --calls--> `NewManager()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\prowlarr\manager.go

## Hyperedges (group relationships)
- **FFmpeg-backed Media Pipeline (Remux + Subtitle)** — hls_remux_handler, hls_subtitle_handler, torrent_service_startremux, grpc_server_startserver [EXTRACTED 0.95]
- **Electron Service Orchestration (Go + Spring + IPC)** — electron_main, electron_servicemanager, electron_preload [EXTRACTED 0.98]
- **Torrent Streaming Flow (Repo + Service + Prioritize)** — torrent_repository, torrent_service_streamfile, torrent_service_prioritize [EXTRACTED 0.97]
- **MKV Seek Pipeline: seekOffsetRef + mkvSeekTo + SeekBar Patch + Time Display Patch** — videoplayer_mkvSeekTo, videoplayer_seekOffsetRef, videoplayer_seekBarPatch, videoplayer_timeDisplayPatch [EXTRACTED 0.95]
- **Electron Auto-Update Flow: updater â†’ IPC events â†’ UpdateBanner** — updater_initUpdater, updater_updateAvailableEvent, updater_updateDownloadedEvent, updatebanner_UpdateBanner, app_updateState [EXTRACTED 0.95]
- **Subtitle Loading Pipeline: fetchSubtitleBlob + encodeFileId + subtitleSync** — app_fetchSubtitleBlob, app_encodeFileId, videoplayer_subtitleSync, utils_srtToVtt [INFERRED 0.85]

## Communities

### Community 0 - "Go Torrent gRPC Service"
Cohesion: 0.07
Nodes (41): NewClient(), FileOpener Closure (HLS Raw File Access), FilePrioritizer Closure (Byte-Range Piece Priority), StartServer(), GetRemuxBaseURL, RemuxHandler (Fragmented MP4 via FFmpeg), RemuxHandler.StopAll (Shutdown Cleanup), SubtitleHandler (WebVTT via FFmpeg) (+33 more)

### Community 1 - "Spring Session & HLS Layer"
Cohesion: 0.06
Nodes (8): TorrentController, TorrentHLSControllerTest, TorrentService, SessionManager, SessionManagerTest, withClosed(), validateMagnetLink(), TorrentUtil

### Community 2 - "App Entry & Config Bootstrap"
Cohesion: 0.05
Nodes (33): App (React root), Piece prioritization strategy (first 5 + last 5 + range), Config, Default(), FromEnv(), Config.Default, FromEnv Config Loader, ProwlarrConfig (+25 more)

### Community 3 - "Spring REST Controllers"
Cohesion: 0.06
Nodes (6): Spring Boot as thin translation layer (design decision), ProwlarrClient, TorrentController, TorrentHLSControllerTest, TorrentService, TorrentStatsControllerTest

### Community 4 - "Frontend App Wizard State"
Cohesion: 0.06
Nodes (47): App Root Component, encodeFileId (base64url), fetchSubtitleBlob Function, handleFetchFiles Function, handlePlay Function, handleSubtitleAdd Function, MOOV_BADGE Display Map, Electron Update State (updateStatus, updateVersion) (+39 more)

### Community 5 - "HLS Transcoder & Server"
Cohesion: 0.09
Nodes (34): activeJob, FileOpener, FilePrioritizer, HLSServer, RemuxHandler, NewHLSServer(), hls server tests, TestRawFileHandler_CORS() (+26 more)

### Community 6 - "Project Config & Architecture"
Cohesion: 0.07
Nodes (42): Spring Boot application.yml, MagnetPlay CLAUDE.md - Project Guidance, MagnetPlay CLAUDE.md, anacrolix/torrent BitTorrent Engine Library, File ID Format: {infoHash}:{fileIndex}, Go Sidecar Owns All Torrent State, God Node: App.tsx (React graph center, 11+ edges), God Node: repository.go (mutex risk, 11+ edges) (+34 more)

### Community 7 - "System Architecture Concepts"
Cohesion: 0.06
Nodes (39): Frontend (React SPA), Go Sidecar (go-server), Peer Network (DHT + Trackers), Spring Boot Backend (mp-spring), BitTorrent Protocol, CORS Headers Required for Browser Video Seeking, DHT (Distributed Hash Table), FileChunk (gRPC Message, 256KB) (+31 more)

### Community 8 - "Spring gRPC Client"
Cohesion: 0.06
Nodes (5): Go sidecar owns all torrent state (design decision), TorrentGrpcClient, ProwlarrClient.resolveApiKey(), ProwlarrClient.search(), TorrentGrpcClient

### Community 9 - "Frontend App Event Handlers"
Cohesion: 0.08
Nodes (17): handleFetchFiles(), handleFullReset(), handlePlay(), handleReset(), HLSStartResponse, encodeFileId(), handleFullReset(), handlePlay() (+9 more)

### Community 10 - "Prowlarr Process Manager"
Cohesion: 0.13
Nodes (13): cloneMap(), extractProwlarrError(), NewManager(), prowlarr.BinaryName, prowlarr.EnsureBinary, prowlarr.FindBinary, indexerDef, Manager (+5 more)

### Community 11 - "Torrent Mapper & Speed Tracker"
Cohesion: 0.15
Nodes (10): toFileInfoList(), NewSpeedTracker(), TestSpeedTracker_OldSamplesDropped(), TestSpeedTracker_SingleSample(), TestSpeedTracker_ZeroAtStart(), toFileInfoList(), sample, SpeedTracker (+2 more)

### Community 12 - "Binary Downloader & Repository"
Cohesion: 0.11
Nodes (20): BinaryName(), downloadToFile(), EnsureBinary(), extractZip(), fetchLatestRelease(), FindBinary(), platformSuffix(), Electron Main Process (+12 more)

### Community 13 - "FFmpeg Binary Manager"
Cohesion: 0.19
Nodes (18): downloadFFmpeg(), downloadToFile(), EnsureFFmpeg(), EnsureFFprobe(), extractZip(), fetchLatestRelease(), FFmpegBinaryName(), FFprobeBinaryName() (+10 more)

### Community 14 - "Torrent List UI"
Cohesion: 0.16
Nodes (13): deleteTorrent(), formatBytes(), formatSpeed(), pause(), resume(), withAction(), deleteTorrent(), fetchData() (+5 more)

### Community 15 - "HLS Subtitle Handler"
Cohesion: 0.21
Nodes (9): NewSubtitleHandler(), TestSubtitleEmbedded_BadPath(), TestSubtitleEmbedded_BadStreamIndex(), TestSubtitleEmbedded_CORSOnOptions(), TestSubtitleEmbedded_PathTraversalRejected(), TestSubtitleFile_BadPath(), TestSubtitleFile_ContentTypeHeader(), TestSubtitleFile_CORSOnOptions() (+1 more)

### Community 16 - "Electron Service Manager"
Cohesion: 0.24
Nodes (6): getResourcePath(), killProc(), safeChmod(), ServiceManager, waitForHttp(), waitForPort()

### Community 17 - "FFmpeg Binary Functions"
Cohesion: 0.24
Nodes (10): ffmpeg.downloadFFmpeg, ffmpeg.EnsureFFmpeg, ffmpeg.EnsureFFprobe, ffmpeg.extractZip, ffmpeg.fetchLatestRelease, ffmpeg.FFmpegBinaryName, ffmpeg.FFprobeBinaryName, ffmpeg.ResolveFFprobePath (+2 more)

### Community 19 - "Electron Main Process"
Cohesion: 0.43
Nodes (7): closeSplash(), createSplash(), createTray(), createWindow(), registerIpcHandlers(), updateTrayMenu(), Electron Preload Script (contextBridge)

### Community 20 - "Status Panel UI"
Cohesion: 0.38
Nodes (4): fmtBytes(), fmtSpeed(), fmtBytes(), fmtSpeed()

### Community 22 - "HLS Server Core"
Cohesion: 0.33
Nodes (7): hls.FileOpener (type), hls.FilePrioritizer (type), hls.HLSServer, hls.HLSServer.serveRawFile, hls.RemuxHandler, torrent.Repository, torrent.TorrentService

### Community 23 - "HLS Remux & Subtitle Serve"
Cohesion: 0.4
Nodes (5): BuildRemuxArgs (FFmpeg Args Builder), RemuxHandler.ServeHTTP, runFFmpegToWebVTT (FFmpeg Subtitle Pipe), ServeEmbedded (Extract MKV Subtitle Stream), ServeFile (Convert Subtitle File to WebVTT)

### Community 24 - "Spring gRPC Config"
Cohesion: 0.5
Nodes (1): GrpcConfig

### Community 25 - "Spring Search Controller"
Cohesion: 0.5
Nodes (1): SearchController

### Community 26 - "Spring App Entry"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplication

### Community 27 - "Spring CORS Config"
Cohesion: 0.67
Nodes (1): CorsConfig

### Community 28 - "Spring Exception Handler"
Cohesion: 0.67
Nodes (1): GlobalExceptionHandler

### Community 29 - "Spring Byte Util"
Cohesion: 0.67
Nodes (1): ByteUtil

### Community 30 - "Spring App Tests"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplicationTests

### Community 32 - "Stop HLS Cross-Layer"
Cohesion: 0.67
Nodes (2): TorrentController.stopHLS() DELETE /v1/torrent/hls/stop, TorrentService.stopHLS()

### Community 34 - "Spring gRPC Test Client"
Cohesion: 1.0
Nodes (1): GrpcClient

### Community 38 - "File ID Types"
Cohesion: 1.0
Nodes (2): File ID format {infoHash}:{fileIndex}, TorrentFile interface

### Community 40 - "VideoJS Skip Buttons"
Cohesion: 1.0
Nodes (2): SkipButton Video.js Component, registerSkipButtons Function

### Community 41 - "VideoJS Audio Track"
Cohesion: 1.0
Nodes (2): AudioTrackMenuButton Video.js Component, registerAudioTrackButton Function

### Community 53 - "Prowlarr Manager Instance"
Cohesion: 1.0
Nodes (1): prowlarr.Manager

### Community 54 - "Torrent Info Type"
Cohesion: 1.0
Nodes (1): torrent.TorrentInfo

### Community 58 - "Get Sessions Controller"
Cohesion: 1.0
Nodes (1): TorrentController.getSessions() GET /v1/torrent/sessions

### Community 59 - "BitTorrent DHT Concept"
Cohesion: 1.0
Nodes (1): BitTorrent DHT + Tracker Peer Discovery

### Community 63 - "Torrent Stats Service"
Cohesion: 1.0
Nodes (1): TorrentService.GetTorrentStats

### Community 64 - "Electron Window API"
Cohesion: 1.0
Nodes (1): Window.electronAPI Global Declaration

### Community 65 - "VideoJS VHS Config"
Cohesion: 1.0
Nodes (1): configureVhs Function

## Knowledge Gaps
- **92 isolated node(s):** `githubAsset`, `githubRelease`, `indexerDef`, `sample`, `GrpcClient` (+87 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Spring gRPC Config`** (4 nodes): `GrpcConfig.java`, `GrpcConfig`, `.customChannelConfigurer()`, `.grpcScheduler()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring Search Controller`** (4 nodes): `SearchController.java`, `SearchController`, `.search()`, `.SearchController()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring App Entry`** (3 nodes): `MpSpringBackendApplication.java`, `MpSpringBackendApplication`, `.main()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring CORS Config`** (3 nodes): `CorsConfig.java`, `CorsConfig`, `.corsWebFilter()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring Exception Handler`** (3 nodes): `GlobalExceptionHandler.java`, `GlobalExceptionHandler`, `.handleValidationException()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring Byte Util`** (3 nodes): `ByteUtil.java`, `ByteUtil`, `.formatSize()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring App Tests`** (3 nodes): `MpSpringBackendApplicationTests.java`, `MpSpringBackendApplicationTests`, `.contextLoads()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Stop HLS Cross-Layer`** (3 nodes): `TorrentController.stopHLS() DELETE /v1/torrent/hls/stop`, `TorrentGrpcClient.stopHLS()`, `TorrentService.stopHLS()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Spring gRPC Test Client`** (2 nodes): `GrpcClient.java`, `GrpcClient`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `File ID Types`** (2 nodes): `File ID format {infoHash}:{fileIndex}`, `TorrentFile interface`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `VideoJS Skip Buttons`** (2 nodes): `SkipButton Video.js Component`, `registerSkipButtons Function`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `VideoJS Audio Track`** (2 nodes): `AudioTrackMenuButton Video.js Component`, `registerAudioTrackButton Function`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Prowlarr Manager Instance`** (1 nodes): `prowlarr.Manager`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Info Type`** (1 nodes): `torrent.TorrentInfo`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Sessions Controller`** (1 nodes): `TorrentController.getSessions() GET /v1/torrent/sessions`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `BitTorrent DHT Concept`** (1 nodes): `BitTorrent DHT + Tracker Peer Discovery`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Stats Service`** (1 nodes): `TorrentService.GetTorrentStats`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Electron Window API`** (1 nodes): `Window.electronAPI Global Declaration`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `VideoJS VHS Config`** (1 nodes): `configureVhs Function`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `StartServer()` connect `Go Torrent gRPC Service` to `App Entry & Config Bootstrap`, `HLS Transcoder & Server`, `Prowlarr Process Manager`, `Binary Downloader & Repository`, `Torrent List UI`, `HLS Subtitle Handler`, `FFmpeg Binary Functions`?**
  _High betweenness centrality (0.115) - this node is a cross-community bridge._
- **Why does `TorrentController` connect `Spring REST Controllers` to `Spring Session & HLS Layer`, `App Entry & Config Bootstrap`, `Torrent List UI`?**
  _High betweenness centrality (0.076) - this node is a cross-community bridge._
- **Why does `TorrentService` connect `Spring REST Controllers` to `Spring gRPC Client`, `Spring Session & HLS Layer`?**
  _High betweenness centrality (0.060) - this node is a cross-community bridge._
- **Are the 12 inferred relationships involving `StartServer()` (e.g. with `NewClient()` and `FileOpener`) actually correct?**
  _`StartServer()` has 12 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `StartServer()` (e.g. with `main (entrypoint)` and `NewManager()`) actually correct?**
  _`StartServer()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **What connects `githubAsset`, `githubRelease`, `indexerDef` to the rest of the system?**
  _92 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Go Torrent gRPC Service` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._