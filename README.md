# MagnetPlay

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)]()
[![Coverage](https://img.shields.io/badge/coverage-82%25-green)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Java](https://img.shields.io/badge/java-17-orange)]()
[![Go](https://img.shields.io/badge/go-1.21-blue)]()
[![React](https://img.shields.io/badge/react-18-blue)]()

A high-performance peer-to-peer video streaming system that enables immediate playback of torrent content without requiring complete file downloads. Think Stremio, but built from scratch with modern technologies.

---

## 🎯 Quick Start

Get up and running in 5 minutes:

```bash
# Clone the repository
git clone https://github.com/your-org/streaming-system.git
cd streaming-system

# Start all services
docker-compose up -d

# Open your browser
open http://localhost:3000
```

That's it! The application is ready to stream torrents.

---

## 📚 Table of Contents

- [Features](#-features)
- [Architecture Overview](#-architecture-overview)
- [Prerequisites](#-prerequisites)
- [Installation](#-installation)
- [Development Setup](#-development-setup)
- [Project Structure](#-project-structure)
- [Configuration](#-configuration)
- [Running Tests](#-running-tests)
- [Deployment](#-deployment)
- [Contributing](#-contributing)
- [Troubleshooting](#-troubleshooting)
- [FAQ](#-faq)
- [Resources](#-resources)
- [Support](#-support)

---

## ✨ Features

### Core Capabilities

- 🎬 **Instant Playback:** Start streaming within 10 seconds of selecting a torrent
- ⏩ **Smart Seeking:** Jump to any position with intelligent piece prioritization
- 💾 **Storage Efficient:** Automatic cache management with 10GB limit
- 🔄 **Progressive Download:** Continue downloading while streaming
- 📊 **Real-time Metrics:** Live download statistics, peer count, and bandwidth usage
- 🛡️ **Resilient:** Handles network failures and slow peers gracefully
- 📱 **Responsive Design:** Works seamlessly on desktop, tablet, and mobile

### Technical Highlights

- **Multi-protocol Support:** BitTorrent DHT + Tracker-based peer discovery
- **HTTP Range Requests:** Efficient byte-range serving (HTTP 206)
- **LRU Caching:** Least Recently Used eviction for optimal storage
- **WebSocket Updates:** Real-time progress notifications
- **Health Monitoring:** Prometheus metrics + Grafana dashboards

---

## 🏗️ Architecture Overview

```
┌─────────────┐
│   Browser   │ ← React SPA with Video.js player
└──────┬──────┘
       │ HTTP/WebSocket
┌──────▼──────┐
│   Backend   │ ← Spring Boot (Java 17) - REST APIs
└──────┬──────┘
       │ gRPC
┌──────▼──────┐
│   Sidecar   │ ← Go + libtorrent - Torrent engine
└──────┬──────┘
       │ BitTorrent Protocol
┌──────▼──────┐
│ Peer Network│ ← DHT + Trackers
└─────────────┘
```

**Key Design Principles:**
- **Separation of Concerns:** Each component has a single responsibility
- **API-First:** Well-defined contracts between services
- **Observable:** Comprehensive logging and metrics
- **Fail-Fast:** Early detection and graceful degradation

For detailed architecture, see [DESIGN_DOCUMENT.md](./DESIGN_DOCUMENT.md).

---

## 🛠️ Prerequisites

Before you begin, ensure you have the following installed:

### Required

- **Docker:** 20.10+ ([Install Docker](https://docs.docker.com/get-docker/))
- **Docker Compose:** 2.0+ (included with Docker Desktop)
- **Git:** 2.30+ ([Install Git](https://git-scm.com/downloads))

### Optional (for local development)

- **Java:** OpenJDK 17+ ([Install Java](https://adoptium.net/))
- **Go:** 1.21+ ([Install Go](https://go.dev/doc/install))
- **Node.js:** 18+ ([Install Node](https://nodejs.org/))
- **Maven:** 3.8+ ([Install Maven](https://maven.apache.org/install.html))

### System Requirements

**Minimum:**
- CPU: 4 cores
- RAM: 16GB
- Disk: 50GB available space
- Network: 10 Mbps download speed

**Recommended:**
- CPU: 8 cores
- RAM: 32GB
- Disk: 500GB SSD
- Network: 100 Mbps download speed

---

## 📦 Installation

### Option 1: Docker Compose (Recommended)

The easiest way to get started:

```bash
# 1. Clone the repository
git clone https://github.com/your-org/streaming-system.git
cd streaming-system

# 2. Copy environment template
cp .env.example .env

# 3. Start all services
docker-compose up -d

# 4. Verify services are running
docker-compose ps

# 5. Check logs
docker-compose logs -f backend
```

**Services will be available at:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3001 (admin/admin)

### Option 2: Manual Setup

For development or customization:

```bash
# 1. Clone repository
git clone https://github.com/your-org/streaming-system.git
cd streaming-system

# 2. Build sidecar
cd sidecar
go build -o bin/sidecar cmd/main.go
cd ..

# 3. Build backend
cd backend
mvn clean package -DskipTests
cd ..

# 4. Build frontend
cd frontend
npm install
npm run build
cd ..

# 5. Start services manually
./scripts/start-local.sh
```

---

## 💻 Development Setup

### First-Time Setup

```bash
# 1. Fork the repository (on GitHub)

# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/streaming-system.git
cd streaming-system

# 3. Add upstream remote
git remote add upstream https://github.com/your-org/streaming-system.git

# 4. Install development tools
./scripts/install-dev-tools.sh

# 5. Set up pre-commit hooks
pre-commit install
```

### Build Go Proto Code
#### Prerequisite for Windows 
  - Install protoc `winget install Google.Protobuf`
  - Install Make.exe  `winget install ezwinports.make`

#### Build Project or go-proto files
- Run below command from MagnetPlay/backend
  - To generate go-proto file only `make -f MakeFile proto`.
  - To build go project `make -f MakeFile all` (proto + build target)
  
#### Manual apporach to run (preferred for debugging)
```bash
protoc --go_out=go-server --go_opt=paths=source_relative --go-grpc_out=go-server --go-grpc_opt=paths=source_relative proto/torrent.proto
```

### Running Individual Components

**Backend (Spring Boot):**
```bash
cd backend
mvn spring-boot:run
```

**Sidecar (Go):**
```bash
cd sidecar
go run cmd/main.go
```

**Frontend (React):**
```bash
cd frontend
npm start
```

### Hot Reload Development

All components support hot reload:

- **Frontend:** Webpack dev server (auto-reload)
- **Backend:** Spring Boot DevTools (auto-restart)
- **Sidecar:** Manual restart required (use `make watch`)

### IDE Setup

**IntelliJ IDEA (Backend):**
1. Open `backend/pom.xml` as a project
2. Enable annotation processing (Settings → Build → Compiler → Annotation Processors)
3. Install Lombok plugin

**VS Code (Frontend & Sidecar):**
1. Install recommended extensions (`.vscode/extensions.json`)
2. Enable format on save
3. Use workspace settings (`.vscode/settings.json`)

---

## 📁 Project Structure

```
streaming-system/
├── frontend/                 # React SPA
│   ├── src/
│   │   ├── components/       # Reusable UI components
│   │   ├── pages/            # Page-level components
│   │   ├── services/         # API clients
│   │   ├── hooks/            # Custom React hooks
│   │   └── utils/            # Helper functions
│   ├── public/               # Static assets
│   └── package.json
│
├── backend/                  # Spring Boot application
│   ├── src/main/java/com/example/streaming/
│   │   ├── controller/       # REST endpoints
│   │   ├── service/          # Business logic
│   │   ├── model/            # Data models
│   │   ├── config/           # Spring configuration
│   │   └── grpc/             # gRPC clients
│   ├── src/main/resources/
│   │   ├── application.yml   # Configuration
│   │   └── proto/            # gRPC definitions
│   └── pom.xml
│
├── sidecar/                  # Go torrent engine
│   ├── cmd/main.go           # Entry point
│   ├── internal/
│   │   ├── torrent/          # Torrent management
│   │   ├── cache/            # LRU cache
│   │   ├── grpc/             # gRPC server
│   │   └── metrics/          # Prometheus metrics
│   ├── pkg/                  # Shared libraries
│   └── go.mod
│
├── docs/                     # Documentation
│   ├── DESIGN_DOCUMENT.md    # Architecture details
│   ├── CONSTITUTION.md       # Project governance
│   ├── MVP_SPECIFICATION.md  # MVP requirements
│   └── api/                  # API documentation
│
├── scripts/                  # Automation scripts
│   ├── install-dev-tools.sh
│   ├── run-tests.sh
│   └── deploy.sh
│
├── docker-compose.yml        # Local development setup
├── .github/                  # GitHub workflows
└── README.md                 # This file
```

---

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the project root:

```bash
# Backend
SPRING_PROFILES_ACTIVE=dev
GRPC_SIDECAR_HOST=localhost
GRPC_SIDECAR_PORT=50051
STREAMING_BUFFER_SIZE=10485760

# Sidecar
CACHE_MAX_SIZE=10737418240
TORRENT_DOWNLOAD_PATH=/data/torrents
TORRENT_CACHE_PATH=/data/cache

# Frontend
REACT_APP_API_URL=http://localhost:8080
REACT_APP_WS_URL=ws://localhost:8080/ws

# Monitoring
PROMETHEUS_PORT=9090
GRAFANA_PORT=3001
```

### Backend Configuration

Edit `backend/src/main/resources/application.yml`:

```yaml
server:
  port: 8080

streaming:
  buffer-size: 10485760  # 10MB
  timeout-seconds: 30

grpc:
  client:
    sidecar:
      address: localhost:50051
      negotiation-type: plaintext
```

### Sidecar Configuration

Edit `sidecar/config.yaml`:

```yaml
grpc:
  port: 50051

torrent:
  download_path: /data/torrents
  cache_path: /data/cache
  max_cache_size: 10737418240  # 10GB

libtorrent:
  listen_ports: [6881, 6889]
  dht_enabled: true
```

---

## 🧪 Running Tests

### Go Sidecar

```bash
cd backend/go-server

# All tests
go test ./...

# Specific package (verbose)
go test ./internal/torrent/... -v

# Single test
go test ./internal/torrent/... -run TestSpeedTracker -v

# Build verify (no tests)
go build ./...
```

### Spring Boot

> **Note:** `./mvnw` wrapper may be missing. Use system `mvn` directly.

```bash
cd backend/mp-spring

# All tests
mvn test -q 2>&1 | tail -20

# Single test class
mvn test -Dtest=SessionManagerTest -q

# Single test class (verbose, shows failures)
mvn test -Dtest=TorrentStatsControllerTest 2>&1 | grep -E "Tests run|BUILD|ERROR|FAIL"

# Compile only (no tests — fast proto check)
mvn compile -q
```

### Frontend (TypeScript)

```bash
cd frontend

# Type-check without building
npx tsc --noEmit

# Dev server (visual verification)
npm run dev
# → Open http://localhost:5173
```

### Integration Smoke Test (manual — requires all services running)

**Terminal 1 — Go sidecar:**
```bash
cd backend/go-server && go run main.go
# Expected: gRPC server listening on :50051
```

**Terminal 2 — Spring Boot:**
```bash
cd backend/mp-spring && mvn spring-boot:run -q
# Expected: Started on port 8080
```

**Terminal 3 — verify endpoints:**
```bash
# Add torrent
curl -s -X POST http://localhost:8080/v1/torrent/add \
  -H "Content-Type: application/json" \
  -d '{"magnetUrl": "<magnet-url>"}' | jq .

# List torrents
curl -s http://localhost:8080/v1/torrent/list | jq .torrents[0].state

# Stats for a file
curl -s "http://localhost:8080/v1/torrent/stats/<infoHash>?fileId=<fileId>" | jq .

# Pause / resume
curl -s -X POST http://localhost:8080/v1/torrent/pause/<infoHash>
curl -s -X POST http://localhost:8080/v1/torrent/resume/<infoHash>

# Active sessions (while streaming)
curl -s "http://localhost:8080/v1/torrent/sessions?activeOnly=true" | jq .sessions[0]
```

---

## 🚀 Deployment

### Staging Deployment

```bash
# Build and deploy to staging
./scripts/deploy.sh staging

# Check deployment status
kubectl get pods -n streaming-staging
```

### Production Deployment

```bash
# Requires approval from Tech Lead
./scripts/deploy.sh production

# Monitor rollout
kubectl rollout status deployment/backend -n streaming-prod
```

### Docker Registry

```bash
# Tag images
docker tag streaming-backend:latest registry.example.com/streaming-backend:v1.2.3

# Push to registry
docker push registry.example.com/streaming-backend:v1.2.3
```

---

## 🤝 Contributing

We welcome contributions! Please follow these steps:

### 1. Pick an Issue

- Browse [open issues](https://github.com/your-org/streaming-system/issues)
- Look for `good first issue` labels for beginners
- Comment on the issue to claim it

### 2. Create a Branch

```bash
git checkout -b feature/add-subtitle-support
```

### 3. Make Changes

- Follow code style guidelines (see [CONSTITUTION.md](./docs/CONSTITUTION.md))
- Write tests for new functionality
- Update documentation

### 4. Commit Changes

Use conventional commit messages:

```bash
git commit -m "Feature(player): add subtitle support for .srt files"
```

### 5. Push and Create PR

```bash
git push origin feature/add-subtitle-support
```

Then open a pull request on GitHub.

### 6. Code Review

- Address reviewer feedback
- Ensure CI passes
- Get 2+ approvals

### Code Style

**Pre-commit Hooks:** Automatically format code

```bash
# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

**Linting:**
- Java: Google Java Format
- Go: gofmt + golangci-lint
- TypeScript: ESLint + Prettier

---

## 🔑 Development Best Practices

### Proto Contract Changes

Any change to `backend/proto/torrent.proto` requires regenerating stubs for both services:

```bash
# Go stubs (output to backend/go-server/proto/ — gitignored by design)
cd backend && make -f MakeFile proto

# Java stubs — auto-compiled on next Maven build
cd backend/mp-spring && mvn compile -q
```

**Rule:** Never edit generated `*.pb.go` or Java proto files by hand.

### God Nodes — High-Risk Files

Two files have 11+ edges in the knowledge graph. Extra care required:

| File | Risk | Rule |
|------|------|------|
| `backend/go-server/internal/torrent/repository.go` | Mutex errors break all torrent ops | Every new method must acquire `r.mu` (RLock for reads, Lock for writes) |
| `frontend/src/App.tsx` | Center of React component graph | Edit in a single pass when wiring multiple components — two separate touches cause merge conflicts and hook ordering bugs |

### Commit Discipline

- No commits mid-implementation for multi-task features
- Commit once after all tasks complete and tests pass
- Commit message format: `Feature(scope): description` (see git log for examples)

### Maven Wrapper

`./mvnw` may be missing on fresh clones. Use system `mvn`:

```bash
# If ./mvnw fails — install the wrapper
cd backend/mp-spring && mvn wrapper:wrapper
# Or just use: mvn <command>
```

### Go Proto Import Path

The generated proto package lives at `MagnetPlay/backend/proto` (module-relative). Import in Go:

```go
import pb "MagnetPlay/backend/proto"
```

The `backend/go-server/proto/` directory is gitignored — regenerate on each clone via `make -f MakeFile proto`.

### TorrentInfo Fields Are Unexported

`repository.go` uses unexported `torrent` and `files` fields. Access via accessor methods:

```go
info.Torrent()  // returns *torrent.Torrent
info.Files()    // returns map[string]*torrent.File
```

Use `repo.GetTorrentInfo(infoHash)` (returns `*TorrentInfo, error`) not `repo.GetTorrent()` (returns `*torrent.Torrent, bool`).

### Session Tracking Behavior

Each HTTP Range request = new streaming session. A single seek creates a new session row — this is correct. VideoJS makes multiple range requests for buffering. Do not conflate session count with viewer count.

---

## 🔧 Troubleshooting

### Common Issues

**Problem: Sidecar won't connect**

```bash
# Check if sidecar is running
docker-compose ps sidecar

# View logs
docker-compose logs sidecar

# Restart sidecar
docker-compose restart sidecar
```

**Problem: No peers found**

- Wait 5 minutes for DHT to bootstrap
- Check tracker whitelist in configuration
- Verify UDP ports 6881-6889 are open

**Problem: Playback stutters**

```bash
# Increase buffer size
export STREAMING_BUFFER_SIZE=20971520

# Check disk space
df -h /data/cache

# Monitor metrics
open http://localhost:3001  # Grafana
```

**Problem: Build fails**

```bash
# Clean and rebuild
docker-compose down -v
docker-compose build --no-cache
docker-compose up -d
```

### Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend

# Last 100 lines
docker-compose logs --tail=100 sidecar
```

### Health Checks

```bash
# Backend
curl http://localhost:8080/actuator/health

# Sidecar
grpcurl -plaintext localhost:50051 health.HealthCheck/Check
```

---

## ❓ FAQ

**Q: Can I use this for commercial purposes?**  
A: Yes, the project is MIT licensed. See [LICENSE](./LICENSE) for details.

**Q: How do I add a new torrent tracker?**  
A: Edit `sidecar/config.yaml` and add to `torrent.tracker_whitelist`.

**Q: Does this work with private trackers?**  
A: Not currently. Private tracker support is planned for v2.0.

**Q: How do I increase cache size?**  
A: Set `CACHE_MAX_SIZE` environment variable (in bytes).

**Q: Can I run this on a Raspberry Pi?**  
A: Possible but not recommended. Minimum 4 cores and 16GB RAM required.

**Q: Is VPN support built-in?**  
A: No, but you can route Docker containers through VPN. See [VPN Guide](./docs/vpn-setup.md).

---

## 📖 Resources

### Documentation

- [Design Document](./docs/DESIGN_DOCUMENT.md) - Complete architecture
- [Constitution](./docs/CONSTITUTION.md) - Project governance
- [MVP Specification](./docs/MVP_SPECIFICATION.md) - MVP requirements
- [API Documentation](./docs/api/README.md) - REST and gRPC APIs

### External Resources

- [libtorrent Documentation](https://www.libtorrent.org/reference.html)
- [Video.js Guide](https://videojs.com/guides/)
- [Spring Boot Reference](https://spring.io/projects/spring-boot)
- [gRPC Best Practices](https://grpc.io/docs/guides/)

### Community

- **GitHub Discussions:** [Ask questions](https://github.com/your-org/streaming-system/discussions)
- **Slack:** `#streaming-system` (for contributors)
- **Stack Overflow:** Tag `streaming-system`

---

## 🆘 Support

### Getting Help

1. **Check Documentation:** Most questions are answered in docs
2. **Search Issues:** Someone may have had the same problem
3. **GitHub Discussions:** Ask the community
4. **Open an Issue:** Report bugs or request features

### Reporting Bugs

Use our [bug report template](.github/ISSUE_TEMPLATE/bug_report.md):

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Add torrent with magnet link '...'
2. Click on file '...'
3. Seek to position '...'
4. See error

**Expected behavior**
What you expected to happen.

**Screenshots**
If applicable, add screenshots.

**Environment:**
- OS: [e.g., macOS 13.1]
- Browser: [e.g., Chrome 110]
- Version: [e.g., v1.2.3]
```

### Feature Requests

Use our [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) and include:
- Use case and user story
- Proposed solution
- Alternatives considered
- Additional context

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

---

## 🙏 Acknowledgments

Built with amazing open-source projects:
- [libtorrent](https://www.libtorrent.org/) - BitTorrent library
- [Video.js](https://videojs.com/) - HTML5 video player
- [Spring Boot](https://spring.io/projects/spring-boot) - Java framework
- [gRPC](https://grpc.io/) - RPC framework
- [Prometheus](https://prometheus.io/) - Monitoring system

Special thanks to the [Stremio](https://www.stremio.com/) team for inspiration.

---

## 🗺️ Roadmap

### Current Version: v1.0 (MVP)

- ✅ Basic torrent streaming
- ✅ HTTP 206 range requests
- ✅ LRU caching
- ✅ Real-time metrics

### Next Version: v1.1

- 🎯 Subtitle support (.srt, .vtt)
- 🎯 Playlist mode
- 🎯 Quality selection
- 🎯 Mobile app (React Native)

### Future Versions

- 🔮 Chromecast support
- 🔮 Transcoding (FFmpeg)
- 🔮 Multi-user accounts
- 🔮 Content discovery

See [ROADMAP.md](./docs/ROADMAP.md) for detailed plans.

---

## 📊 Project Stats

![GitHub stars](https://img.shields.io/github/stars/your-org/streaming-system)
![GitHub forks](https://img.shields.io/github/forks/your-org/streaming-system)
![GitHub issues](https://img.shields.io/github/issues/your-org/streaming-system)
![GitHub pull requests](https://img.shields.io/github/issues-pr/your-org/streaming-system)

---

**Happy Streaming! 🎬**

If you find this project useful, please consider giving it a ⭐️ on GitHub!

---

**Last Updated:** January 27, 2026  
**Maintainers:** [@tech-lead](https://github.com/tech-lead), [@team-alpha-lead](https://github.com/team-alpha-lead)  
**Status:** Active Development
