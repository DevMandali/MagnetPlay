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
	"sync"
	"time"
)

var healthClient = &http.Client{Timeout: 5 * time.Second}
var schemaClient = &http.Client{Timeout: 60 * time.Second}

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
	{"1337x", "1337x"},
	{"YTS", "yts"},
	{"EZTV", "eztv"},
	{"Nyaa", "nyaa"},
}

func (m *Manager) seedDefaultIndexers() {
	listURL := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", m.BaseURL(), m.APIKey())

	// Skip if indexers already configured
	resp, err := healthClient.Get(listURL)
	if err != nil {
		log.Printf("[prowlarr] cannot check indexers: %v", err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var existing []json.RawMessage
	if json.Unmarshal(bytes.TrimSpace(body), &existing) == nil && len(existing) > 0 {
		log.Printf("[prowlarr] indexers already configured, skipping seed")
		return
	}

	// Fetch the Cardigann schema template from Prowlarr
	schema, err := m.cardigannSchema()
	if err != nil {
		log.Printf("[prowlarr] cannot fetch Cardigann schema: %v", err)
		return
	}

	postURL := fmt.Sprintf("%s/api/v1/indexer?apikey=%s", m.BaseURL(), m.APIKey())
	for _, def := range defaultIndexers {
		payload := cloneMap(schema)
		payload["name"] = def.name
		payload["enableRss"] = false
		payload["enableAutomaticSearch"] = true
		payload["enableInteractiveSearch"] = true

		// Set definitionFile inside the fields array
		if fields, ok := payload["fields"].([]interface{}); ok {
			for _, f := range fields {
				if field, ok := f.(map[string]interface{}); ok && field["name"] == "definitionFile" {
					field["value"] = def.defFile
				}
			}
		}

		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[prowlarr] marshal %s: %v", def.name, err)
			continue
		}
		r, err := healthClient.Post(postURL, "application/json", bytes.NewReader(data))
		if err != nil {
			log.Printf("[prowlarr] seed %s: %v", def.name, err)
			continue
		}
		r.Body.Close()
		if r.StatusCode == 201 {
			log.Printf("[prowlarr] seeded: %s", def.name)
		} else {
			log.Printf("[prowlarr] seed %s: status %d", def.name, r.StatusCode)
		}
	}
}

// cardigannSchema fetches Prowlarr's Cardigann indexer schema to use as a POST template.
func (m *Manager) cardigannSchema() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/indexer/schema?apikey=%s", m.BaseURL(), m.APIKey())
	resp, err := schemaClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var schemas []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&schemas); err != nil {
		return nil, err
	}
	for _, s := range schemas {
		if impl, ok := s["implementation"].(string); ok && impl == "Cardigann" {
			return s, nil
		}
	}
	return nil, fmt.Errorf("Cardigann not found in schema list")
}

// cloneMap deep-copies a map via JSON round-trip.
func cloneMap(m map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(m)
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	return out
}
