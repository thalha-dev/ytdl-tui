package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created on first run: %v", err)
	}
	if cfg.MergeFormat != "mp4" {
		t.Errorf("MergeFormat = %q, want mp4", cfg.MergeFormat)
	}
	if cfg.FilenameTemplate == "" {
		t.Error("FilenameTemplate should default, got empty")
	}
	if cfg.ConcurrentFragments < 1 || cfg.ConcurrentFragments > 8 {
		t.Errorf("ConcurrentFragments = %d, out of range", cfg.ConcurrentFragments)
	}
}

func TestLoadMergesPartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "download_dir: \"~/Music/YT\"\nmerge_format: mkv\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.MergeFormat != "mkv" {
		t.Errorf("MergeFormat = %q, want mkv (user value)", cfg.MergeFormat)
	}
	if cfg.AudioFormat != "m4a" {
		t.Errorf("AudioFormat = %q, want default m4a", cfg.AudioFormat)
	}
	if filepath.Base(cfg.DownloadDir) != "YT" || cfg.DownloadDir[:1] == "~" {
		t.Errorf("DownloadDir = %q, want expanded ~ path ending in YT", cfg.DownloadDir)
	}
}

func TestNormalizeRejectsInvalidValues(t *testing.T) {
	cfg := Config{MergeFormat: "avi", AudioFormat: "wma", PlaylistQuality: "9000", ConcurrentFragments: 99}
	cfg.Normalize()

	if cfg.MergeFormat != "mp4" {
		t.Errorf("MergeFormat = %q, want reset to mp4", cfg.MergeFormat)
	}
	if cfg.AudioFormat != "m4a" {
		t.Errorf("AudioFormat = %q, want reset to m4a", cfg.AudioFormat)
	}
	if cfg.PlaylistQuality != "1080" {
		t.Errorf("PlaylistQuality = %q, want reset to 1080", cfg.PlaylistQuality)
	}
	if cfg.ConcurrentFragments != 4 {
		t.Errorf("ConcurrentFragments = %d, want reset to 4", cfg.ConcurrentFragments)
	}
}

func TestResolveOutput(t *testing.T) {
	cfg := Default()
	cfg.DownloadDir = "/tmp/dl"
	got := cfg.ResolveOutput()
	if want := "/tmp/dl/%(title)s [%(id)s].%(ext)s"; got != want {
		t.Errorf("ResolveOutput() = %q, want %q", got, want)
	}
}

func TestSaveRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.DownloadDir = "/tmp/custom"
	cfg.PlaylistQuality = "720"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DownloadDir != "/tmp/custom" || got.PlaylistQuality != "720" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}
