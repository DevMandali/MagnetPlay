# Prowlarr Search Integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a torrent search tab at Step 1 alongside the magnet-link input; powered by Prowlarr which the Go sidecar auto-downloads, spawns, and seeds with public indexers on first run.

**Architecture:** Go sidecar downloads Prowlarr binary from GitHub, starts it as a child process, reads the generated API key from `prowlarr-data/config.xml`, and seeds default public indexers on first launch. Spring Boot adds a thin `GET /v1/search?q=` endpoint that proxies Torznab XML from Prowlarr to JSON using Java's built-in DOM parser. Frontend adds a tab toggle at Step 1; selecting a search result auto-fills the magnet input.

**Tech Stack:** Go `net/http`, `os/exec`, `encoding/xml`, `archive/zip`; Spring WebFlux `WebClient` (already in pom.xml); Java DOM (`javax.xml`); React + TypeScript

---

## File Map

**Create (Go):**
- `backend/go-server/internal/prowlarr/downloader.go` — GitHub release fetch + zip extract
- `backend/go-server/internal/prowlarr/manager.go` — spawn, health-check, API key read, indexer seed

**Modify (Go):**
- `backend/go-server/config/config.go` — add `ProwlarrConfig` struct
- `backend/go-server/internal/grpc/server.go` — start/stop Prowlarr manager in lifecycle

**Create (Spring):**
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SearchResult.java`
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SearchResultsResponse.java`
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/ProwlarrClient.java`
- `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/SearchController.java`

**Modify (Spring):**
- `backend/mp-spring/src/main/resources/application.yml` — add prowlarr config block

**Create (Frontend):**
- `frontend/src/components/SearchPanel.tsx` — search UI with quality badges + result list

**Modify (Frontend):**
- `frontend/src/types/index.ts` — add `SearchResult`, `SearchResultsResponse`
- `frontend/src/App.tsx` — tab toggle at Step 1, wire result → magnet input

---

### Task 1: Go — Add ProwlarrConfig to config.go

**Files:**
- Modify: `backend/go-server/config/config.go`

- [ ] **Step 1: Update config.go**

```go
package config

import "time"

type ProwlarrConfig struct {
	DataDir      string
	BinDir       string
	Port         int
	SeedIndexers bool
}

type Config struct {
	GRPCPort        int
	DataDir         string
	MetadataTimeout time.Duration
	Prowlarr        ProwlarrConfig
}

func Default() Config {
	return Config{
		GRPCPort:        50051,
		DataDir:         "./downloads",
		MetadataTimeout: 60 * time.Second,
		Prowlarr: ProwlarrConfig{
			DataDir:      "./prowlarr-data",
			BinDir:       "./prowlarr",
			Port:         9696,
			SeedIndexers: true,
		},
	}
}
```

- [ ] **Step 2: Build check**

```bash
cd backend/go-server && go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/go-server/config/config.go
git commit -m "feat(prowlarr): add ProwlarrConfig to Go config"
```

---

### Task 2: Go — Prowlarr binary downloader

**Files:**
- Create: `backend/go-server/internal/prowlarr/downloader.go`

- [ ] **Step 1: Create downloader.go**

```go
package prowlarr

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const githubReleasesURL = "https://api.github.com/repos/Prowlarr/Prowlarr/releases/latest"

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func platformSuffix() string {
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("windows-core-%s.zip", arch)
	case "linux":
		return fmt.Sprintf("linux-core-%s.tar.gz", arch)
	case "darwin":
		return fmt.Sprintf("osx-core-%s.tar.gz", arch)
	}
	return ""
}

func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "Prowlarr.exe"
	}
	return "Prowlarr"
}

