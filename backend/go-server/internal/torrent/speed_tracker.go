package torrent

import (
	"sync"
	"time"
)

type sample struct {
	bytes int64
	at    time.Time
}

type SpeedTracker struct {
	mu      sync.Mutex
	window  time.Duration
	samples []sample
}

func NewSpeedTracker(windowSeconds int) *SpeedTracker {
	return &SpeedTracker{window: time.Duration(windowSeconds) * time.Second}
}

// Record adds a cumulative byte-count observation at the given time.
func (s *SpeedTracker) Record(totalBytes int64, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := at.Add(-s.window)
	kept := s.samples[:0]
	for _, p := range s.samples {
		if p.at.After(cutoff) {
			kept = append(kept, p)
		}
	}
	s.samples = append(kept, sample{bytes: totalBytes, at: at})
}

// SpeedBps returns bytes/second averaged over the window. Returns 0 if fewer than 2 samples.
func (s *SpeedTracker) SpeedBps() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.samples) < 2 {
		return 0
	}
	oldest := s.samples[0]
	newest := s.samples[len(s.samples)-1]
	elapsed := newest.at.Sub(oldest.at).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(newest.bytes-oldest.bytes) / elapsed
}
