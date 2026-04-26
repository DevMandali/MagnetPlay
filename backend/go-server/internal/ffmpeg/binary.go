package ffmpeg

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

const btbnReleasesURL = "https://api.github.com/repos/BtbN/FFmpeg-Builds/releases/latest"

var (
	apiClient = &http.Client{Timeout: 30 * time.Second}
	dlClient  = &http.Client{Timeout: 20 * time.Minute}
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func FFmpegBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

func FFprobeBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffprobe.exe"
	}
	return "ffprobe"
}

func ResolvePath(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("configured binary not found: %s", configPath)
	}
	if found := findInDir(binDir, FFmpegBinaryName()); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("ffmpeg not found in %s", binDir)
}

func ResolveFFprobePath(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("configured ffprobe not found: %s", configPath)
	}
	if found := findInDir(binDir, FFprobeBinaryName()); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("ffprobe not found in %s", binDir)
}

func EnsureFFmpeg(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}
	if found := findInDir(binDir, FFmpegBinaryName()); found != "" {
		return found, nil
	}
	if err := downloadFFmpeg(binDir); err != nil {
		return "", err
	}
	found := findInDir(binDir, FFmpegBinaryName())
	if found == "" {
		return "", fmt.Errorf("ffmpeg not found in %s after download", binDir)
	}
	return found, nil
}

func EnsureFFprobe(configPath, binDir string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}
	if found := findInDir(binDir, FFprobeBinaryName()); found != "" {
		return found, nil
	}
	if err := downloadFFmpeg(binDir); err != nil {
		return "", err
	}
	found := findInDir(binDir, FFprobeBinaryName())
	if found == "" {
		return "", fmt.Errorf("ffprobe not found in %s after download", binDir)
	}
	return found, nil
}

func findInDir(dir, name string) string {
	var found string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func platformAssetSuffix() string {
	switch runtime.GOOS {
	case "windows":
		return "win64-gpl.zip"
	case "linux":
		return "linux64-gpl.tar.xz"
	default:
		return ""
	}
}

func downloadFFmpeg(binDir string) error {
	suffix := platformAssetSuffix()
	if suffix == "" {
		return fmt.Errorf("unsupported platform %s/%s — set FFmpegPath in config", runtime.GOOS, runtime.GOARCH)
	}

	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch FFmpeg release: %w", err)
	}

	var assetURL string
	for _, a := range rel.Assets {
		if strings.Contains(a.Name, "master-latest") && strings.HasSuffix(a.Name, suffix) {
			assetURL = a.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		return fmt.Errorf("no FFmpeg asset matching suffix %q in release %s", suffix, rel.TagName)
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	ext := ".zip"
	if strings.HasSuffix(assetURL, ".tar.xz") {
		ext = ".tar.xz"
	}
	tmp, err := os.CreateTemp("", "ffmpeg-*"+ext)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	fmt.Printf("[ffmpeg] downloading %s...\n", rel.TagName)
	if err := downloadToFile(assetURL, tmp); err != nil {
		return fmt.Errorf("download FFmpeg: %w", err)
	}

	if ext == ".zip" {
		return extractZip(tmp.Name(), binDir)
	}
	return fmt.Errorf("tar.xz extraction not implemented; set FFmpegPath in config or extract manually to %s", binDir)
}

func fetchLatestRelease() (*githubRelease, error) {
	req, _ := http.NewRequest("GET", btbnReleasesURL, nil)
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
		return fmt.Errorf("HTTP %d downloading FFmpeg", resp.StatusCode)
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
			continue
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