// FindBinary walks binDir looking for the Prowlarr executable (handles nested zip layout).
func FindBinary(binDir string) string {
	target := BinaryName()
	var found string
	filepath.WalkDir(binDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// EnsureBinary downloads and extracts Prowlarr if not already present in binDir.
func EnsureBinary(binDir string) error {
	if FindBinary(binDir) != "" {
		return nil
	}

	suffix := platformSuffix()
	if suffix == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch release info: %w", err)
	}

	var assetURL string
	for _, a := range release.Assets {
		if strings.HasSuffix(a.Name, suffix) {
			assetURL = a.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no asset found for suffix %q in release %s", suffix, release.TagName)
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "prowlarr-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	fmt.Printf("[prowlarr] downloading %s (%s)...\n", release.TagName, suffix)
	if err := downloadToFile(assetURL, tmpFile); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	tmpFile.Close()

	if strings.HasSuffix(assetURL, ".zip") {
		return extractZip(tmpFile.Name(), binDir)
	}
	// Linux/macOS tar.gz: requires archive/tar — add when targeting non-Windows
	return fmt.Errorf("tar.gz extraction not yet implemented; extract manually to %s", binDir)
}

func fetchLatestRelease() (*githubRelease, error) {
	req, _ := http.NewRequest("GET", githubReleasesURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MagnetPlay/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rel githubRelease
	return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}

func downloadToFile(url string, dest *os.File) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(dest, resp.Body)
	return err
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dest)+string(os.PathSeparator)) {
			continue // zip-slip guard
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: Build check**

```bash
cd backend/go-server && go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/go-server/internal/prowlarr/downloader.go
git commit -m "feat(prowlarr): binary downloader — GitHub releases, zip extract, zip-slip guard"
```

---

### Task 3: Go — Prowlarr process manager

**Files:**
- Create: `backend/go-server/internal/prowlarr/manager.go`

- [ ] **Step 1: Create manager.go**

```go
package prowlarr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

type Manager struct {
	binDir   string
	dataDir  string
	port     int
	seedOnce bool
	apiKey   string
	cmd      *exec.Cmd
}

func NewManager(binDir, dataDir string, port int, seedOnce bool) *Manager {
	return &Manager{binDir: binDir, dataDir: dataDir, port: port, seedOnce: seedOnce}
}

// Start downloads Prowlarr if needed, spawns it, waits for readiness, reads API key.
func (m *Manager) Start() error {
	if err := EnsureBinary(m.binDir); err != nil {
		return fmt.Errorf("ensure binary: %w", err)
	}

	binPath := FindBinary(m.binDir)
	if binPath == "" {
		return fmt.Errorf("prowlarr binary not found in %s after download", m.binDir)
	}

	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return err
	}

	m.cmd = exec.Command(binPath,
		"--data="+m.dataDir,
		fmt.Sprintf("--port=%d", m.port),
		"--nobrowser",
	)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start prowlarr: %w", err)
	}
	log.Printf("[prowlarr] started PID %d on port %d", m.cmd.Process.Pid, m.port)

	if err := m.waitReady(90 * time.Second); err != nil {
		m.Stop()
		return err
	}

	key, err := m.readAPIKey()
	if err != nil {
		return fmt.Errorf("read API key: %w", err)
	}
	m.apiKey = key
	log.Printf("[prowlarr] ready — API key acquired")

	if m.seedOnce {
		go m.seedDefaultIndexers()
	}
	return nil
}

func (m *Manager) Stop() {
	if m.cmd == nil || m.cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		m.cmd.Process.Kill()
	} else {
		m.cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { m.cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			m.cmd.Process.Kill()
		}
	}
	log.Printf("[prowlarr] stopped")
}

func (m *Manager) APIKey() string  { return m.apiKey }
func (m *Manager) BaseURL() string { return fmt.Sprintf("http://localhost:%d", m.port) }

func (m *Manager) waitReady(timeout time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d/ping", m.port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("prowlarr not ready after %s", timeout)
}

var apiKeyRe = regexp.MustCompile(`<ApiKey>([^<]+)</ApiKey>`)

func (m *Manager) readAPIKey() (string, error) {
	configPath := filepath.Join(m.dataDir, "config.xml")
	for i := 0; i < 15; i++ {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if match := apiKeyRe.FindSubmatch(data); len(match) == 2 {
				return string(match[1]), nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("API key not found in %s", configPath)
}

// ── Indexer seeding ───────────────────────────────────────────────────────────

type indexerField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type indexerPayload struct {
	Name                    string         `json:"name"`
	EnableRss               bool           `json:"enableRss"`
	EnableAutomaticSearch   bool           `json:"enableAutomaticSearch"`
	EnableInteractiveSearch bool           `json:"enableInteractiveSearch"`
	Implementation          string         `json:"implementation"`
	ConfigContract          string         `json:"configContract"`
	Protocol                string         `json:"protocol"`
	Privacy                 string         `json:"privacy"`
	Fields                  []indexerField `json:"fields"`
	Tags                    []int          `json:"tags"`
}

func cardigannIndexer(name, defFile string) indexerPayload {
	return indexerPayload{
		Name: name, EnableRss: false,
		EnableAutomaticSearch: true, EnableInteractiveSearch: true,
		Implementation: "Cardigann", ConfigContract: "CardigannSettings",
		Protocol: "torrent", Privacy: "public",
		Fields: []indexerField{{Name: "definitionFile", Value: defFile}},
		Tags:   []int{},
	}
}

var defaultIndexers = []indexerPayload{
	cardigannIndexer("1337x", "1337x"),
	cardigannIndexer("YTS", "yts"),
	cardigannIndexer("EZTV", "eztv"),
	cardigannIndexer("Nyaa", "nyaa"),
}

func (m *Manager) seedDefaultIndexers() {
	checkURL := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", m.BaseURL(), m.apiKey)
	resp, err := http.Get(checkURL)
	if err != nil {
		log.Printf("[prowlarr] cannot check indexers: %v", err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// If response is non-empty array, indexers already configured
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 2 && trimmed[0] == '[' && trimmed[1] != ']' {
		log.Printf("[prowlarr] indexers already configured, skipping seed")
		return
	}

	postURL := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", m.BaseURL(), m.apiKey)
	for _, idx := range defaultIndexers {
		data, _ := json.Marshal(idx)
		r, err := http.Post(postURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("[prowlarr] seed %s: %v", idx.Name, err)
			continue
		}
		r.Body.Close()
		if r.StatusCode == 201 {
			log.Printf("[prowlarr] seeded: %s", idx.Name)
		} else {
			log.Printf("[prowlarr] seed %s: status %d", idx.Name, r.StatusCode)
		}
	}
}
```

- [ ] **Step 2: Build check**

```bash
cd backend/go-server && go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/go-server/internal/prowlarr/manager.go
git commit -m "feat(prowlarr): process manager — spawn, health-check, API key, indexer seed"
```

---

### Task 4: Go — Wire Prowlarr manager into server.go

**Files:**
- Modify: `backend/go-server/internal/grpc/server.go`

- [ ] **Step 1: Update server.go**

```go
package grpc_server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"server/config"
	"server/internal/prowlarr"
	"server/internal/torrent"
	pb "server/proto"

	"google.golang.org/grpc"
)

func StartServer(cfg config.Config) {
	// Start Prowlarr — non-fatal: search unavailable if it fails
	pm := prowlarr.NewManager(
		cfg.Prowlarr.BinDir,
		cfg.Prowlarr.DataDir,
		cfg.Prowlarr.Port,
		cfg.Prowlarr.SeedIndexers,
	)
	if err := pm.Start(); err != nil {
		log.Printf("[prowlarr] startup failed: %v (search will be unavailable)", err)
	}
	defer pm.Stop()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", cfg.GRPCPort, err)
	}

	client, err := torrent.NewClient(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to create torrent client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Error closing torrent client: %v", err)
		}
	}()

	repo := torrent.NewRepository(client, cfg.MetadataTimeout)
	svc := torrent.NewTorrentService(repo)

	grpcServer := grpc.NewServer()
	pb.RegisterTorrentServiceServer(grpcServer, svc)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped: %v", err)
			stop <- syscall.SIGTERM
		}
	}()

	<-stop
	log.Println("Shutting down...")
	repo.Clearup()
	grpcServer.GracefulStop()
}
```

- [ ] **Step 2: Build check**

```bash
cd backend/go-server && go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/go-server/internal/grpc/server.go
git commit -m "feat(prowlarr): wire manager into grpc server lifecycle"
```

---

### Task 5: Spring — SearchResult DTOs

**Files:**
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SearchResult.java`
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SearchResultsResponse.java`

- [ ] **Step 1: Create SearchResult.java**

```java
package org.devMandali.magnetPlay.model;

import java.util.List;

public record SearchResult(
    String title,
    long sizeBytes,
    int seeders,
    int peers,
    String indexer,
    String pubDate,
    String magnetUrl,
    List<String> qualityTags
) {}
```

- [ ] **Step 2: Create SearchResultsResponse.java**

```java
package org.devMandali.magnetPlay.model;

import java.util.List;

public record SearchResultsResponse(List<SearchResult> results, int total) {}
```

- [ ] **Step 3: Commit**

```bash
git add backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SearchResult.java \
        backend/mp-spring/src/main/java/org/devMandali/magnetPlay/model/SearchResultsResponse.java
git commit -m "feat(search): SearchResult and SearchResultsResponse DTOs"
```

---

### Task 6: Spring — application.yml prowlarr config block

**Files:**
- Modify: `backend/mp-spring/src/main/resources/application.yml`

- [ ] **Step 1: Append prowlarr config**

Add to end of `application.yml`:

```yaml
# ── Prowlarr search ───────────────────────────────────────────────────────────
prowlarr:
  url: ${PROWLARR_URL:http://localhost:9696}
  api-key: ${PROWLARR_API_KEY:}
  config-xml: ${PROWLARR_CONFIG_XML:../prowlarr-data/config.xml}
```

> `api-key` is blank by default — `ProwlarrClient` reads it from `config.xml` lazily on first search.
> Override via env var `PROWLARR_API_KEY` if desired.

- [ ] **Step 2: Commit**

```bash
git add backend/mp-spring/src/main/resources/application.yml
git commit -m "feat(search): prowlarr config block in application.yml"
```

---

### Task 7: Spring — ProwlarrClient (Torznab XML → SearchResult list)

**Files:**
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/ProwlarrClient.java`

No new Maven dependencies — uses `WebClient` (spring-boot-starter-webflux, already present) and Java's built-in `javax.xml.parsers.DocumentBuilder`.

- [ ] **Step 1: Create ProwlarrClient.java**

```java
package org.devMandali.magnetPlay.client;

import org.devMandali.magnetPlay.model.SearchResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.WebClient;
import org.w3c.dom.Document;
import org.w3c.dom.Element;
import org.w3c.dom.NodeList;
import reactor.core.publisher.Mono;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

@Component
public class ProwlarrClient {

    private static final Logger log = LoggerFactory.getLogger(ProwlarrClient.class);
    private static final Pattern QUALITY = Pattern.compile(
        "(2160p|1080p|720p|480p|x265|x264|HEVC|BluRay|WEB-DL|WEBRip|HDRip|HDR|REMUX)",
        Pattern.CASE_INSENSITIVE);
    private static final Pattern API_KEY_RE = Pattern.compile("<ApiKey>([^<]+)</ApiKey>");

    @Value("${prowlarr.url}")
    private String prowlarrUrl;

    @Value("${prowlarr.api-key:}")
    private String configuredApiKey;

    @Value("${prowlarr.config-xml:../prowlarr-data/config.xml}")
    private String configXmlPath;

    private volatile String resolvedApiKey;
    private final WebClient webClient;

    public ProwlarrClient(WebClient.Builder builder) {
        this.webClient = builder.build();
    }

    public Mono<List<SearchResult>> search(String query) {
        String key = resolveApiKey();
        if (key == null || key.isBlank()) {
            return Mono.error(new IllegalStateException("Prowlarr API key unavailable — is Go sidecar running?"));
        }
        String url = prowlarrUrl + "/api/v1/indexer/all/newznab?apikey=" + key
            + "&t=search&q=" + encode(query);

        return webClient.get()
            .uri(url)
            .retrieve()
            .bodyToMono(String.class)
            .map(this::parseAtomXml)
            .doOnError(e -> log.error("Prowlarr search error: {}", e.getMessage()));
    }

    private String resolveApiKey() {
        if (resolvedApiKey != null) return resolvedApiKey;
        if (configuredApiKey != null && !configuredApiKey.isBlank()) {
            resolvedApiKey = configuredApiKey;
            return resolvedApiKey;
        }
        try {
            String xml = Files.readString(Path.of(configXmlPath));
            Matcher m = API_KEY_RE.matcher(xml);
            if (m.find()) {
                resolvedApiKey = m.group(1);
                log.info("Prowlarr API key loaded from config.xml");
                return resolvedApiKey;
            }
        } catch (IOException e) {
            log.warn("Cannot read prowlarr config.xml at {}: {}", configXmlPath, e.getMessage());
        }
        return null;
    }

    private List<SearchResult> parseAtomXml(String xml) {
        List<SearchResult> results = new ArrayList<>();
        try {
            DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
            factory.setNamespaceAware(true);
            DocumentBuilder builder = factory.newDocumentBuilder();
            Document doc = builder.parse(new ByteArrayInputStream(xml.getBytes(StandardCharsets.UTF_8)));
            NodeList items = doc.getElementsByTagName("item");
            for (int i = 0; i < items.getLength(); i++) {
                results.add(parseItem((Element) items.item(i)));
            }
        } catch (Exception e) {
            log.error("Torznab XML parse error: {}", e.getMessage());
        }
        return results;
    }

    private SearchResult parseItem(Element item) {
        String title = text(item, "title");
        String pubDate = text(item, "pubDate");
        String magnetUrl = "";
        long sizeBytes = 0;
        int seeders = 0;
        int peers = 0;
        String indexer = "";

        NodeList enclosures = item.getElementsByTagName("enclosure");
        if (enclosures.getLength() > 0) {
            Element enc = (Element) enclosures.item(0);
            String url = enc.getAttribute("url");
            if (url.startsWith("magnet:")) magnetUrl = url;
            String len = enc.getAttribute("length");
            if (!len.isBlank()) sizeBytes = parseLong(len);
        }

        // torznab:attr elements (namespace: http://torznab.com/schemas/2015/feed)
        NodeList attrs = item.getElementsByTagNameNS("http://torznab.com/schemas/2015/feed", "attr");
        for (int i = 0; i < attrs.getLength(); i++) {
            Element attr = (Element) attrs.item(i);
            String name = attr.getAttribute("name");
            String value = attr.getAttribute("value");
            switch (name) {
                case "seeders"   -> seeders = parseInt(value);
                case "peers"     -> peers = parseInt(value);
                case "size"      -> { if (sizeBytes == 0) sizeBytes = parseLong(value); }
                case "magneturl" -> { if (magnetUrl.isBlank()) magnetUrl = value; }
                case "indexer"   -> indexer = value;
            }
        }

        List<String> qualityTags = new ArrayList<>();
        Matcher m = QUALITY.matcher(title);
        while (m.find()) {
            String tag = m.group(1).toUpperCase();
            if (!qualityTags.contains(tag)) qualityTags.add(tag);
        }

        return new SearchResult(title, sizeBytes, seeders, peers, indexer, pubDate, magnetUrl, qualityTags);
    }

    private String text(Element parent, String tag) {
        NodeList nl = parent.getElementsByTagName(tag);
        return nl.getLength() == 0 ? "" : nl.item(0).getTextContent();
    }

    private int parseInt(String s) {
        try { return Integer.parseInt(s.trim()); } catch (NumberFormatException e) { return 0; }
    }

    private long parseLong(String s) {
        try { return Long.parseLong(s.trim()); } catch (NumberFormatException e) { return 0; }
    }

    private String encode(String q) {
        return java.net.URLEncoder.encode(q, StandardCharsets.UTF_8);
    }
}
```

- [ ] **Step 2: Build check**

```bash
cd backend/mp-spring && ./mvnw compile -q
```

Expected: `BUILD SUCCESS`

- [ ] **Step 3: Commit**

```bash
git add backend/mp-spring/src/main/java/org/devMandali/magnetPlay/client/ProwlarrClient.java
git commit -m "feat(search): ProwlarrClient — Torznab XML proxy with quality tag extraction"
```

---

### Task 8: Spring — SearchController

**Files:**
- Create: `backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/SearchController.java`

- [ ] **Step 1: Create SearchController.java**

```java
package org.devMandali.magnetPlay.controller;

import org.devMandali.magnetPlay.client.ProwlarrClient;
import org.devMandali.magnetPlay.model.SearchResultsResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import reactor.core.publisher.Mono;

@RestController
@RequestMapping("/v1/search")
public class SearchController {

    private static final Logger log = LoggerFactory.getLogger(SearchController.class);
    private final ProwlarrClient prowlarrClient;

    public SearchController(ProwlarrClient prowlarrClient) {
        this.prowlarrClient = prowlarrClient;
    }

    @GetMapping
    public Mono<ResponseEntity<SearchResultsResponse>> search(@RequestParam String q) {
        if (q == null || q.isBlank()) {
            return Mono.just(ResponseEntity.badRequest().<SearchResultsResponse>build());
        }
        return prowlarrClient.search(q.trim())
            .map(results -> ResponseEntity.ok(new SearchResultsResponse(results, results.size())))
            .doOnError(e -> log.error("Search failed: {}", e.getMessage()))
            .onErrorReturn(ResponseEntity.status(503).<SearchResultsResponse>build());
    }
}
```

- [ ] **Step 2: Build check**

```bash
cd backend/mp-spring && ./mvnw compile -q
```

Expected: `BUILD SUCCESS`

- [ ] **Step 3: Commit**

```bash
git add backend/mp-spring/src/main/java/org/devMandali/magnetPlay/controller/SearchController.java
git commit -m "feat(search): SearchController GET /v1/search?q="
```

---

### Task 9: Frontend — SearchResult types

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Append to types/index.ts**

Add at end of file:

```typescript
export interface SearchResult {
  title: string;
  sizeBytes: number;
  seeders: number;
  peers: number;
  indexer: string;
  pubDate: string;
  magnetUrl: string;
  qualityTags: string[];
}

export interface SearchResultsResponse {
  results: SearchResult[];
  total: number;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/types/index.ts
git commit -m "feat(search): SearchResult and SearchResultsResponse types"
```

---

### Task 10: Frontend — SearchPanel component

**Files:**
- Create: `frontend/src/components/SearchPanel.tsx`

- [ ] **Step 1: Create SearchPanel.tsx**

```tsx
import { useState, useCallback } from 'react';
import { SearchResult, SearchResultsResponse } from '../types';

interface Props {
  onSelect: (magnetUrl: string) => void;
}

function fmtBytes(bytes: number): string {
  if (bytes === 0) return '—';
  if (bytes >= 1e9) return (bytes / 1e9).toFixed(2) + ' GB';
  if (bytes >= 1e6) return (bytes / 1e6).toFixed(1) + ' MB';
  return Math.round(bytes / 1e3) + ' KB';
}

export default function SearchPanel({ onSelect }: Props) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = useCallback(async () => {
    const q = query.trim();
    if (!q) return;
    setLoading(true);
    setError(null);
    setResults([]);
    try {
      const res = await fetch(`/v1/search?q=${encodeURIComponent(q)}`, {
        signal: AbortSignal.timeout(30000),
      });
      if (res.status === 503) throw new Error('Search unavailable — Prowlarr not running');
      if (!res.ok) throw new Error(`Search error: ${res.status}`);
      const data: SearchResultsResponse = await res.json();
      setResults(data.results);
      if (data.results.length === 0) setError('No results found');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, [query]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          type="text"
          placeholder="Search torrents… (e.g. Dune 2024 1080p)"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleSearch()}
          style={{ flex: 1, fontFamily: 'var(--mono)', fontSize: 12 }}
          autoFocus
        />
        <button
          className="btn btn-primary"
          onClick={handleSearch}
          disabled={!query.trim() || loading}
        >
          {loading ? '…' : '⌕ Search'}
        </button>
      </div>

      {error && (
        <div style={{
          padding: '8px 12px',
          background: 'rgba(230,57,70,.1)',
          border: '1px solid rgba(230,57,70,.3)',
          borderRadius: 6,
          fontSize: 11,
          color: '#ff8080',
          fontFamily: 'var(--mono)',
        }}>
          ⚠ {error}
        </div>
      )}

      {results.length > 0 && (
        <div style={{
          display: 'flex', flexDirection: 'column', gap: 4,
          maxHeight: 300, overflowY: 'auto',
        }}>
          {results.map((r, i) => (
            <div
              key={i}
              onClick={() => r.magnetUrl && onSelect(r.magnetUrl)}
              title={r.magnetUrl ? 'Click to use this torrent' : 'No magnet link available'}
              style={{
                padding: '8px 12px',
                background: 'var(--surface)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                cursor: r.magnetUrl ? 'pointer' : 'default',
                opacity: r.magnetUrl ? 1 : 0.45,
                display: 'flex',
                flexDirection: 'column',
                gap: 4,
              }}
            >
              <div style={{ fontSize: 12, fontWeight: 500, wordBreak: 'break-word', lineHeight: 1.3 }}>
                {r.title}
              </div>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
                {r.qualityTags.map(tag => (
                  <span
                    key={tag}
                    style={{
                      padding: '1px 5px', borderRadius: 3,
                      background: 'var(--accent-dim, rgba(99,102,241,.15))',
                      color: 'var(--accent, #818cf8)',
                      fontSize: 9, fontFamily: 'var(--mono)',
                    }}
                  >
                    {tag}
                  </span>
                ))}
                <span style={{ marginLeft: 'auto', fontSize: 10, color: 'var(--muted)' }}>
                  {fmtBytes(r.sizeBytes)}
                </span>
                <span style={{ fontSize: 10, color: r.seeders > 0 ? '#4ade80' : 'var(--muted)' }}>
                  ↑{r.seeders}
                </span>
                <span style={{ fontSize: 10, color: 'var(--muted)' }}>↔{r.peers}</span>
                {r.indexer && (
                  <span style={{ fontSize: 10, color: 'var(--muted)' }}>[{r.indexer}]</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Type check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/SearchPanel.tsx
git commit -m "feat(search): SearchPanel with quality badges, size, seeds, peers"
```

---

### Task 11: Frontend — Tab toggle at Step 1 in App.tsx

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Add import**

Add to the imports block at the top of `App.tsx`:

```typescript
import SearchPanel from './components/SearchPanel';
```

- [ ] **Step 2: Add inputMode state**

Add after existing state declarations inside `App()` (after the `configCollapsed` line):

```typescript
const [inputMode, setInputMode] = useState<'magnet' | 'search'>('magnet');
```

- [ ] **Step 3: Replace the Step 1 wizard-panel div**

Find the comment `{/* Step 1 */}` and replace the entire `<div className="wizard-panel">` block (from `<div className="wizard-panel">` through its closing `</div>`) with:

```tsx
{/* Step 1 */}
<div className="wizard-panel">
  {/* Tab bar */}
  <div style={{ display: 'flex', marginBottom: 12, borderBottom: '1px solid var(--border)' }}>
    {(['magnet', 'search'] as const).map(mode => (
      <button
        key={mode}
        onClick={() => setInputMode(mode)}
        style={{
          padding: '6px 16px',
          background: 'none',
          border: 'none',
          borderBottom: inputMode === mode ? '2px solid var(--accent, #818cf8)' : '2px solid transparent',
          color: inputMode === mode ? 'var(--text)' : 'var(--muted)',
          cursor: 'pointer',
          fontSize: 12,
          fontFamily: 'var(--mono)',
          marginBottom: -1,
        }}
      >
        {mode === 'magnet' ? '⎘ Paste Magnet' : '⌕ Search'}
      </button>
    ))}
  </div>

  {inputMode === 'search' ? (
    <SearchPanel
      onSelect={(mag) => {
        setMagnetLink(mag);
        setFetchState('idle');
        setFetchError(null);
        setInputMode('magnet');
      }}
    />
  ) : (
    <div className="form-grid" style={{ gridTemplateColumns: '1fr' }}>
      <div className="form-group full">
        <label htmlFor="magnetLink">Magnet Link</label>
        <input
          id="magnetLink"
          type="text"
          placeholder="magnet:?xt=urn:btih:…"
          value={magnetLink}
          onChange={e => { setMagnetLink(e.target.value); setFetchState('idle'); setFetchError(null); }}
          onKeyDown={e => e.key === 'Enter' && handleFetchFiles()}
          style={{ fontFamily: 'var(--mono)', fontSize: 11 }}
        />
        {fetchState === 'error' && (
          <div style={{
            marginTop: 6, padding: '8px 12px',
            background: 'rgba(230,57,70,.1)', border: '1px solid rgba(230,57,70,.3)',
            borderRadius: 6, fontFamily: 'var(--mono)', fontSize: 10,
            color: '#ff8080', display: 'flex', alignItems: 'center', gap: 8,
          }}>
            <span>⚠</span>{fetchError}
          </div>
        )}
      </div>
    </div>
  )}

  {inputMode === 'magnet' && (
    <div className="form-actions">
      <button
        className="btn btn-primary"
        onClick={handleFetchFiles}
        disabled={!magnetLink.trim() || fetchState === 'loading'}
      >
        ▶ Fetch Files
      </button>
      <button
        className="btn btn-ghost"
        onClick={async () => {
          try {
            const text = await navigator.clipboard.readText();
            if (text) { setMagnetLink(text); setFetchState('idle'); setFetchError(null); }
          } catch { /* clipboard permission denied */ }
        }}
        title="Paste from clipboard"
      >
        ⎘ Paste
      </button>
      {magnetLink && (
        <button
          className="btn btn-ghost"
          onClick={() => { setMagnetLink(''); setFetchState('idle'); setFetchError(null); }}
          title="Clear"
        >
          ✕ Clear
        </button>
      )}
    </div>
  )}
</div>
```

- [ ] **Step 4: Type check**

```bash
cd frontend && npx tsc --noEmit
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(search): search tab at step 1, result select auto-fills magnet input"
```

---

## Self-Review

**Spec coverage:**
- ✅ Search tab alongside magnet input — tab bar: `⎘ Paste Magnet` | `⌕ Search`
- ✅ Rich results — title, quality badges, size, seeders, peers, indexer source
- ✅ No poster/thumbnail — quality tags + seeds replace visual enrichment
- ✅ Go manages Prowlarr subprocess — `server.go` starts manager before gRPC, stops on shutdown
- ✅ Auto-download binary from GitHub on first run — zip-slip guard included
- ✅ Default indexers seeded (1337x, YTS, EZTV, Nyaa) — skips if already configured
- ✅ User can customize via Prowlarr UI at `http://localhost:9696`
- ✅ Spring proxies Torznab XML → JSON — API key never reaches browser
- ✅ Graceful 503 when Prowlarr unavailable — search tab shows clear error

**Known gaps (out of scope for v1):**
- Linux/macOS tar.gz extraction not implemented in `downloader.go` — code comment flags it
- No `.torrent` fallback (indexers without magnet links show as unclickable, opacity 0.45)
- First run download is ~80MB with no UI progress indicator — Go logs to stdout only
