# Graph Report - .  (2026-04-21)

## Corpus Check
- Corpus is ~22,228 words - fits in a single context window. You may not need a graph.

## Summary
- 228 nodes · 296 edges · 36 communities detected
- Extraction: 71% EXTRACTED · 29% INFERRED · 0% AMBIGUOUS · INFERRED: 85 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Architecture and Planning Docs|Architecture and Planning Docs]]
- [[_COMMUNITY_Go Torrent Engine|Go Torrent Engine]]
- [[_COMMUNITY_Spring REST Stats API|Spring REST Stats API]]
- [[_COMMUNITY_Go Server Bootstrap|Go Server Bootstrap]]
- [[_COMMUNITY_Session Tracking|Session Tracking]]
- [[_COMMUNITY_Speed Tracker|Speed Tracker]]
- [[_COMMUNITY_gRPC Client Stub|gRPC Client Stub]]
- [[_COMMUNITY_Torrents Management UI|Torrents Management UI]]
- [[_COMMUNITY_Video Player Setup|Video Player Setup]]
- [[_COMMUNITY_Session Design Docs|Session Design Docs]]
- [[_COMMUNITY_Magnet Input Validation|Magnet Input Validation]]
- [[_COMMUNITY_React App Root|React App Root]]
- [[_COMMUNITY_Spring gRPC Config|Spring gRPC Config]]
- [[_COMMUNITY_Status Panel Widget|Status Panel Widget]]
- [[_COMMUNITY_Frontend Utilities|Frontend Utilities]]
- [[_COMMUNITY_Spring Boot Entry Point|Spring Boot Entry Point]]
- [[_COMMUNITY_CORS Configuration|CORS Configuration]]
- [[_COMMUNITY_Error Handling|Error Handling]]
- [[_COMMUNITY_Byte Formatting Util|Byte Formatting Util]]
- [[_COMMUNITY_Spring Context Tests|Spring Context Tests]]
- [[_COMMUNITY_Skip Overlay UI|Skip Overlay UI]]
- [[_COMMUNITY_Go Repo Tests|Go Repo Tests]]
- [[_COMMUNITY_gRPC Test Client|gRPC Test Client]]
- [[_COMMUNITY_Subtitle Panel|Subtitle Panel]]
- [[_COMMUNITY_Video Player Component|Video Player Component]]
- [[_COMMUNITY_Session Response DTO|Session Response DTO]]
- [[_COMMUNITY_Add Response DTO|Add Response DTO]]
- [[_COMMUNITY_List Response DTO|List Response DTO]]
- [[_COMMUNITY_Stats Response DTO|Stats Response DTO]]
- [[_COMMUNITY_Vite Build Config|Vite Build Config]]
- [[_COMMUNITY_React Entry Point|React Entry Point]]
- [[_COMMUNITY_Vite Type Declarations|Vite Type Declarations]]
- [[_COMMUNITY_Frontend Types|Frontend Types]]
- [[_COMMUNITY_CORS Config Doc|CORS Config Doc]]
- [[_COMMUNITY_gRPC Config Doc|gRPC Config Doc]]
- [[_COMMUNITY_Go Client Doc|Go Client Doc]]

## God Nodes (most connected - your core abstractions)
1. `TorrentController` - 12 edges
2. `MagnetPlay Project (CLAUDE.md)` - 11 edges
3. `TorrentGrpcClient` - 10 edges
4. `TorrentService` - 10 edges
5. `Repository` - 9 edges
6. `TorrentService` - 9 edges
7. `Implementation Plan: Torrent Dashboard (2026-04-19)` - 9 edges
8. `service.go (Go: gRPC TorrentService Implementation)` - 8 edges
9. `Shared Proto Contract: backend/proto/torrent.proto` - 7 edges
10. `StartServer()` - 6 edges

## Surprising Connections (you probably didn't know these)
- `StartServer()` --calls--> `NewRepository()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\repository.go
- `SpeedTracker: Rolling 5-Second Window Speed Calculator` --semantically_similar_to--> `Prometheus + Grafana Monitoring`  [INFERRED] [semantically similar]
  POC/poc-torrent-dashboard.md → README.md
- `StartServer()` --calls--> `NewClient()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\client.go
- `Spring WebFlux Flux Lifecycle Hooks for Session Tracking` --references--> `TorrentController (Spring REST)`  [EXTRACTED]
  POC/poc-torrent-dashboard.md → CLAUDE.md
