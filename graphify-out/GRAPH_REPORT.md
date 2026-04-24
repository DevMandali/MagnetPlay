# Graph Report - .  (2026-04-24)

## Corpus Check
- Corpus is ~19,591 words - fits in a single context window. You may not need a graph.

## Summary
- 274 nodes · 361 edges · 41 communities detected
- Extraction: 72% EXTRACTED · 28% INFERRED · 0% AMBIGUOUS · INFERRED: 100 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Architecture Overview|Architecture Overview]]
- [[_COMMUNITY_Go Torrent Core|Go Torrent Core]]
- [[_COMMUNITY_Torrent Service + Tests|Torrent Service + Tests]]
- [[_COMMUNITY_gRPC Server + Prowlarr Manager|gRPC Server + Prowlarr Manager]]
- [[_COMMUNITY_Prowlarr HTTP Client|Prowlarr HTTP Client]]
- [[_COMMUNITY_Session Management|Session Management]]
- [[_COMMUNITY_Spring gRPC Client|Spring gRPC Client]]
- [[_COMMUNITY_Speed Tracker|Speed Tracker]]
- [[_COMMUNITY_Binary Downloader|Binary Downloader]]
- [[_COMMUNITY_Torrents Page UI|Torrents Page UI]]
- [[_COMMUNITY_App State Handlers|App State Handlers]]
- [[_COMMUNITY_Video.js Setup|Video.js Setup]]
- [[_COMMUNITY_Go Config + Entry|Go Config + Entry]]
- [[_COMMUNITY_Project Documentation|Project Documentation]]
- [[_COMMUNITY_Torrent Input Validation|Torrent Input Validation]]
- [[_COMMUNITY_Search Panel UI|Search Panel UI]]
- [[_COMMUNITY_gRPC Channel Config|gRPC Channel Config]]
- [[_COMMUNITY_Search REST Controller|Search REST Controller]]
- [[_COMMUNITY_Status Panel UI|Status Panel UI]]
- [[_COMMUNITY_Subtitle Utilities|Subtitle Utilities]]
- [[_COMMUNITY_Spring Boot Entry|Spring Boot Entry]]
- [[_COMMUNITY_CORS Config|CORS Config]]
- [[_COMMUNITY_Global Error Handler|Global Error Handler]]
- [[_COMMUNITY_Byte Formatter|Byte Formatter]]
- [[_COMMUNITY_Spring Context Test|Spring Context Test]]
- [[_COMMUNITY_Netflix Skip Overlay|Netflix Skip Overlay]]
- [[_COMMUNITY_Repository Test|Repository Test]]
- [[_COMMUNITY_gRPC Test Client|gRPC Test Client]]
- [[_COMMUNITY_Subtitle Panel|Subtitle Panel]]
- [[_COMMUNITY_Video Player|Video Player]]
- [[_COMMUNITY_Search Result DTO|Search Result DTO]]
- [[_COMMUNITY_Search Results Response|Search Results Response]]
- [[_COMMUNITY_Session Response DTO|Session Response DTO]]
- [[_COMMUNITY_Torrent Add Response|Torrent Add Response]]
- [[_COMMUNITY_Torrent List Response|Torrent List Response]]
- [[_COMMUNITY_Torrent Stats Response|Torrent Stats Response]]
- [[_COMMUNITY_Vite Config|Vite Config]]
- [[_COMMUNITY_Frontend Entry|Frontend Entry]]
- [[_COMMUNITY_Vite Env Types|Vite Env Types]]
- [[_COMMUNITY_Frontend Types|Frontend Types]]
- [[_COMMUNITY_Session Tracking Concept|Session Tracking Concept]]

## God Nodes (most connected - your core abstractions)
1. `ProwlarrClient` - 13 edges
2. `Spring Boot Backend (mp-spring)` - 13 edges
3. `TorrentController` - 12 edges
4. `Manager` - 11 edges
5. `Go Sidecar (go-server)` - 11 edges
6. `Repository` - 10 edges
7. `TorrentGrpcClient` - 10 edges
8. `TorrentService` - 10 edges
9. `StartServer()` - 9 edges
10. `TorrentService` - 9 edges

