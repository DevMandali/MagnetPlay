# MVP Specification Document

**Project:** MagnetPlay  
**Version:** 1.0 - MVP  
**Status:** Ready for Development  
**Sprint Duration:** 2 weeks  
**Total Timeline:** 8 weeks (4 sprints)  
**Last Updated:** January 27, 2026

---

## Table of Contents

- [1. MVP Scope](#1-mvp-scope)
- [2. Success Criteria](#2-success-criteria)
- [3. Team Structure](#3-team-structure)
- [4. Sprint Plan](#4-sprint-plan)
- [5. Team Alpha Tasks](#5-team-alpha-tasks)
- [6. Team Beta Tasks](#6-team-beta-tasks)
- [7. Integration Points](#7-integration-points)
- [8. Dependencies](#8-dependencies)
- [9. Risk Management](#9-risk-management)
- [10. Testing Strategy](#10-testing-strategy)
- [11. Acceptance Criteria](#11-acceptance-criteria)

---

## 1. MVP Scope

### 1.1 In Scope (MUST HAVE)

#### Core Features
- ✅ Add torrent via magnet link
- ✅ Select video file from multi-file torrents
- ✅ Stream video with < 10 second startup time
- ✅ Forward/backward seeking with piece prioritization
- ✅ Display download progress and peer count
- ✅ Basic LRU cache (10GB limit)
- ✅ Error handling for common failures

#### Technical Requirements
- ✅ Frontend: React SPA with Video.js
- ✅ Backend: Spring Boot REST API
- ✅ Sidecar: Go + libtorrent gRPC service
- ✅ Docker Compose deployment
- ✅ Basic monitoring (logs + Prometheus metrics)

### 1.2 Out of Scope (FUTURE)

- ❌ Subtitle support
- ❌ Multiple quality selection
- ❌ Playlist/queue functionality
- ❌ User authentication
- ❌ Content discovery/search
- ❌ Mobile apps
- ❌ Chromecast support
- ❌ Video transcoding

### 1.3 MVP User Flow

```
User Journey:
1. Opens app → Sees torrent input screen
2. Pastes magnet link → Clicks "Add"
3. Waits 5-10 seconds → Sees file list
4. Clicks video file → Player loads
5. Video starts within 10 seconds → Watches content
6. Seeks to different position → Buffer updates, continues playing
7. Checks metrics → Sees download speed, peers, progress
```

---

## 2. Success Criteria

### 2.1 Functional Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Startup Time** | < 10 seconds | Time from file selection to first frame |
| **Seek Latency** | < 3 seconds | Time from seek to playback resume |
| **Cache Hit Rate** | > 80% | % of requests served from cache |
| **Peer Discovery** | > 5 peers | Average peers within 2 minutes |
| **Error Recovery** | > 90% | % of failures gracefully handled |

### 2.2 Non-Functional Metrics

| Aspect | Target | Validation |
|--------|--------|------------|
| **Code Coverage** | > 80% | Unit + integration tests |
| **API Response Time** | < 100ms | 95th percentile |
| **Memory Usage** | < 2GB | Per service |
| **Uptime** | > 99% | During testing period |
| **Documentation** | 100% | All APIs and components documented |

### 2.3 Definition of Done (MVP)

- [ ] All user stories completed
- [ ] E2E test suite passes
- [ ] Load test with 10 concurrent streams succeeds
- [ ] Documentation complete (README, API docs)
- [ ] No P0 or P1 bugs
- [ ] Deployed to staging environment
- [ ] Demo to stakeholders completed

---

## 3. Team Structure

### 3.1 Team Alpha: Frontend & Backend

**Focus:** User interface and HTTP streaming layer

**Team Composition:**
- **Team Lead:** Sarah (Senior Full-Stack Developer)
- **Frontend Developer 1:** Alex (React specialist)
- **Frontend Developer 2:** Jamie (UI/UX focus)
- **Backend Developer:** Chris (Spring Boot expert)

**Responsibilities:**
- React SPA with Video.js player
- REST API endpoints
- WebSocket for real-time updates
- HTTP 206 byte-range streaming
- Frontend state management

### 3.2 Team Beta: Sidecar & Infrastructure

**Focus:** Torrent engine and system integration

**Team Composition:**
- **Team Lead:** Morgan (Senior Backend/DevOps)
- **Backend Developer 1:** Taylor (Go specialist)
- **Backend Developer 2:** Jordan (Distributed systems)
- **DevOps Engineer:** Riley (Docker/K8s expert)

**Responsibilities:**
- Go sidecar with libtorrent integration
- gRPC server implementation
- LRU cache management
- Docker Compose setup
- Monitoring infrastructure

### 3.3 Shared Resources

- **Tech Lead:** Dana (Cross-team architecture)
- **QA Engineer:** Casey (Test automation)
- **Product Owner:** Avery (Prioritization)

---

## 4. Sprint Plan

### 4.1 Sprint Breakdown

```
Sprint 1 (Weeks 1-2): Foundation
├── Team Alpha: Basic UI + REST skeleton
└── Team Beta: Sidecar core + gRPC

Sprint 2 (Weeks 3-4): Integration
├── Team Alpha: Video player + API calls
└── Team Beta: Torrent management + cache

Sprint 3 (Weeks 5-6): Polish
├── Team Alpha: Metrics dashboard + error handling
└── Team Beta: Performance optimization

Sprint 4 (Weeks 7-8): Testing & Deployment
├── Team Alpha: E2E tests + documentation
└── Team Beta: Load testing + production setup
```

### 4.2 Milestones

| Milestone | Date | Deliverable |
|-----------|------|-------------|
| **M1: Basic Streaming** | Week 2 | Can add torrent and see file list |
| **M2: Working Player** | Week 4 | Can play video (no seek yet) |
| **M3: Full Features** | Week 6 | Seeking, metrics, error handling |
| **M4: MVP Complete** | Week 8 | Production-ready deployment |

---

## 5. Team Alpha Tasks

### 5.1 Sprint 1: Foundation (Weeks 1-2)

#### ALPHA-1: Project Setup
**Owner:** Sarah (Team Lead)  
**Story Points:** 2  
**Priority:** P0

**Description:** Initialize React and Spring Boot projects with basic structure.

**Tasks:**
- [ ] Create React app with TypeScript template
- [ ] Set up Material-UI and Video.js dependencies
- [ ] Initialize Spring Boot project with Maven
- [ ] Configure application.yml with profiles (dev/prod)
- [ ] Set up ESLint, Prettier, and pre-commit hooks
- [ ] Create Docker Compose for local development

**Acceptance Criteria:**
- [ ] `npm start` runs frontend on localhost:3000
- [ ] `mvn spring-boot:run` runs backend on localhost:8080
- [ ] Pre-commit hooks format code automatically
- [ ] Docker Compose brings up both services

**Estimated Time:** 8 hours

---

#### ALPHA-2: Torrent Input UI
**Owner:** Alex (Frontend Dev)  
**Story Points:** 3  
**Priority:** P0

**Description:** Build UI for users to paste magnet links.

**Tasks:**
- [ ] Create `TorrentInputPanel` component
- [ ] Add text field with validation (regex for magnet links)
- [ ] Implement "Add Torrent" button with loading state
- [ ] Display error messages for invalid input
- [ ] Add basic styling with Material-UI
- [ ] Write unit tests for component

**Acceptance Criteria:**
- [ ] User can paste magnet link
- [ ] Invalid links show error message
- [ ] Button disables during submission
- [ ] Unit tests cover validation logic

**Estimated Time:** 12 hours

---

#### ALPHA-3: REST Controller - Add Torrent
**Owner:** Chris (Backend Dev)  
**Story Points:** 5  
**Priority:** P0

**Description:** Create REST endpoint to accept magnet links.

**Tasks:**
- [ ] Create `TorrentController` with POST /api/torrents
- [ ] Implement request DTO with validation (@Valid)
- [ ] Create `TorrentService` interface
- [ ] Add mock gRPC stub for now (actual in Sprint 2)
- [ ] Return torrent ID and status
- [ ] Add error handling (400, 500 responses)
- [ ] Write controller and service tests
- [ ] Document API with Swagger/OpenAPI

**Acceptance Criteria:**
- [ ] POST /api/torrents accepts JSON with magnetLink
- [ ] Returns 200 with torrentId on success
- [ ] Returns 400 for invalid magnet links
- [ ] Unit tests achieve >80% coverage
- [ ] Swagger UI accessible at /swagger-ui.html

**Estimated Time:** 20 hours

---

#### ALPHA-4: File List UI
**Owner:** Jamie (Frontend Dev)  
**Story Points:** 3  
**Priority:** P1

**Description:** Display video files from torrent for user selection.

**Tasks:**
- [ ] Create `FileSelector` component
- [ ] Fetch file list from API (GET /api/torrents/{id}/files)
- [ ] Render list with file names, sizes, formats
- [ ] Add click handler to select file
- [ ] Show loading spinner while fetching
- [ ] Filter to show only video files (.mp4, .mkv, .avi, .webm)
- [ ] Write integration test with mock API

**Acceptance Criteria:**
- [ ] User sees list of video files after adding torrent
- [ ] Clicking file triggers navigation to player
- [ ] Non-video files are hidden
- [ ] Loading state shown during API call

**Estimated Time:** 12 hours

---

#### ALPHA-5: File List API Endpoint
**Owner:** Chris (Backend Dev)  
**Story Points:** 3  
**Priority:** P1

**Description:** Endpoint to retrieve files from torrent.

**Tasks:**
- [ ] Create GET /api/torrents/{id}/files endpoint
- [ ] Call sidecar gRPC to get file metadata
- [ ] Map response to FileInfo DTO
- [ ] Filter video files (check mime types)
- [ ] Handle torrent not found (404)
- [ ] Write integration test with mock gRPC
- [ ] Add to Swagger documentation

**Acceptance Criteria:**
- [ ] Returns array of FileInfo objects
- [ ] Each object has: index, name, size, mimeType
- [ ] Returns 404 if torrent doesn't exist
- [ ] Integration test validates response structure

**Estimated Time:** 12 hours

---

### 5.2 Sprint 2: Integration (Weeks 3-4)

#### ALPHA-6: Video.js Player Integration
**Owner:** Alex (Frontend Dev)  
**Story Points:** 5  
**Priority:** P0

**Description:** Integrate Video.js to play video from backend stream.

**Tasks:**
- [ ] Install Video.js and React wrapper
- [ ] Create `VideoPlayer` component
- [ ] Configure Video.js options (controls, autoplay, preload)
- [ ] Set video source to /api/stream/{torrentId}/{fileIndex}
- [ ] Handle player events (play, pause, error)
- [ ] Add buffering indicator overlay
- [ ] Style player controls
- [ ] Test with sample video file

**Acceptance Criteria:**
- [ ] Video player renders correctly
- [ ] Plays video from backend stream URL
- [ ] Shows buffering indicator when loading
- [ ] Controls (play/pause/volume) work
- [ ] Error events logged to console

**Estimated Time:** 20 hours

---

#### ALPHA-7: HTTP Streaming Endpoint
**Owner:** Chris (Backend Dev)  
**Story Points:** 8  
**Priority:** P0

**Description:** Implement HTTP 206 byte-range streaming.

**Tasks:**
- [ ] Create `StreamingController` with GET /api/stream/{torrentId}/{fileIndex}
- [ ] Parse Range header (e.g., "bytes=0-1048575")
- [ ] Call sidecar gRPC to read data chunks
- [ ] Return HTTP 206 Partial Content
- [ ] Set Content-Range header correctly
- [ ] Handle missing pieces (return 416 Range Not Satisfiable)
- [ ] Implement streaming with StreamingResponseBody
- [ ] Write integration test with byte-range requests
- [ ] Load test with JMeter (100 concurrent requests)

**Acceptance Criteria:**
- [ ] Supports Range header parsing
- [ ] Returns 206 with correct Content-Range
- [ ] Streams data in chunks (no full buffer)
- [ ] Returns 416 if range unavailable
- [ ] Integration test validates byte-range logic
- [ ] Load test shows < 200ms latency (95th percentile)

**Estimated Time:** 32 hours

---

#### ALPHA-8: gRPC Client Integration
**Owner:** Chris (Backend Dev)  
**Story Points:** 5  
**Priority:** P0

**Description:** Connect backend to sidecar via gRPC.

**Tasks:**
- [ ] Add gRPC dependencies to pom.xml
- [ ] Copy proto files from sidecar project
- [ ] Generate Java stubs with protobuf-maven-plugin
- [ ] Create `GrpcClientService` with connection management
- [ ] Implement AddTorrent, GetStatus, ReadData calls
- [ ] Add retry policy with exponential backoff
- [ ] Implement circuit breaker (Resilience4j)
- [ ] Add health check endpoint
- [ ] Write unit tests with mock gRPC server

**Acceptance Criteria:**
- [ ] Backend connects to sidecar on startup
- [ ] Can add torrent via gRPC
- [ ] Retries failed calls (max 3 attempts)
- [ ] Circuit breaker opens after 5 consecutive failures
- [ ] Health check returns UP if sidecar reachable
- [ ] Unit tests mock gRPC responses

**Estimated Time:** 20 hours

---

#### ALPHA-9: State Management (React Context)
**Owner:** Jamie (Frontend Dev)  
**Story Points:** 5  
**Priority:** P1

**Description:** Centralize app state for torrent and playback.

**Tasks:**
- [ ] Create AppContext with useReducer
- [ ] Define state shape (activeTorrent, playbackState, metrics)
- [ ] Implement actions (ADD_TORRENT, SELECT_FILE, UPDATE_METRICS)
- [ ] Create custom hooks (useTorrent, usePlayback)
- [ ] Refactor components to use context
- [ ] Add localStorage persistence for torrent list
- [ ] Write unit tests for reducers

**Acceptance Criteria:**
- [ ] All components access state via context
- [ ] State updates trigger re-renders
- [ ] Torrent list persists in localStorage
- [ ] Reducers have 100% test coverage

**Estimated Time:** 20 hours

---

### 5.3 Sprint 3: Polish (Weeks 5-6)

#### ALPHA-10: Seek Functionality
**Owner:** Alex (Frontend Dev)  
**Story Points:** 5  
**Priority:** P0

**Description:** Enable seeking with piece prioritization.

**Tasks:**
- [ ] Listen to Video.js `seeking` event
- [ ] Calculate byte offset from seek time
- [ ] Call POST /api/torrents/{id}/prioritize with range
- [ ] Show loading overlay during seek
- [ ] Resume playback when pieces available
- [ ] Handle seek to unprioritized pieces
- [ ] Add seek bar with piece availability colors (green=downloaded, yellow=downloading, gray=not started)
- [ ] Test seeking edge cases (start, end, middle)

**Acceptance Criteria:**
- [ ] User can drag seek bar
- [ ] Playback resumes within 3 seconds
- [ ] Seek bar shows piece availability
- [ ] Works for seeks forward and backward
- [ ] No crashes on rapid seeks

**Estimated Time:** 20 hours

---

#### ALPHA-11: Prioritize API Endpoint
**Owner:** Chris (Backend Dev)  
**Story Points:** 3  
**Priority:** P0

**Description:** Endpoint to prioritize pieces for seeking.

**Tasks:**
- [ ] Create POST /api/torrents/{id}/prioritize
- [ ] Accept startByte and endByte in request
- [ ] Call sidecar gRPC PrioritizePieces
- [ ] Return estimated wait time
- [ ] Add validation (range within file bounds)
- [ ] Write unit test

**Acceptance Criteria:**
- [ ] Accepts byte range and forwards to sidecar
- [ ] Returns 200 with ETA
- [ ] Returns 400 for invalid ranges
- [ ] Unit test covers validation

**Estimated Time:** 12 hours

---

#### ALPHA-12: Metrics Dashboard
**Owner:** Jamie (Frontend Dev)  
**Story Points:** 5  
**Priority:** P1

**Description:** Display real-time download metrics.

**Tasks:**
- [ ] Create `MetricsDashboard` component
- [ ] Fetch metrics from GET /api/metrics
- [ ] Display: download speed, upload speed, peers, progress, cache size
- [ ] Use Chart.js for line graph (speed over time)
- [ ] Update every 2 seconds via polling
- [ ] Add expand/collapse toggle
- [ ] Style with responsive grid

**Acceptance Criteria:**
- [ ] Dashboard shows all 5 metrics
- [ ] Speed graph updates in real-time
- [ ] Responsive on mobile and desktop
- [ ] Can collapse to save space

**Estimated Time:** 20 hours

---

#### ALPHA-13: Metrics API Endpoint
**Owner:** Chris (Backend Dev)  
**Story Points:** 3  
**Priority:** P1

**Description:** Expose metrics from sidecar.

**Tasks:**
- [ ] Create GET /api/metrics endpoint
- [ ] Call sidecar gRPC GetMetrics
- [ ] Map to MetricsDTO
- [ ] Cache response for 1 second (avoid gRPC spam)
- [ ] Write unit test

**Acceptance Criteria:**
- [ ] Returns JSON with all metrics
- [ ] Response cached for 1 second
- [ ] Unit test validates caching

**Estimated Time:** 12 hours

---

#### ALPHA-14: WebSocket for Real-Time Updates
**Owner:** Chris (Backend Dev)  
**Story Points:** 5  
**Priority:** P2

**Description:** Push updates to frontend without polling.

**Tasks:**
- [ ] Add Spring WebSocket dependency
- [ ] Configure WebSocket endpoint /ws
- [ ] Create `WebSocketBroadcaster` service
- [ ] Subscribe to sidecar events (piece downloaded, status change)
- [ ] Broadcast to connected clients
- [ ] Handle client disconnect gracefully
- [ ] Write integration test

**Acceptance Criteria:**
- [ ] Frontend can connect to ws://localhost:8080/ws
- [ ] Receives messages on piece download
- [ ] Reconnects on disconnect
- [ ] Integration test validates message flow

**Estimated Time:** 20 hours

---

#### ALPHA-15: Error Handling & User Feedback
**Owner:** Alex (Frontend Dev)  
**Story Points:** 5  
**Priority:** P1

**Description:** Graceful error handling with user-friendly messages.

**Tasks:**
- [ ] Create `ErrorBoundary` component
- [ ] Add toast notifications (react-toastify)
- [ ] Handle API errors (network, 4xx, 5xx)
- [ ] Display error messages in UI
- [ ] Add retry button for failed operations
- [ ] Log errors to console (or error tracking service)
- [ ] Write error simulation tests

**Acceptance Criteria:**
- [ ] All API errors show toast notification
- [ ] ErrorBoundary catches React errors
- [ ] User sees actionable error messages
- [ ] Retry button works for recoverable errors

**Estimated Time:** 20 hours

---

### 5.4 Sprint 4: Testing & Deployment (Weeks 7-8)

#### ALPHA-16: E2E Test Suite
**Owner:** Alex & Jamie (Frontend Devs)  
**Story Points:** 8  
**Priority:** P0

**Description:** End-to-end tests with Cypress.

**Tasks:**
- [ ] Set up Cypress with TypeScript
- [ ] Write test: Add torrent → Select file → Play video
- [ ] Write test: Seek to position → Verify playback
- [ ] Write test: Error handling (invalid magnet)
- [ ] Write test: Metrics dashboard updates
- [ ] Mock backend responses for predictable tests
- [ ] Run in CI pipeline
- [ ] Generate test report

**Acceptance Criteria:**
- [ ] 4 E2E tests pass consistently
- [ ] Tests run in CI on every PR
- [ ] Test report generated as artifact
- [ ] Tests complete in < 5 minutes

**Estimated Time:** 32 hours

---

#### ALPHA-17: API Documentation
**Owner:** Chris (Backend Dev)  
**Story Points:** 3  
**Priority:** P1

**Description:** Complete OpenAPI specification.

**Tasks:**
- [ ] Review all endpoints for Swagger annotations
- [ ] Add request/response examples
- [ ] Document error codes
- [ ] Generate Swagger UI
- [ ] Export OpenAPI JSON
- [ ] Write API guide in docs/api/README.md

**Acceptance Criteria:**
- [ ] Swagger UI shows all endpoints
- [ ] Each endpoint has description and examples
- [ ] Error codes documented
- [ ] OpenAPI JSON validates with Swagger Editor

**Estimated Time:** 12 hours

---

#### ALPHA-18: Frontend Build & Deployment
**Owner:** Jamie (Frontend Dev)  
**Story Points:** 3  
**Priority:** P0

**Description:** Production build and Docker image.

**Tasks:**
- [ ] Create production build script (npm run build)
- [ ] Optimize bundle size (code splitting, lazy loading)
- [ ] Create Dockerfile for frontend (nginx)
- [ ] Add build to docker-compose.yml
- [ ] Configure environment variables
- [ ] Write deployment guide

**Acceptance Criteria:**
- [ ] Production build succeeds
- [ ] Bundle size < 1MB (gzipped)
- [ ] Docker image builds and runs
- [ ] Frontend accessible in Docker Compose

**Estimated Time:** 12 hours

---

#### ALPHA-19: Load Testing
**Owner:** Sarah (Team Lead)  
**Story Points:** 5  
**Priority:** P1

**Description:** Validate system under load.

**Tasks:**
- [ ] Write load test with k6 (10 concurrent streams)
- [ ] Measure API response times
- [ ] Measure WebSocket latency
- [ ] Identify bottlenecks
- [ ] Optimize slow endpoints
- [ ] Document test results

**Acceptance Criteria:**
- [ ] Load test runs successfully
- [ ] 95th percentile API latency < 100ms
- [ ] No errors under 10 concurrent users
- [ ] Results documented in docs/performance.md

**Estimated Time:** 20 hours

---

#### ALPHA-20: User Documentation
**Owner:** Alex & Jamie (Frontend Devs)  
**Story Points:** 3  
**Priority:** P2

**Description:** Write user-facing guide.

**Tasks:**
- [ ] Create docs/user-guide.md
- [ ] Add screenshots of each step
- [ ] Write troubleshooting section
- [ ] Document keyboard shortcuts
- [ ] Add FAQ section

**Acceptance Criteria:**
- [ ] User guide covers complete flow
- [ ] Screenshots included
- [ ] Troubleshooting has 5+ common issues

**Estimated Time:** 12 hours

---

## 6. Team Beta Tasks

### 6.1 Sprint 1: Foundation (Weeks 1-2)

#### BETA-1: Project Setup
**Owner:** Morgan (Team Lead)  
**Story Points:** 2  
**Priority:** P0

**Description:** Initialize Go project with structure.

**Tasks:**
- [ ] Create Go module (go mod init)
- [ ] Set up directory structure (cmd, internal, pkg)
- [ ] Install libtorrent Go bindings
- [ ] Add Makefile for common tasks
- [ ] Set up gofmt and golangci-lint
- [ ] Create Dockerfile for sidecar
- [ ] Add Docker Compose service definition

**Acceptance Criteria:**
- [ ] `go build` compiles successfully
- [ ] golangci-lint passes
- [ ] Docker image builds
- [ ] Sidecar runs in Docker Compose

**Estimated Time:** 8 hours

---

#### BETA-2: libtorrent Integration
**Owner:** Taylor (Backend Dev)  
**Story Points:** 8  
**Priority:** P0

**Description:** Integrate libtorrent for torrent handling.

**Tasks:**
- [ ] Create `TorrentEngine` struct
- [ ] Initialize libtorrent session
- [ ] Configure session settings (DHT, ports, peers)
- [ ] Implement AddTorrent from magnet link
- [ ] Wait for metadata download
- [ ] Extract file list from torrent info
- [ ] Handle alerts (piece finished, torrent finished)
- [ ] Write unit tests with mock session

**Acceptance Criteria:**
- [ ] Can add torrent via magnet link
- [ ] Retrieves metadata within 30 seconds
- [ ] Returns list of files with metadata
- [ ] Unit tests cover happy path and errors

**Estimated Time:** 32 hours

---

#### BETA-3: gRPC Service Definition
**Owner:** Jordan (Backend Dev)  
**Story Points:** 3  
**Priority:** P0

**Description:** Define gRPC service contract.

**Tasks:**
- [ ] Create proto/torrent.proto file
- [ ] Define AddTorrent RPC
- [ ] Define GetStatus RPC
- [ ] Define ReadData streaming RPC
- [ ] Define PrioritizePieces RPC
- [ ] Define GetMetrics RPC
- [ ] Generate Go code with protoc
- [ ] Document proto file with comments

**Acceptance Criteria:**
- [ ] Proto file compiles without errors
- [ ] Go stubs generated
- [ ] All RPCs documented
- [ ] Proto file committed to repo

**Estimated Time:** 12 hours

---

#### BETA-4: gRPC Server Implementation
**Owner:** Jordan (Backend Dev)  
**Story Points:** 5  
**Priority:** P0

**Description:** Implement gRPC server skeleton.

**Tasks:**
- [ ] Create `GrpcServer` struct
- [ ] Implement AddTorrent handler
- [ ] Implement GetStatus handler (stub for now)
- [ ] Implement ReadData handler (stub for now)
- [ ] Start server on port 50051
- [ ] Add graceful shutdown
- [ ] Write integration test with gRPC client
- [ ] Add logging for all RPC calls

**Acceptance Criteria:**
- [ ] Server starts and listens on port 50051
- [ ] AddTorrent accepts requests
- [ ] Logs RPC calls to stdout
- [ ] Shuts down gracefully on SIGTERM
- [ ] Integration test connects and calls RPC

**Estimated Time:** 20 hours

---

#### BETA-5: Docker Compose Configuration
**Owner:** Riley (DevOps Engineer)  
**Story Points:** 3  
**Priority:** P0

**Description:** Set up local development environment.

**Tasks:**
- [ ] Create docker-compose.yml
- [ ] Add services: frontend, backend, sidecar
- [ ] Configure networks and volumes
- [ ] Add health checks for each service
- [ ] Set up depends_on ordering
- [ ] Add Prometheus and Grafana services
- [ ] Write docker-compose-dev.yml for development mode
- [ ] Document usage in README

**Acceptance Criteria:**
- [ ] `docker-compose up` starts all services
- [ ] Services can communicate (backend → sidecar)
- [ ] Volumes persist data correctly
- [ ] Health checks work
- [ ] Prometheus accessible at :9090

**Estimated Time:** 12 hours

---

### 6.2 Sprint 2: Integration (Weeks 3-4)

#### BETA-6: Piece Prioritization Logic
**Owner:** Taylor (Backend Dev)  
**Story Points:** 5  
**Priority:** P0

**Description:** Implement sequential + on-demand piece downloading.

**Tasks:**
- [ ] Create `PiecePrioritizer` class
- [ ] Implement sequential mode (download pieces 0, 1, 2, ...)
- [ ] Implement on-demand mode (prioritize requested range)
- [ ] Use libtorrent piece_priority API
- [ ] Maintain 10MB buffer ahead of current position
- [ ] Deprioritize far-away pieces
- [ ] Write unit tests for prioritization logic

**Acceptance Criteria:**
- [ ] Downloads pieces sequentially by default
- [ ] Can prioritize specific byte range
- [ ] Maintains buffer of next 10MB
- [ ] Unit tests cover all modes

**Estimated Time:** 20 hours

---

#### BETA-7: ReadData Streaming Implementation
**Owner:** Jordan (Backend Dev)  
**Story Points:** 8  
**Priority:** P0

**Description:** Stream data chunks to backend via gRPC.

**Tasks:**
- [ ] Implement ReadData streaming RPC
- [ ] Accept startByte, endByte, fileIndex in request
- [ ] Read data from libtorrent file storage
- [ ] Stream chunks (1MB each) back to client
- [ ] Handle pieces not yet downloaded (wait or error)
- [ ] Add timeout (30 seconds)
- [ ] Write integration test with mock client

**Acceptance Criteria:**
- [ ] Streams data in 1MB chunks
- [ ] Returns error if pieces unavailable after timeout
- [ ] Client receives all requested bytes
- [ ] Integration test validates streaming

**Estimated Time:** 32 hours

---

#### BETA-8: LRU Cache Foundation
**Owner:** Taylor (Backend Dev)  
**Story Points:** 5  
**Priority:** P1

**Description:** Build LRU cache for downloaded pieces.

**Tasks:**
- [ ] Create `LRUCache` struct with map + doubly-linked list
- [ ] Implement Put(key, value, metadata)
- [ ] Implement Get(key) → (value, found)
- [ ] Implement eviction when size exceeds limit
- [ ] Track cache statistics (hits, misses, evictions)
- [ ] Write unit tests for eviction logic

**Acceptance Criteria:**
- [ ] Cache stores up to 10GB
- [ ] LRU eviction works correctly
- [ ] Get returns cached data
- [ ] Stats tracked accurately
- [ ] Unit tests cover all operations

**Estimated Time:** 20 hours

---

#### BETA-9: Cache Integration with ReadData
**Owner:** Jordan (Backend Dev)  
**Story Points:** 5  
**Priority:** P1

**Description:** Check cache before reading from torrent.

**Tasks:**
- [ ] Modify ReadData to check cache first
- [ ] Generate cache key from (torrentId, fileIndex, pieceIndex)
- [ ] Return cached data if available
- [ ] Read from torrent and cache on miss
- [ ] Update cache stats
- [ ] Write integration test

**Acceptance Criteria:**
- [ ] Cache checked before torrent read
- [ ] Cache hit avoids torrent I/O
- [ ] Cache populated on miss
- [ ] Integration test validates caching

**Estimated Time:** 20 hours

---

#### BETA-10: GetStatus Implementation
**Owner:** Jordan (Backend Dev)  
**Story Points:** 3  
**Priority:** P1

**Description:** Return torrent status (progress, peers, speed).

**Tasks:**
- [ ] Implement GetStatus RPC
- [ ] Query libtorrent for torrent status
- [ ] Return: state, progress, downloadRate, uploadRate, numPeers, ETA
- [ ] Handle torrent not found
- [ ] Write unit test

**Acceptance Criteria:**
- [ ] Returns accurate status
- [ ] Updates in real-time
- [ ] Returns error for unknown torrent
- [ ] Unit test validates response

**Estimated Time:** 12 hours

---

### 6.3 Sprint 3: Polish (Weeks 5-6)

#### BETA-11: PrioritizePieces Implementation
**Owner:** Taylor (Backend Dev)  
**Story Points:** 3  
**Priority:** P0

**Description:** RPC to prioritize byte range for seeking.

**Tasks:**
- [ ] Implement PrioritizePieces RPC
- [ ] Accept torrentId, fileIndex, startByte, endByte
- [ ] Calculate affected pieces
- [ ] Set piece priority to HIGH
- [ ] Estimate wait time based on current speed
- [ ] Write unit test

**Acceptance Criteria:**
- [ ] Prioritizes correct pieces
- [ ] Returns ETA
- [ ] Unit test validates logic

**Estimated Time:** 12 hours

---

#### BETA-12: Prometheus Metrics
**Owner:** Riley (DevOps Engineer)  
**Story Points:** 5  
**Priority:** P1

**Description:** Expose metrics for monitoring.

**Tasks:**
- [ ] Add Prometheus Go client library
- [ ] Create `/metrics` HTTP endpoint
- [ ] Track metrics: download_rate, upload_rate, peers, cache_hit_ratio, cache_size
- [ ] Register metrics collectors
- [ ] Configure Prometheus to scrape sidecar
- [ ] Create Grafana dashboard
- [ ] Write guide for adding new metrics

**Acceptance Criteria:**
- [ ] Metrics exposed at :50051/metrics
- [ ] Prometheus scrapes metrics
- [ ] Grafana dashboard visualizes data
- [ ] Guide documented

**Estimated Time:** 20 hours

---

#### BETA-13: GetMetrics Implementation
**Owner:** Jordan (Backend Dev)  
**Story Points:** 3  
**Priority:** P1

**Description:** gRPC endpoint for metrics.

**Tasks:**
- [ ] Implement GetMetrics RPC
- [ ] Return: downloadRate, uploadRate, peers, cacheSize, cacheHitRatio
- [ ] Calculate cache hit ratio from stats
- [ ] Write unit test

**Acceptance Criteria:**
- [ ] Returns all 5 metrics
- [ ] Cache hit ratio calculated correctly
- [ ] Unit test validates response

**Estimated Time:** 12 hours

---

#### BETA-14: Health Check Endpoint
**Owner:** Riley (DevOps Engineer)  
**Story Points:** 2  
**Priority:** P1

**Description:** gRPC health check for Docker.

**Tasks:**
- [ ] Implement gRPC Health Check protocol
- [ ] Return SERVING if libtorrent session alive
- [ ] Add to Docker healthcheck
- [ ] Write integration test

**Acceptance Criteria:**
- [ ] Health check returns SERVING when ready
- [ ] Docker uses health check
- [ ] Integration test validates

**Estimated Time:** 8 hours

---

#### BETA-15: Error Handling & Logging
**Owner:** Morgan (Team Lead)  
**Story Points:** 5  
**Priority:** P1

**Description:** Comprehensive error handling.

**Tasks:**
- [ ] Add structured logging (zerolog)
- [ ] Log all gRPC calls with trace IDs
- [ ] Handle libtorrent errors gracefully
- [ ] Return gRPC status codes correctly (NotFound, InvalidArgument, Unavailable)
- [ ] Add panic recovery middleware
- [ ] Write error simulation tests

**Acceptance Criteria:**
- [ ] All errors logged with context
- [ ] gRPC errors returned properly
- [ ] Panics caught and logged
- [ ] Error tests pass

**Estimated Time:** 20 hours

---

#### BETA-16: Cache Persistence
**Owner:** Taylor (Backend Dev)  
**Story Points:** 5  
**Priority:** P2

**Description:** Persist cache across restarts.

**Tasks:**
- [ ] Save cache metadata to disk (JSON)
- [ ] Load cache on startup
- [ ] Validate cached pieces still exist
- [ ] Evict stale entries
- [ ] Write integration test

**Acceptance Criteria:**
- [ ] Cache survives restart
- [ ] Stale entries removed
- [ ] Integration test validates persistence

**Estimated Time:** 20 hours

---

### 6.4 Sprint 4: Testing & Deployment (Weeks 7-8)

#### BETA-17: Integration Tests
**Owner:** Jordan (Backend Dev)  
**Story Points:** 8  
**Priority:** P0

**Description:** Full sidecar integration tests.

**Tasks:**
- [ ] Write test: Add torrent → Wait for metadata → Get file list
- [ ] Write test: Read data → Verify cache hit
- [ ] Write test: Prioritize pieces → Verify priority set
- [ ] Write test: Concurrent requests (10 streams)
- [ ] Run tests in CI
- [ ] Generate coverage report

**Acceptance Criteria:**
- [ ] 4 integration tests pass
- [ ] Tests run in CI on every PR
- [ ] Coverage > 80%
- [ ] Tests complete in < 10 minutes

**Estimated Time:** 32 hours

---

#### BETA-18: Performance Optimization
**Owner:** Taylor & Jordan (Backend Devs)  
**Story Points:** 5  
**Priority:** P1

**Description:** Optimize bottlenecks.

**Tasks:**
- [ ] Profile with pprof
- [ ] Optimize cache lookups (hash map)
- [ ] Reduce gRPC serialization overhead
- [ ] Tune libtorrent settings (max peers, buffer size)
- [ ] Benchmark before/after
- [ ] Document findings

**Acceptance Criteria:**
- [ ] 20% improvement in throughput
- [ ] Profiling results documented
- [ ] Benchmarks show improvement

**Estimated Time:** 20 hours

---

#### BETA-19: Production Docker Image
**Owner:** Riley (DevOps Engineer)  
**Story Points:** 3  
**Priority:** P0

**Description:** Optimized Docker image for production.

**Tasks:**
- [ ] Use multi-stage build
- [ ] Minimize image size (< 100MB)
- [ ] Run as non-root user
- [ ] Add security scanning (Trivy)
- [ ] Push to Docker registry
- [ ] Write deployment guide

**Acceptance Criteria:**
- [ ] Image size < 100MB
- [ ] No critical vulnerabilities
- [ ] Runs as non-root
- [ ] Pushed to registry

**Estimated Time:** 12 hours

---

#### BETA-20: Monitoring & Alerting
**Owner:** Riley (DevOps Engineer)  
**Story Points:** 5  
**Priority:** P1

**Description:** Set up alerts for production.

**Tasks:**
- [ ] Create Prometheus alerting rules
- [ ] Alert on: high error rate, low peers, cache full
- [ ] Configure Alertmanager
- [ ] Test alerts
- [ ] Document runbooks

**Acceptance Criteria:**
- [ ] 3 alerting rules configured
- [ ] Alerts trigger correctly in tests
- [ ] Runbooks written for each alert

**Estimated Time:** 20 hours

---

## 7. Integration Points

### 7.1 API Contracts

**Backend → Sidecar (gRPC):**
```protobuf
service TorrentService {
  rpc AddTorrent(AddTorrentRequest) returns (AddTorrentResponse);
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  rpc ReadData(ReadDataRequest) returns (stream DataChunk);
  rpc PrioritizePieces(PrioritizeRequest) returns (PrioritizeResponse);
  rpc GetMetrics(GetMetricsRequest) returns (GetMetricsResponse);
}
```

**Frontend → Backend (REST):**
```
POST   /api/torrents
GET    /api/torrents/{id}/files
GET    /api/stream/{torrentId}/{fileIndex}
POST   /api/torrents/{id}/prioritize
GET    /api/metrics
WS     /ws
```

### 7.2 Integration Schedule

| Week | Integration Activity | Teams Involved |
|------|---------------------|----------------|
| Week 2 | API contract review | Alpha + Beta |
| Week 3 | gRPC connection testing | Alpha + Beta |
| Week 4 | End-to-end streaming test | Alpha + Beta |
| Week 5 | WebSocket integration | Alpha + Beta |
| Week 6 | Load testing | Alpha + Beta + QA |
| Week 7 | Production readiness review | All |

### 7.3 Integration Testing

**Scenario 1: Happy Path**
1. Frontend calls POST /api/torrents
2. Backend calls gRPC AddTorrent
3. Sidecar downloads metadata
4. Frontend calls GET /api/torrents/{id}/files
5. Backend calls gRPC GetStatus
6. Frontend plays video from /api/stream
7. Backend calls gRPC ReadData

**Scenario 2: Seeking**
1. User seeks in player
2. Frontend calls POST /api/torrents/{id}/prioritize
3. Backend calls gRPC PrioritizePieces
4. Sidecar prioritizes pieces
5. Frontend resumes playing

**Scenario 3: Error Handling**
1. Sidecar unreachable
2. Backend circuit breaker opens
3. Backend returns 503 to frontend
4. Frontend shows error toast

---

## 8. Dependencies

### 8.1 Critical Path

```
Milestone M1 (Week 2):
  ALPHA-1 (Setup) → ALPHA-2 (UI) → ALPHA-3 (API)
  BETA-1 (Setup) → BETA-2 (libtorrent) → BETA-4 (gRPC)

Milestone M2 (Week 4):
  ALPHA-6 (Player) → ALPHA-7 (Streaming) → ALPHA-8 (gRPC Client)
  BETA-6 (Prioritization) → BETA-7 (ReadData)

Milestone M3 (Week 6):
  ALPHA-10 (Seek) → ALPHA-11 (Prioritize API)
  BETA-11 (PrioritizePieces)

Milestone M4 (Week 8):
  ALPHA-16 (E2E) ← All features complete
  BETA-17 (Integration) ← All features complete
```

### 8.2 Blocking Issues

| Blocker | Impact | Mitigation |
|---------|--------|-----------|
| libtorrent installation fails | BETA-2 delayed | Use Docker image with prebuilt libtorrent |
| gRPC proto mismatch | Integration broken | Version control proto files, generate in CI |
| Sidecar crashes | All streaming fails | Add health checks, auto-restart |
| Cache fills up | Performance degrades | Implement LRU eviction early (BETA-8) |

---

## 9. Risk Management

### 9.1 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| libtorrent bindings incompatible | Medium | High | Test early, have fallback (torrent-client-go) |
| HTTP 206 not supported in browsers | Low | High | Test with multiple browsers (Chrome, Firefox, Safari) |
| gRPC connection instability | Medium | Medium | Implement retry + circuit breaker (ALPHA-8) |
| LRU cache memory leak | Low | High | Load test early, profile memory usage |
| Slow torrents (< 1 Mbps) | High | Medium | Set timeout, show user-friendly message |

### 9.2 Schedule Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| Sprint overcommitment | Medium | High | Buffer 20% of capacity, adjust mid-sprint |
| Dependencies block progress | Medium | High | Daily standup to identify blockers early |
| Team member unavailable | Low | Medium | Cross-train on critical tasks, pair programming |
| Requirements change | Low | High | Lock MVP scope, defer new features to v1.1 |

### 9.3 Contingency Plans

**If Sprint 2 delayed by 1 week:**
- Descope WebSocket (ALPHA-14) → Use polling temporarily
- Descope cache persistence (BETA-16) → Fresh cache on restart

**If integration fails in Week 4:**
- Allocate full week 5 to integration fixes
- Defer metrics dashboard (ALPHA-12) to post-MVP

**If load testing reveals bottleneck:**
- Performance sprint in Week 7
- Optimize backend streaming (ALPHA-7) or sidecar ReadData (BETA-7)

---

## 10. Testing Strategy

### 10.1 Test Pyramid

```
         E2E (10%)
       /           \
      /  Integration \
     /     (30%)      \
    /___________________\
         Unit (60%)
```

**Unit Tests:**
- All services, controllers, components
- Target: 80% coverage
- Run on every commit

**Integration Tests:**
- Backend ↔ Sidecar gRPC
- Frontend ↔ Backend API
- Target: Cover all happy paths + 3 error cases per feature
- Run on every PR

**E2E Tests:**
- Full user flows (add torrent → play video)
- Target: 4 critical paths
- Run on merge to main

### 10.2 Test Matrix

| Component | Unit | Integration | E2E | Load |
|-----------|------|-------------|-----|------|
| Frontend | Jest + RTL | Cypress | Cypress | k6 |
| Backend | JUnit | Spring Boot Test | - | JMeter |
| Sidecar | Go test | Testcontainers | - | k6 |

### 10.3 Test Data

**Sample Torrents:**
- Small (100MB, 1 file): For fast tests
- Medium (1GB, 5 files): For realistic tests
- Large (10GB, 20 files): For stress tests

**Mock Data:**
- Magnet links in `tests/fixtures/magnets.json`
- Fake torrent metadata in `tests/fixtures/torrents/`

---

## 11. Acceptance Criteria

### 11.1 Functional Requirements

- [ ] User can add torrent via magnet link
- [ ] User can select video file from list
- [ ] Video plays within 10 seconds
- [ ] User can seek forward and backward
- [ ] Metrics dashboard shows real-time data
- [ ] Errors display user-friendly messages
- [ ] Cache respects 10GB limit

### 11.2 Non-Functional Requirements

- [ ] API response time < 100ms (95th percentile)
- [ ] Startup time < 10 seconds (metadata download)
- [ ] Seek latency < 3 seconds
- [ ] Code coverage > 80%
- [ ] No P0/P1 bugs
- [ ] Documentation complete

### 11.3 Release Checklist

**Before Staging:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] E2E tests pass
- [ ] Load test with 10 concurrent users succeeds
- [ ] Docker Compose deployment works
- [ ] Logs reviewed (no errors/warnings)

**Before Production:**
- [ ] Security scan passes (no critical vulnerabilities)
- [ ] Performance benchmarks meet targets
- [ ] Monitoring dashboards created
- [ ] Alerting rules configured
- [ ] Rollback plan documented
- [ ] Post-launch support plan ready

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **Story Points** | Relative estimation of task complexity (Fibonacci: 1, 2, 3, 5, 8) |
| **P0/P1/P2** | Priority levels (0=critical, 1=high, 2=medium) |
| **gRPC** | Google Remote Procedure Call framework |
| **LRU** | Least Recently Used cache eviction policy |
| **HTTP 206** | Partial Content response for byte-range requests |
| **E2E** | End-to-End testing |

---

## Appendix B: Task Summary by Team

### Team Alpha (Frontend & Backend)

| Sprint | Tasks | Story Points | Hours |
|--------|-------|--------------|-------|
| Sprint 1 | ALPHA-1 to ALPHA-5 | 16 | 64 |
| Sprint 2 | ALPHA-6 to ALPHA-9 | 23 | 92 |
| Sprint 3 | ALPHA-10 to ALPHA-15 | 26 | 104 |
| Sprint 4 | ALPHA-16 to ALPHA-20 | 22 | 88 |
| **Total** | **20 tasks** | **87** | **348** |

### Team Beta (Sidecar & Infrastructure)

| Sprint | Tasks | Story Points | Hours |
|--------|-------|--------------|-------|
| Sprint 1 | BETA-1 to BETA-5 | 21 | 84 |
| Sprint 2 | BETA-6 to BETA-10 | 26 | 104 |
| Sprint 3 | BETA-11 to BETA-16 | 23 | 92 |
| Sprint 4 | BETA-17 to BETA-20 | 21 | 84 |
| **Total** | **20 tasks** | **91** | **364** |

---

## Appendix C: Communication Plan

### Daily Standups
- **Format:** Async via Slack
- **Time:** Before 10 AM
- **Template:** ✅ Yesterday, 🎯 Today, 🚧 Blockers

### Weekly Sync
- **When:** Every Monday 10 AM
- **Duration:** 30 minutes
- **Agenda:**
  - Sprint progress review
  - Blocker discussion
  - Integration point coordination

### Bi-weekly Sprint Ceremonies
- **Sprint Planning:** Wednesday 2 PM (2 hours)
- **Sprint Review:** Friday 3 PM (1 hour)
- **Retrospective:** Friday 4 PM (1 hour)

---

**Document Status:** Approved for Development  
**Last Updated:** January 27, 2026  
**Version:** 1.0  
**Approvals:**
- Tech Lead: ✅ Dana
- Team Alpha Lead: ✅ Sarah
- Team Beta Lead: ✅ Morgan
- Product Owner: ✅ Avery