- `HTTP Range Requests (HTTP 206) for Streaming` --conceptually_related_to--> `Session Tracking Behavior: Each HTTP Range = New Session`  [INFERRED]
  README.md → CLAUDE.md

## Hyperedges (group relationships)
- **End-to-End Torrent Stats Pipeline (SpeedTracker → gRPC → Spring → StatusPanel)** — plan_speed_tracker_go, plan_get_torrent_stats_rpc, plan_torrent_stats_response_dto, plan_status_panel_tsx [INFERRED 0.88]
- **Torrent Management Flow (Proto RPCs → Go Handlers → Spring Endpoints → TorrentsPage)** — plan_pause_resume_delete_rpcs, plan_list_torrents_rpc, plan_torrents_page_tsx, claudemd_go_repository [INFERRED 0.85]
- **Session Lifecycle Flow (Flux hooks → SessionManager → SessionResponse → UI)** — poc_session_flux_hooks, poc_session_manager_java, poc_streaming_session_record, plan_session_response_dto [EXTRACTED 0.92]

## Communities

### Community 0 - "Architecture and Planning Docs"
Cohesion: 0.07
Nodes (43): Three-Layer Architecture (Browser → Spring → Go → Peers), File ID Format: {infoHash}:{fileIndex}, Frontend Stack: Vite + React 18 + TypeScript, mapper.go (Go: Proto FileInfo Mapper), Rationale: Go Sidecar Owns All Torrent State, repository.go (Go: In-Memory Torrent Store), service.go (Go: gRPC TorrentService Implementation), God Node: App.tsx (#2 by Edge Count) (+35 more)

### Community 1 - "Go Torrent Engine"
Cohesion: 0.12
Nodes (6): toFileInfoList(), NewRepository(), prioritize(), Repository, TorrentInfo, TorrentService

### Community 2 - "Spring REST Stats API"
Cohesion: 0.11
Nodes (3): TorrentController, TorrentService, TorrentStatsControllerTest

### Community 3 - "Go Server Bootstrap"
Cohesion: 0.18
Nodes (9): NewClient(), Config, Default(), main(), StartServer(), NewTorrentService(), newTestRepository(), TestGetTorrentStats_UnknownHash() (+1 more)

### Community 4 - "Session Tracking"
Cohesion: 0.24
Nodes (3): SessionManager, SessionManagerTest, withClosed()

### Community 5 - "Speed Tracker"
Cohesion: 0.35
Nodes (6): NewSpeedTracker(), TestSpeedTracker_OldSamplesDropped(), TestSpeedTracker_SingleSample(), TestSpeedTracker_ZeroAtStart(), sample, SpeedTracker

### Community 6 - "gRPC Client Stub"
Cohesion: 0.2
Nodes (1): TorrentGrpcClient

### Community 7 - "Torrents Management UI"
Cohesion: 0.36
Nodes (7): deleteTorrent(), fetchData(), formatBytes(), formatSpeed(), pause(), resume(), withAction()

### Community 8 - "Video Player Setup"
Cohesion: 0.33
Nodes (2): fmtBytes(), fmtSpeed()

### Community 9 - "Session Design Docs"
Cohesion: 0.33
Nodes (7): Session Tracking Behavior: Each HTTP Range = New Session, SessionResponse.java (Spring DTO), F2ii: Session Tracking (Spring SessionManager), Spring WebFlux Flux Lifecycle Hooks for Session Tracking, SessionManager.java (Spring Component), StreamingSession.java (Java Record), Rationale: Extract Client IP via X-Forwarded-For Header

### Community 10 - "Magnet Input Validation"
Cohesion: 0.33
Nodes (2): validateMagnetLink(), TorrentUtil

### Community 11 - "React App Root"
Cohesion: 0.4
Nodes (2): handleFullReset(), handleReset()

### Community 12 - "Spring gRPC Config"
Cohesion: 0.5
Nodes (1): GrpcConfig

### Community 13 - "Status Panel Widget"
Cohesion: 0.67
Nodes (2): fmtBytes(), fmtSpeed()

### Community 14 - "Frontend Utilities"
Cohesion: 0.5
Nodes (0): 

### Community 15 - "Spring Boot Entry Point"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplication

### Community 16 - "CORS Configuration"
Cohesion: 0.67
Nodes (1): CorsConfig

### Community 17 - "Error Handling"
Cohesion: 0.67
Nodes (1): GlobalExceptionHandler

### Community 18 - "Byte Formatting Util"
Cohesion: 0.67
Nodes (1): ByteUtil

### Community 19 - "Spring Context Tests"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplicationTests

### Community 20 - "Skip Overlay UI"
Cohesion: 0.67
Nodes (0): 

### Community 21 - "Go Repo Tests"
Cohesion: 1.0
Nodes (0): 

### Community 22 - "gRPC Test Client"
Cohesion: 1.0
Nodes (1): GrpcClient

### Community 23 - "Subtitle Panel"
Cohesion: 1.0
Nodes (0): 

### Community 24 - "Video Player Component"
Cohesion: 1.0
Nodes (0): 

### Community 25 - "Session Response DTO"
Cohesion: 1.0
Nodes (0): 

### Community 26 - "Add Response DTO"
Cohesion: 1.0
Nodes (0): 

### Community 27 - "List Response DTO"
Cohesion: 1.0
Nodes (0): 

### Community 28 - "Stats Response DTO"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "Vite Build Config"
Cohesion: 1.0
Nodes (0): 

### Community 30 - "React Entry Point"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "Vite Type Declarations"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "Frontend Types"
Cohesion: 1.0
Nodes (0): 

### Community 33 - "CORS Config Doc"
Cohesion: 1.0
Nodes (1): CorsConfig (Spring CORS Config)

### Community 34 - "gRPC Config Doc"
Cohesion: 1.0
Nodes (1): GrpcConfig (Spring gRPC Channel Config)

### Community 35 - "Go Client Doc"
Cohesion: 1.0
Nodes (1): client.go (Go: anacrolix/torrent client creation)

## Knowledge Gaps
- **20 isolated node(s):** `Config`, `sample`, `GrpcClient`, `Three-Layer Architecture (Browser → Spring → Go → Peers)`, `Rationale: Spring Boot as Thin Translation Layer` (+15 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Go Repo Tests`** (2 nodes): `repository_test.go`, `TestTorrentInfo_PauseResume()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `gRPC Test Client`** (2 nodes): `GrpcClient.java`, `GrpcClient`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Subtitle Panel`** (2 nodes): `SubtitlePanel.tsx`, `handleFilePick()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Video Player Component`** (2 nodes): `VideoPlayer.tsx`, `VideoPlayer()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Session Response DTO`** (1 nodes): `SessionResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Add Response DTO`** (1 nodes): `TorrentAddResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `List Response DTO`** (1 nodes): `TorrentListResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Stats Response DTO`** (1 nodes): `TorrentStatsResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Build Config`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React Entry Point`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Type Declarations`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend Types`** (1 nodes): `index.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `CORS Config Doc`** (1 nodes): `CorsConfig (Spring CORS Config)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `gRPC Config Doc`** (1 nodes): `GrpcConfig (Spring gRPC Channel Config)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Go Client Doc`** (1 nodes): `client.go (Go: anacrolix/torrent client creation)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `TorrentController` connect `Spring REST Stats API` to `Go Torrent Engine`, `Magnet Input Validation`, `Session Tracking`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **Why does `TorrentService` connect `Go Torrent Engine` to `Speed Tracker`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **Why does `TorrentService` connect `Spring REST Stats API` to `Go Torrent Engine`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **What connects `Config`, `sample`, `GrpcClient` to the rest of the system?**
  _20 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Architecture and Planning Docs` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._
- **Should `Go Torrent Engine` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._
- **Should `Spring REST Stats API` be split into smaller, more focused modules?**
  _Cohesion score 0.11 - nodes in this community are weakly interconnected._