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

### Run All Tests

```bash
# All components
./scripts/run-tests.sh

# Individual components
cd backend && mvn test
cd sidecar && go test ./...
cd frontend && npm test
```

### Test Coverage

```bash
# Generate coverage report
./scripts/coverage.sh

# View coverage in browser
open coverage/index.html
```

### Integration Tests

```bash
# Requires Docker
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

### Load Testing

```bash
# Install k6
brew install k6

# Run load test
k6 run tests/load/streaming-test.js
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
