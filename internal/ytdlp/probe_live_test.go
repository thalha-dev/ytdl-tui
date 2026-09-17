package ytdlp

import (
	"context"
	"testing"
	"time"
)

// TestProbeLive hits YouTube for real. Skipped in -short runs so
// `go test ./...` stays hermetic.
func TestProbeLive(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	info, err := Probe(ctx, "https://www.youtube.com/watch?v=278IRQ6HSi4")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if info.IsPlaylist() {
		t.Fatal("expected a single video")
	}
	if info.Title == "" {
		t.Fatal("empty title")
	}
	if len(info.DownloadableFormats()) < 10 {
		t.Fatalf("expected a rich format list, got %d", len(info.DownloadableFormats()))
	}
	if info.Duration <= 0 {
		t.Error("duration missing")
	}
}