## Surprising Connections (you probably didn't know these)
- `StartServer()` --calls--> `NewRepository()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\repository.go
- `main()` --calls--> `StartServer()`  [INFERRED]
  backend\go-server\main.go → backend\go-server\internal\grpc\server.go
- `StartServer()` --calls--> `NewManager()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\prowlarr\manager.go
- `StartServer()` --calls--> `NewClient()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\client.go
- `StartServer()` --calls--> `NewTorrentService()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\service.go

## Hyperedges (group relationships)
- **Three-Layer MagnetPlay Architecture** — component_frontend, component_spring_boot, component_go_sidecar, component_peer_network [EXTRACTED 1.00]
- **Shared gRPC Proto Contract Between Go and Java** — file_torrent_proto, component_spring_boot, component_go_sidecar [EXTRACTED 1.00]
- **HTTP Range to BitTorrent Streaming Pipeline** — concept_http_range, concept_stream_request, concept_file_chunk, concept_grpc_streaming, concept_bittorrent [EXTRACTED 0.90]

## Communities

### Community 0 - "Architecture Overview"
Cohesion: 0.07
Nodes (37): Frontend (React SPA), Go Sidecar (go-server), Peer Network (DHT + Trackers), Spring Boot Backend (mp-spring), BitTorrent Protocol, DHT (Distributed Hash Table), FileChunk (gRPC Message, 256KB), File ID Format ({infoHash}:{fileIndex}) (+29 more)

### Community 1 - "Go Torrent Core"
Cohesion: 0.13
Nodes (6): toFileInfoList(), NewRepository(), prioritize(), Repository, TorrentInfo, TorrentService

### Community 2 - "Torrent Service + Tests"
Cohesion: 0.1
Nodes (7): NewTorrentService(), newTestRepository(), TestGetTorrentStats_UnknownHash(), TestListTorrents_EmptyRepo(), TorrentController, TorrentService, TorrentStatsControllerTest

### Community 3 - "gRPC Server + Prowlarr Manager"
Cohesion: 0.18
Nodes (7): NewClient(), cloneMap(), extractProwlarrError(), NewManager(), indexerDef, Manager, StartServer()

### Community 4 - "Prowlarr HTTP Client"
Cohesion: 0.24
Nodes (1): ProwlarrClient

### Community 5 - "Session Management"
Cohesion: 0.24
Nodes (3): SessionManager, SessionManagerTest, withClosed()

### Community 6 - "Spring gRPC Client"
Cohesion: 0.17
Nodes (1): TorrentGrpcClient

### Community 7 - "Speed Tracker"
Cohesion: 0.35
Nodes (6): NewSpeedTracker(), TestSpeedTracker_OldSamplesDropped(), TestSpeedTracker_SingleSample(), TestSpeedTracker_ZeroAtStart(), sample, SpeedTracker

### Community 8 - "Binary Downloader"
Cohesion: 0.33
Nodes (9): BinaryName(), downloadToFile(), EnsureBinary(), extractZip(), fetchLatestRelease(), FindBinary(), platformSuffix(), githubAsset (+1 more)

### Community 9 - "Torrents Page UI"
Cohesion: 0.36
Nodes (7): deleteTorrent(), fetchData(), formatBytes(), formatSpeed(), pause(), resume(), withAction()

### Community 10 - "App State Handlers"
Cohesion: 0.29
Nodes (2): handleFullReset(), handleReset()

### Community 11 - "Video.js Setup"
Cohesion: 0.33
Nodes (2): fmtBytes(), fmtSpeed()

### Community 12 - "Go Config + Entry"
Cohesion: 0.33
Nodes (4): Config, Default(), ProwlarrConfig, main()

### Community 13 - "Project Documentation"
Cohesion: 0.33
Nodes (6): MagnetPlay CLAUDE.md, MagnetPlay System, Peer-to-Peer Video Streaming, Piece Prioritization Strategy, Rationale: Piece Prioritization for Streaming, MagnetPlay README

### Community 14 - "Torrent Input Validation"
Cohesion: 0.4
Nodes (2): validateMagnetLink(), TorrentUtil

### Community 15 - "Search Panel UI"
Cohesion: 0.4
Nodes (0): 

### Community 16 - "gRPC Channel Config"
Cohesion: 0.5
Nodes (1): GrpcConfig

