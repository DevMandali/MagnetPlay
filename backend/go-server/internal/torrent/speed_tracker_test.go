package torrent

import (
	"testing"
	"time"
)

func TestSpeedTracker_ZeroAtStart(t *testing.T) {
	tr := NewSpeedTracker(5)
	if got := tr.SpeedBps(); got != 0 {
		t.Errorf("want 0 at start, got %f", got)
	}
}

func TestSpeedTracker_SingleSample(t *testing.T) {
	tr := NewSpeedTracker(5)
	tr.Record(1000, time.Now().Add(-1*time.Second))
	tr.Record(2000, time.Now())
	speed := tr.SpeedBps()
	if speed < 900 || speed > 1100 {
		t.Errorf("want ~1000 bps, got %f", speed)
	}
}

func TestSpeedTracker_OldSamplesDropped(t *testing.T) {
	tr := NewSpeedTracker(2) // 2-second window
	tr.Record(0, time.Now().Add(-10*time.Second))
	tr.Record(500, time.Now().Add(-5*time.Second)) // outside window
	tr.Record(1000, time.Now().Add(-1*time.Second))
	tr.Record(2000, time.Now())
	speed := tr.SpeedBps()
	if speed < 900 || speed > 1100 {
		t.Errorf("want ~1000 bps (only last 2s window), got %f", speed)
	}
}
