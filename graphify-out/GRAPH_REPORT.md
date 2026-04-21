# Graph Report - .  (2026-04-22)

## Corpus Check
- 7 files · ~5,000 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 230 nodes · 299 edges · 35 communities detected
- Extraction: 71% EXTRACTED · 29% INFERRED · 0% AMBIGUOUS · INFERRED: 86 edges (avg confidence: 0.81)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Architecture Documentation|Architecture Documentation]]
- [[_COMMUNITY_Go Torrent Core|Go Torrent Core]]
- [[_COMMUNITY_Spring REST Layer|Spring REST Layer]]
- [[_COMMUNITY_Go gRPC Server|Go gRPC Server]]
- [[_COMMUNITY_Session Management|Session Management]]
- [[_COMMUNITY_gRPC Client Bridge|gRPC Client Bridge]]
- [[_COMMUNITY_Speed Tracking|Speed Tracking]]
- [[_COMMUNITY_Stats Tests|Stats Tests]]
- [[_COMMUNITY_React App Shell|React App Shell]]
- [[_COMMUNITY_Video Setup Utilities|Video Setup Utilities]]
- [[_COMMUNITY_Session Tracking Design|Session Tracking Design]]
- [[_COMMUNITY_gRPC Config|gRPC Config]]
- [[_COMMUNITY_Status Panel UI|Status Panel UI]]
- [[_COMMUNITY_Frontend Utilities|Frontend Utilities]]
- [[_COMMUNITY_Spring App Entry|Spring App Entry]]
- [[_COMMUNITY_CORS Config|CORS Config]]
- [[_COMMUNITY_Error Handling|Error Handling]]
- [[_COMMUNITY_Byte Formatting|Byte Formatting]]
- [[_COMMUNITY_Spring App Tests|Spring App Tests]]
- [[_COMMUNITY_Skip Overlay UI|Skip Overlay UI]]
- [[_COMMUNITY_Repository Tests|Repository Tests]]
- [[_COMMUNITY_gRPC Client Tests|gRPC Client Tests]]
- [[_COMMUNITY_Subtitle Panel UI|Subtitle Panel UI]]
- [[_COMMUNITY_Video Player UI|Video Player UI]]
- [[_COMMUNITY_Session Response Model|Session Response Model]]
- [[_COMMUNITY_Torrent Add Response|Torrent Add Response]]
- [[_COMMUNITY_Torrent List Response|Torrent List Response]]
- [[_COMMUNITY_Torrent Stats Response|Torrent Stats Response]]
- [[_COMMUNITY_Vite Config|Vite Config]]
- [[_COMMUNITY_React Entry Point|React Entry Point]]
- [[_COMMUNITY_Vite Types|Vite Types]]
- [[_COMMUNITY_Frontend Types|Frontend Types]]
- [[_COMMUNITY_CORS Config Docs|CORS Config Docs]]
- [[_COMMUNITY_gRPC Config Docs|gRPC Config Docs]]
- [[_COMMUNITY_Go Client Docs|Go Client Docs]]

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
- `SpeedTracker: Rolling 5-Second Window Speed Calculator` --semantically_similar_to--> `Prometheus + Grafana Monitoring`  [INFERRED] [semantically similar]
  POC/poc-torrent-dashboard.md → README.md
- `StartServer()` --calls--> `NewClient()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\client.go
- `StartServer()` --calls--> `NewRepository()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\repository.go
- `StartServer()` --calls--> `NewTorrentService()`  [INFERRED]
  backend\go-server\internal\grpc\server.go → backend\go-server\internal\torrent\service.go
- `HTTP Range Requests (HTTP 206) for Streaming` --conceptually_related_to--> `Session Tracking Behavior: Each HTTP Range = New Session`  [INFERRED]
  README.md → CLAUDE.md

