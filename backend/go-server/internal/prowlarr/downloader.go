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
	"time"
)

const githubReleasesURL = "https://api.github.com/repos/Prowlarr/Prowlarr/releases/latest"
const maxFileSize = 500 * 1024 * 1024 // 500MB per file

var (
	apiClient = &http.Client{Timeout: 30 * time.Second}
	dlClient  = &http.Client{Timeout: 10 * time.Minute}
)

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

	if strings.HasSuffix(assetURL, ".zip") {
		if err := extractZip(tmpFile.Name(), binDir); err != nil {
			return err
		}
		if FindBinary(binDir) == "" {
			return fmt.Errorf("binary not found in %s after extraction", binDir)
		}
		return nil
	}
	// Linux/macOS tar.gz: requires archive/tar — add when targeting non-Windows
	return fmt.Errorf("tar.gz extraction not yet implemented; extract manually to %s", binDir)
}

func fetchLatestRelease() (*githubRelease, error) {
	req, _ := http.NewRequest("GET", githubReleasesURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "MagnetPlay/1.0")
	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	var rel githubRelease
	return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}

func downloadToFile(url string, dest *os.File) error {
	resp, err := dlClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d downloading asset", resp.StatusCode)
	}
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
		_, err = io.Copy(out, io.LimitReader(rc, maxFileSize))
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
