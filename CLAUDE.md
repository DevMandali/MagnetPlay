# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

MagnetPlay is a peer-to-peer video streaming system — add a magnet URL and start watching immediately without waiting for the full download. Think Stremio, built from scratch.

## Architecture

Three-layer system communicating top-to-bottom:

```
Browser (frontend/index.html)
    ↕ HTTP REST + Range requests (port 8080)
Spring Boot — mp-spring (Java 17, WebFlux)
    ↕ gRPC server-side streaming (port 50051)
Go sidecar — go-server (anacrolix/torrent)
    ↕ BitTorrent protocol
Peer Network (DHT + Trackers)
```

**Key design decisions:**
- Spring Boot is a thin translation layer: HTTP Range headers → gRPC `StreamRequest`. It does not touch torrent logic.
- The Go sidecar owns all torrent state. It uses `anacrolix/torrent` as the BitTorrent engine.
- File IDs are `{infoHash}:{fileIndex}` (e.g. `abc123:0`), generated in `go-server/internal/torrent/mapper.go`.
- gRPC `StreamFile` is server-side streaming: Go pushes 256KB `FileChunk` messages; Spring wraps them as a reactive `Flux<DataBuffer>`.
- Piece prioritization at streaming start: first 5 pieces (moov atom), last 5 pieces (seek metadata), then the requested range — see `service.go:prioritize()`.
- The frontend is a **Vite + React 18 + TypeScript** app (`frontend/src/`). Entry: `frontend/src/main.tsx`. Dev server: `cd frontend && npm run dev`. Components: `VideoPlayer.tsx`, `SubtitlePanel.tsx`, `NetflixSkipOverlay.tsx`. Types: `frontend/src/types/index.ts`.
## Shared Proto Contract

The gRPC API is defined in `backend/proto/torrent.proto` and used by both services. When you change the proto, regenerate code for both:

- **Go**: `cd backend && make -f MakeFile proto`
- **Java**: proto is auto-compiled during `mvn compile` via `protobuf-maven-plugin`

Manual Go proto generation:
```bash
protoc --go_out=go-server --go_opt=paths=source_relative \
       --go-grpc_out=go-server --go-grpc_opt=paths=source_relative \
       proto/torrent.proto
```

Windows prerequisites: `winget install Google.Protobuf` and `winget install ezwinports.make`

## Running the Services

**Go sidecar** (starts gRPC server on port 50051, downloads to `./downloads`):
```bash
cd backend
make -f MakeFile run
# or directly:
cd backend/go-server && go run main.go
```

**Spring Boot** (starts REST server on port 8080):
```bash
cd backend/mp-spring
./mvnw spring-boot:run
```

**Frontend** (Vite dev server on port 5173):
```bash
cd frontend && npm run dev
```

## Build

```bash
# Go: generate proto + build binary (outputs to go-server/bin/server.exe)
cd backend && make -f MakeFile all

# Spring Boot: compile + package
cd backend/mp-spring && ./mvnw clean package -DskipTests
```

## Tests

```bash
# Go
cd backend/go-server && go test ./...

# Spring Boot
cd backend/mp-spring && ./mvnw test
```

## Key Configuration

**Spring Boot** (`backend/mp-spring/src/main/resources/application.yml`):
- REST port: `8080`
- gRPC sidecar address: env `GRPC_TORRENT_SERVICE_ADDRESS` (default `localhost:50051`)
- CORS allowed origins: `localhost:3000`, `localhost:5500`, `127.0.0.1:5500`, and `"null"` (local file:// access — temporary until a proper frontend server is set up)
- Swagger UI: `http://localhost:8080/swagger-ui.html`
- API docs: `http://localhost:8080/magnetplay-api-docs`

**Go sidecar** (`backend/go-server/config/config.go`):
- gRPC port: `50051`
- Data dir: `./downloads`
- Metadata timeout: `60s` (magnet link metadata resolution)

## Spring Boot Package Structure

`org.devMandali.magnetPlay`:
- `controller/TorrentController` — REST endpoints: `POST /v1/torrent/add`, `GET /v1/torrent/stream/{infoHash}?fileId=`
- `service/TorrentService` — translates HTTP requests to gRPC calls
- `client/TorrentGrpcClient` — gRPC stub wrapper
- `config/CorsConfig` — exposes `Content-Range`, `Accept-Ranges`, `Content-Length` (required for browser video seeking)
- `config/GrpcConfig` — configures the gRPC channel to the Go sidecar

## Go Sidecar Package Structure

`server/internal/torrent`:
- `client.go` — creates the `anacrolix/torrent` client
- `repository.go` — in-memory store of torrents and their files (keyed by info-hash); handles concurrent access
- `service.go` — implements the gRPC `TorrentService` interface: `AddTorrent`, `GetFileInfo`, `StreamFile`
- `mapper.go` — converts torrent file list to proto `FileInfo`, filters to video-only files, assigns file IDs

## Commit Message Convention

```
Feature(scope): description
```
Examples from history: `Feature(UI)`, `Feature(Player)`, `Feature(CC + Priority Setting)`, `Feature(CORS + Frontend)`.

## graphify

This project has a graphify knowledge graph at `graphify-out/`. The graph stores nodes (files, functions, concepts), edges (calls, imports, inferred dependencies), community clusters, god nodes (highest-edge-count = highest change risk), and hyperedges (cross-cutting flows).

**All Superpowers skills must use the graph before touching files:**
- Read `graphify-out/GRAPH_REPORT.md` first — god nodes, community map, surprising connections, and knowledge gaps give you architecture context without reading raw files.
- Use community boundaries as natural task/module split lines when planning or dispatching parallel agents.
- Use god nodes (top 10 by edge count) as your high-risk file list — flag them in plans and code reviews.
- Use `/graphify query "<question>"` to BFS/DFS the graph for relevant subgraphs before exploring the codebase directly.
- After modifying any code file, run `graphify update .` (AST-only, no API cost) to keep the graph current.
- If `graphify-out/wiki/index.md` exists, navigate it instead of reading raw files.
