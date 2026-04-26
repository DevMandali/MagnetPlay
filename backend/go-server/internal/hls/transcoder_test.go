package hls_test

import (
	"strings"
	"testing"

	"server/internal/hls"
)

const testFileURL = "http://localhost:8091/rawfile/abc123/dGVzdA"

func TestBuildRemuxArgs_SeekZero(t *testing.T) {
	args := hls.BuildRemuxArgs(0.0, testFileURL, "libx264", "aac")
	for i, a := range args {
		if a == "-ss" {
			t.Fatalf("expected no -ss for seekSec=0, found at index %d", i)
		}
	}
	if !containsSeq(args, []string{"-i", testFileURL}) {
		t.Errorf("expected -i %s", testFileURL)
	}
	if !containsSeq(args, []string{"-c:v", "libx264"}) {
		t.Error("expected -c:v libx264")
	}
	if !containsSeq(args, []string{"-c:a", "aac"}) {
		t.Error("expected -c:a aac")
	}
	if !containsSeq(args, []string{"-f", "mp4"}) {
		t.Error("expected -f mp4")
	}
	if !contains(args, "pipe:1") {
		t.Error("expected pipe:1 in args")
	}
	if !containsSeq(args, []string{"-movflags", "frag_keyframe+empty_moov"}) {
		t.Error("expected -movflags frag_keyframe+empty_moov")
	}
}

func TestBuildRemuxArgs_SeekNonZero(t *testing.T) {
	args := hls.BuildRemuxArgs(600.0, testFileURL, "libx264", "aac")

	ssIdx := -1
	for i, a := range args {
		if a == "-ss" {
			ssIdx = i
			break
		}
	}
	if ssIdx == -1 {
		t.Fatal("expected -ss for seekSec=600")
	}
	if ssIdx+1 >= len(args) || args[ssIdx+1] != "600.000" {
		t.Fatalf("expected -ss 600.000, got %v", args[ssIdx:ssIdx+2])
	}

	iIdx := -1
	for i, a := range args {
		if a == "-i" {
			iIdx = i
			break
		}
	}
	if ssIdx >= iIdx {
		t.Error("-ss must appear before -i")
	}
}

func TestBuildRemuxArgs_VideoCopy(t *testing.T) {
	args := hls.BuildRemuxArgs(0.0, testFileURL, "copy", "aac")
	if !containsSeq(args, []string{"-c:v", "copy"}) {
		t.Error("expected -c:v copy for videoCodec=copy")
	}
	if contains(args, "libx264") {
		t.Error("expected no libx264 when videoCodec=copy")
	}
}

func TestBuildRemuxArgs_AudioCopy(t *testing.T) {
	args := hls.BuildRemuxArgs(0.0, testFileURL, "libx264", "copy")
	if !containsSeq(args, []string{"-c:a", "copy"}) {
		t.Error("expected -c:a copy for audioCodec=copy")
	}
}

func TestBuildRemuxArgs_MapsAllAudio(t *testing.T) {
	args := hls.BuildRemuxArgs(0.0, testFileURL, "libx264", "aac")
	if !containsSeq(args, []string{"-map", "0:v:0"}) {
		t.Error("expected -map 0:v:0")
	}
	if !containsSeq(args, []string{"-map", "0:a"}) {
		t.Error("expected -map 0:a")
	}
}

func TestRemuxHandler_GetRemuxBaseURL(t *testing.T) {
	h := hls.NewRemuxHandler("ffmpeg", "http://localhost:8091/rawfile", "http://localhost:8091")
	url := h.GetRemuxBaseURL("abc123", "abc123:0", "copy", "aac")
	if !strings.Contains(url, "abc123") {
		t.Error("expected infoHash in URL")
	}
	if !strings.Contains(url, "vc=copy") {
		t.Error("expected vc=copy in URL")
	}
	if !strings.Contains(url, "ac=aac") {
		t.Error("expected ac=aac in URL")
	}
	if !strings.HasPrefix(url, "http://localhost:8091/remux/") {
		t.Errorf("unexpected URL prefix: %s", url)
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if strings.Contains(v, s) {
			return true
		}
	}
	return false
}

func containsSeq(slice []string, seq []string) bool {
	for i := 0; i <= len(slice)-len(seq); i++ {
		match := true
		for j, s := range seq {
			if slice[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
