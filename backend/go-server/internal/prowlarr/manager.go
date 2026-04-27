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
	"strings"
	"sync"
	"time"
)

var healthClient = &http.Client{Timeout: 5 * time.Second}
var seedClient = &http.Client{Timeout: 60 * time.Second}

type Manager struct {
	mu       sync.Mutex
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

	absDataDir, err := filepath.Abs(m.dataDir)
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	m.dataDir = absDataDir // ensure all subsequent uses (readAPIKey, etc.) use absolute path

	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return err
	}

	m.mu.Lock()
	m.cmd = exec.Command(binPath,
		"--data="+m.dataDir,
		fmt.Sprintf("--port=%d", m.port),
		"--nobrowser",
	)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.mu.Unlock()

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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		m.cmd.Process.Kill()
		m.cmd.Wait() // reap child
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

func (m *Manager) APIKey() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.apiKey
}
func (m *Manager) BaseURL() string { return fmt.Sprintf("http://localhost:%d", m.port) }

func (m *Manager) waitReady(timeout time.Duration) error {
	url := fmt.Sprintf("http://localhost:%d/ping", m.port)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := healthClient.Get(url)
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
		if err != nil {
			if i == 0 {
				log.Printf("[prowlarr] waiting for config.xml: %v", err)
			}
			time.Sleep(2 * time.Second)
			continue
		}
		if match := apiKeyRe.FindSubmatch(data); len(match) == 2 {
			return string(match[1]), nil
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("API key not found in %s", configPath)
}

// ── Indexer seeding ───────────────────────────────────────────────────────────

type indexerDef struct {
	name    string
	defFile string
}

var defaultIndexers = []indexerDef{
	{name: "The Pirate Bay", defFile: "thepiratebay"},
	{name: "1337x", defFile: "1337x"},
	{name: "YTS", defFile: "yts"},
	{name: "EZTV", defFile: "eztv"},
	{name: "TorrentGalaxy", defFile: "torrentgalaxyclone"},
	{name: "LimeTorrents", defFile: "limetorrents"},
	{name: "TorrentDownloads", defFile: "torrentdownloads"},
	{name: "RuTracker", defFile: "rutracker-ru"},
	{name: "MagnetDownload", defFile: "magnetdownload"},
}

func (m *Manager) seedDefaultIndexers() {
	baseURL := m.BaseURL()
	apiKey := m.APIKey()

	// Always delete existing indexers and re-seed from definition-specific schemas.
	// Stale indexers get marked "Gone/410" by Prowlarr when definitions update;
	// a fresh seed from the schema endpoint avoids this.
	m.deleteAllIndexers(baseURL, apiKey)

	appProfileID, err := m.defaultAppProfileID()
	if err != nil {
		log.Printf("[prowlarr] cannot fetch app profiles: %v", err)
		return
	}

	// Fetch all definition-specific schemas. Each entry in the list is a
	// ready-to-POST template for one indexer — no field guessing needed.
	schemas, err := m.fetchIndexerSchemas(baseURL, apiKey)
	if err != nil {
		log.Printf("[prowlarr] cannot fetch indexer schemas: %v", err)
		return
	}

	postURL := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", baseURL, apiKey)
	for _, def := range defaultIndexers {
		schema, ok := schemas[def.defFile]
		if !ok {
			log.Printf("[prowlarr] no schema found for %s (defFile=%s)", def.name, def.defFile)
			continue
		}
		payload := cloneMap(schema)
		payload["name"] = def.name
		payload["enableRss"] = false
		payload["enableAutomaticSearch"] = true
		payload["enableInteractiveSearch"] = true
		payload["appProfileId"] = appProfileID
		payload["priority"] = 25

		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[prowlarr] marshal %s: %v", def.name, err)
			continue
		}
		r, err := seedClient.Post(postURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("[prowlarr] seed %s: %v", def.name, err)
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		r.Body.Close()
		if r.StatusCode == 201 {
			log.Printf("[prowlarr] seeded: %s", def.name)
		} else if r.StatusCode == 400 {
			log.Printf("[prowlarr] seed %s skipped (unreachable): %s", def.name, extractProwlarrError(respBody))
		} else {
			log.Printf("[prowlarr] seed %s: status %d — %s", def.name, r.StatusCode, string(respBody))
		}
	}
}

func (m *Manager) deleteAllIndexers(baseURL, apiKey string) {
	resp, err := healthClient.Get(fmt.Sprintf("%s/api/v1/indexer?apikey=%s", baseURL, apiKey))
	if err != nil {
		log.Printf("[prowlarr] failed to list indexers for deletion: %v", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	resp.Body.Close()
	var existing []map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(body), &existing); err != nil || len(existing) == 0 {
		if err != nil {
			log.Printf("[prowlarr] failed to parse indexer list: %v", err)
		}
		return
	}
	for _, idx := range existing {
		id, ok := idx["id"].(float64)
		if !ok {
			continue
		}
		req, _ := http.NewRequest(http.MethodDelete,
			fmt.Sprintf("%s/api/v1/indexer/%d?apikey=%s", baseURL, int(id), apiKey), nil)
		r, err := healthClient.Do(req)
		if err == nil {
			r.Body.Close()
			log.Printf("[prowlarr] deleted indexer %d", int(id))
		}
	}
}

// fetchIndexerSchemas returns a map of defFile → schema, fetched from Prowlarr's
// schema endpoint which lists one ready-to-POST template per available definition.
func (m *Manager) fetchIndexerSchemas(baseURL, apiKey string) (map[string]map[string]interface{}, error) {
	schemaClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := schemaClient.Get(fmt.Sprintf("%s/api/v1/indexer/schema?apikey=%s", baseURL, apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var schemas []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&schemas); err != nil {
		return nil, fmt.Errorf("decode schemas: %w", err)
	}
	result := make(map[string]map[string]interface{}, len(schemas))
	for _, s := range schemas {
		// Each Cardigann schema has a definitionFile field identifying which tracker it is.
		fields, _ := s["fields"].([]interface{})
		for _, f := range fields {
			field, _ := f.(map[string]interface{})
			if field["name"] == "definitionFile" {
				if val, _ := field["value"].(string); val != "" {
					result[val] = s
				}
				break
			}
		}
	}
	log.Printf("[prowlarr] loaded %d indexer schemas", len(result))
	return result, nil
}

// extractProwlarrError pulls the first errorMessage from Prowlarr's validation JSON array.
func extractProwlarrError(body []byte) string {
	var errs []struct {
		ErrorMessage string `json:"errorMessage"`
	}
	if err := json.Unmarshal(body, &errs); err == nil && len(errs) > 0 {
		msg := errs[0].ErrorMessage
		// Truncate after the first sentence to keep logs clean
		if i := strings.Index(msg, ". See:"); i > 0 {
			msg = msg[:i]
		}
		return msg
	}
	if len(body) > 120 {
		return string(body[:120])
	}
	return string(body)
}

// cloneMap deep-copies a map via JSON round-trip. Returns empty map on error.
func cloneMap(m map[string]interface{}) map[string]interface{} {
	b, err := json.Marshal(m)
	if err != nil {
		return make(map[string]interface{})
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return make(map[string]interface{})
	}
	return out
}

func (m *Manager) defaultAppProfileID() (int, error) {
	url := fmt.Sprintf("%s/api/v1/appprofile?apikey=%s", m.BaseURL(), m.APIKey())
	resp, err := healthClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var profiles []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return 0, fmt.Errorf("decode app profiles: %w", err)
	}
	if len(profiles) == 0 {
		return 0, fmt.Errorf("no app profiles found in Prowlarr")
	}
	id, ok := profiles[0]["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("app profile id not a number")
	}
	return int(id), nil
}