## Hyperedges (group relationships)
- **End-to-End Torrent Stats Pipeline (SpeedTracker → gRPC → Spring → StatusPanel)** — plan_speed_tracker_go, plan_get_torrent_stats_rpc, plan_torrent_stats_response_dto, plan_status_panel_tsx [INFERRED 0.88]
- **Torrent Management Flow (Proto RPCs → Go Handlers → Spring Endpoints → TorrentsPage)** — plan_pause_resume_delete_rpcs, plan_list_torrents_rpc, plan_torrents_page_tsx, claudemd_go_repository [INFERRED 0.85]
- **Session Lifecycle Flow (Flux hooks → SessionManager → SessionResponse → UI)** — poc_session_flux_hooks, poc_session_manager_java, poc_streaming_session_record, plan_session_response_dto [EXTRACTED 0.92]

## Communities

### Community 0 - "Architecture Documentation"
Cohesion: 0.07
Nodes (44): Three-Layer Architecture (Browser → Spring → Go → Peers), File ID Format: {infoHash}:{fileIndex}, Frontend Stack: Vite + React 18 + TypeScript, mapper.go (Go: Proto FileInfo Mapper), Rationale: Go Sidecar Owns All Torrent State, repository.go (Go: In-Memory Torrent Store), service.go (Go: gRPC TorrentService Implementation), God Node: App.tsx (#2 by Edge Count) (+36 more)

### Community 1 - "Go Torrent Core"
Cohesion: 0.15
Nodes (5): toFileInfoList(), prioritize(), Repository, TorrentInfo, TorrentService

### Community 2 - "Spring REST Layer"
Cohesion: 0.11
Nodes (4): validateMagnetLink(), TorrentController, TorrentService, TorrentUtil

### Community 3 - "Go gRPC Server"
Cohesion: 0.12
Nodes (13): NewClient(), Config, Default(), main(), NewRepository(), StartServer(), deleteTorrent(), fetchData() (+5 more)

### Community 4 - "Session Management"
Cohesion: 0.24
Nodes (3): SessionManager, SessionManagerTest, withClosed()

### Community 5 - "gRPC Client Bridge"
Cohesion: 0.17
Nodes (1): TorrentGrpcClient

### Community 6 - "Speed Tracking"
Cohesion: 0.35
Nodes (6): NewSpeedTracker(), TestSpeedTracker_OldSamplesDropped(), TestSpeedTracker_SingleSample(), TestSpeedTracker_ZeroAtStart(), sample, SpeedTracker

### Community 7 - "Stats Tests"
Cohesion: 0.27
Nodes (5): NewTorrentService(), newTestRepository(), TestGetTorrentStats_UnknownHash(), TestListTorrents_EmptyRepo(), TorrentStatsControllerTest

### Community 8 - "React App Shell"
Cohesion: 0.29
Nodes (2): handleFullReset(), handleReset()

### Community 9 - "Video Setup Utilities"
Cohesion: 0.33
Nodes (2): fmtBytes(), fmtSpeed()

### Community 10 - "Session Tracking Design"
Cohesion: 0.4
Nodes (6): Session Tracking Behavior: Each HTTP Range = New Session, SessionResponse.java (Spring DTO), F2ii: Session Tracking (Spring SessionManager), SessionManager.java (Spring Component), StreamingSession.java (Java Record), Rationale: Extract Client IP via X-Forwarded-For Header

### Community 11 - "gRPC Config"
Cohesion: 0.5
Nodes (1): GrpcConfig

### Community 12 - "Status Panel UI"
Cohesion: 0.67
Nodes (2): fmtBytes(), fmtSpeed()

### Community 13 - "Frontend Utilities"
Cohesion: 0.5
Nodes (0): 

### Community 14 - "Spring App Entry"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplication

### Community 15 - "CORS Config"
Cohesion: 0.67
Nodes (1): CorsConfig

### Community 16 - "Error Handling"
Cohesion: 0.67
Nodes (1): GlobalExceptionHandler

### Community 17 - "Byte Formatting"
Cohesion: 0.67
Nodes (1): ByteUtil

### Community 18 - "Spring App Tests"
Cohesion: 0.67
Nodes (1): MpSpringBackendApplicationTests

### Community 19 - "Skip Overlay UI"
Cohesion: 0.67
Nodes (0): 

### Community 20 - "Repository Tests"
Cohesion: 1.0
Nodes (0): 

### Community 21 - "gRPC Client Tests"
Cohesion: 1.0
Nodes (1): GrpcClient

### Community 22 - "Subtitle Panel UI"
Cohesion: 1.0
Nodes (0): 

### Community 23 - "Video Player UI"
Cohesion: 1.0
Nodes (0): 

### Community 24 - "Session Response Model"
Cohesion: 1.0
Nodes (0): 

### Community 25 - "Torrent Add Response"
Cohesion: 1.0
Nodes (0): 

### Community 26 - "Torrent List Response"
Cohesion: 1.0
Nodes (0): 

### Community 27 - "Torrent Stats Response"
Cohesion: 1.0
Nodes (0): 

### Community 28 - "Vite Config"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "React Entry Point"
Cohesion: 1.0
Nodes (0): 

### Community 30 - "Vite Types"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "Frontend Types"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "CORS Config Docs"
Cohesion: 1.0
Nodes (1): CorsConfig (Spring CORS Config)

### Community 33 - "gRPC Config Docs"
Cohesion: 1.0
Nodes (1): GrpcConfig (Spring gRPC Channel Config)

### Community 34 - "Go Client Docs"
Cohesion: 1.0
Nodes (1): client.go (Go: anacrolix/torrent client creation)

## Knowledge Gaps
- **20 isolated node(s):** `Config`, `sample`, `GrpcClient`, `Three-Layer Architecture (Browser → Spring → Go → Peers)`, `Rationale: Spring Boot as Thin Translation Layer` (+15 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Repository Tests`** (2 nodes): `repository_test.go`, `TestTorrentInfo_PauseResume()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `gRPC Client Tests`** (2 nodes): `GrpcClient.java`, `GrpcClient`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Subtitle Panel UI`** (2 nodes): `SubtitlePanel.tsx`, `handleFilePick()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Video Player UI`** (2 nodes): `VideoPlayer.tsx`, `VideoPlayer()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Session Response Model`** (1 nodes): `SessionResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Add Response`** (1 nodes): `TorrentAddResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent List Response`** (1 nodes): `TorrentListResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Torrent Stats Response`** (1 nodes): `TorrentStatsResponse.java`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Config`** (1 nodes): `vite.config.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `React Entry Point`** (1 nodes): `main.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Vite Types`** (1 nodes): `vite-env.d.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Frontend Types`** (1 nodes): `index.ts`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `CORS Config Docs`** (1 nodes): `CorsConfig (Spring CORS Config)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `gRPC Config Docs`** (1 nodes): `GrpcConfig (Spring gRPC Channel Config)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Go Client Docs`** (1 nodes): `client.go (Go: anacrolix/torrent client creation)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `TorrentController` connect `Spring REST Layer` to `Go gRPC Server`, `Session Management`, `Stats Tests`?**
  _High betweenness centrality (0.062) - this node is a cross-community bridge._
- **Why does `TorrentService` connect `Spring REST Layer` to `Go Torrent Core`, `gRPC Client Bridge`, `Stats Tests`?**
  _High betweenness centrality (0.050) - this node is a cross-community bridge._
- **Why does `TorrentService` connect `Go Torrent Core` to `Speed Tracking`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **What connects `Config`, `sample`, `GrpcClient` to the rest of the system?**
  _20 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Architecture Documentation` be split into smaller, more focused modules?**
  _Cohesion score 0.07 - nodes in this community are weakly interconnected._
- **Should `Spring REST Layer` be split into smaller, more focused modules?**
  _Cohesion score 0.11 - nodes in this community are weakly interconnected._
- **Should `Go gRPC Server` be split into smaller, more focused modules?**
  _Cohesion score 0.12 - nodes in this community are weakly interconnected._