### Community 17 - "Search REST Controller"
Cohesion: 0.5
Nodes (1): SearchController

### Community 18 - "Status Panel UI"
Cohesion: 0.67
Nodes (2): fmtBytes(), fmtSpeed()

### Community 19 - "Subtitle Utilities"
Cohesion: 0.5
Nodes (0): 

### Community 20 - "Spring Boot Entry"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplication

### Community 21 - "CORS Config"
Cohesion: 0.67
Nodes (1): CorsConfig

### Community 22 - "Global Error Handler"
Cohesion: 0.67
Nodes (1): GlobalExceptionHandler

### Community 23 - "Byte Formatter"
Cohesion: 0.67
Nodes (1): ByteUtil

### Community 24 - "Spring Context Test"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplicationTests

### Community 25 - "Netflix Skip Overlay"
Cohesion: 0.67
Nodes (0): 

### Community 26 - "Repository Test"
Cohesion: 1.0
Nodes (0): 

### Community 27 - "gRPC Test Client"
Cohesion: 1.0
Nodes (1): GrpcClient

### Community 28 - "Subtitle Panel"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "Video Player"
Cohesion: 1.0
Nodes (0): 

### Community 30 - "Search Result DTO"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "Search Results Response"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "Session Response DTO"
Cohesion: 1.0
Nodes (0): 

### Community 33 - "Torrent Add Response"
Cohesion: 1.0
Nodes (0): 

### Community 34 - "Torrent List Response"
Cohesion: 1.0
Nodes (0): 

### Community 35 - "Torrent Stats Response"
Cohesion: 1.0
Nodes (0): 

### Community 36 - "Vite Config"
Cohesion: 1.0
Nodes (0): 

### Community 37 - "Frontend Entry"
Cohesion: 1.0
Nodes (0): 

### Community 38 - "Vite Env Types"
Cohesion: 1.0
Nodes (0): 

### Community 39 - "Frontend Types"
Cohesion: 1.0
Nodes (0): 

### Community 40 - "Session Tracking Concept"
Cohesion: 1.0
Nodes (1): Session Tracking (Per Range Request)

## Knowledge Gaps
- **29 isolated node(s):** `ProwlarrConfig`, `Config`, `githubAsset`, `githubRelease`, `indexerDef` (+24 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Repository Test`** (2 nodes): `repository_test.go`, `TestTorrentInfo_PauseResume()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `gRPC Test Client`** (2 nodes): `GrpcClient.java`, `GrpcClient`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Subtitle Panel`** (2 nodes): `SubtitlePanel.tsx`, `handleFilePick()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Video Player`** (2 nodes): `VideoPlayer.tsx`, `VideoPlayer()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Search Result DTO`** (1 nodes): `SearchResult.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Search Results Response`** (1 nodes): `SearchResultsResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Session Response DTO`** (1 nodes): `SessionResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Add Response`** (1 nodes): `TorrentAddResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent List Response`** (1 nodes): `TorrentListResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Stats Response`** (1 nodes): `TorrentStatsResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Config`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend Entry`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Env Types`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend Types`** (1 nodes): `index.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Session Tracking Concept`** (1 nodes): `Session Tracking (Per Range Request)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `TorrentController` connect `Torrent Service + Tests` to `Go Torrent Core`, `Prowlarr HTTP Client`, `Session Management`?**
  _High betweenness centrality (0.080) - this node is a cross-community bridge._
- **Why does `StartServer()` connect `gRPC Server + Prowlarr Manager` to `Go Torrent Core`, `Torrent Service + Tests`, `Go Config + Entry`?**
  _High betweenness centrality (0.079) - this node is a cross-community bridge._
- **Why does `EnsureBinary()` connect `Binary Downloader` to `Go Torrent Core`, `gRPC Server + Prowlarr Manager`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **What connects `ProwlarrConfig`, `Config`, `githubAsset` to the rest of the system?**
  _29 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Architecture Overview` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._
- **Should `Go Torrent Core` be split into smaller, more focused modules?**
  _Cohesion score 0.13 - nodes in this community are weakly interconnected._
- **Should `Torrent Service + Tests` be split into smaller, more focused modules?**
  _Cohesion score 0.1 - nodes in this community are weakly interconnected._