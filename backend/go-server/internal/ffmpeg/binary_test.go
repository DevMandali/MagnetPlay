package ffmpeg_test

import (
	"os"
	"path/filepath"
	"testing"

	"server/internal/ffmpeg"
)

func TestFFmpegBinaryName(t *testing.T) {
	name := ffmpeg.FFmpegBinaryName()
	if name == "" {
		t.Fatal("FFmpegBinaryName() returned empty string")
	}
}

func TestFFprobeBinaryName(t *testing.T) {
	name := ffmpeg.FFprobeBinaryName()
	if name == "" {
		t.Fatal("FFprobeBinaryName() returned empty string")
	}
}

func TestConfigPathOverride_FFmpeg(t *testing.T) {
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "ffmpeg_fake")
	if err := os.WriteFile(fakeBin, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := ffmpeg.ResolvePath(fakeBin, tmp)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if got != fakeBin {
		t.Fatalf("expected %s, got %s", fakeBin, got)
	}
}

func TestResolvePath_UsesExistingInBinDir(t *testing.T) {
	tmp := t.TempDir()
	binName := ffmpeg.FFmpegBinaryName()
	fakeBin := filepath.Join(tmp, binName)
	if err := os.WriteFile(fakeBin, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := ffmpeg.ResolvePath("", tmp)
	if err != nil {
		t.Fatalf("ResolvePath returned error: %v", err)
	}
	if got != fakeBin {
		t.Fatalf("expected %s, got %s", fakeBin, got)
	}
}
