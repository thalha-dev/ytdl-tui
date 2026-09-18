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
	if filepath.Base(cfg.DownloadDirs[0]) != "YT" || cfg.DownloadDirs[0][:1] == "~" {
		t.Errorf("DownloadDirs = %v, want expanded ~ path ending in YT", cfg.DownloadDirs)
	}
	if cfg.DownloadDir != cfg.DownloadDirs[0] {
		t.Errorf("DownloadDir = %q, should track primary folder", cfg.DownloadDir)
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
	cfg.DownloadDirs = []string{"/tmp/dl"}
	cfg.Normalize()
	want := filepath.Join("/tmp/dl", "%(title)s [%(id)s].%(ext)s")
	if got := cfg.ResolveOutput("/tmp/dl"); got != want {
		t.Errorf("ResolveOutput() = %q, want %q", got, want)
	}
	if empty := cfg.ResolveOutput(""); empty != want {
		t.Errorf("ResolveOutput(\"\") = %q, want default folder %q", empty, want)
	}
}

func TestSaveRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.DownloadDirs = []string{"/tmp/custom", "/tmp/other"}
	cfg.PlaylistQuality = "720"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DownloadDirs[0] != "/tmp/custom" || got.PlaylistQuality != "720" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestLegacyDownloadDirMigrates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := "download_dir: \"/tmp/legacy\"\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.DownloadDirs) != 1 || cfg.DownloadDirs[0] != "/tmp/legacy" {
		t.Errorf("DownloadDirs = %v, want [/tmp/legacy]", cfg.DownloadDirs)
	}
}

func TestNormalizeCapsAndDedupesDirs(t *testing.T) {
	cfg := Config{
		DownloadDirs: []string{
			"/a", "/a", "", "~/b", "/c", "/d", "/e", "/f", // > 5, dup, blank
		},
	}
	cfg.Normalize()

	if len(cfg.DownloadDirs) != MaxDownloadDirs {
		t.Fatalf("len(DownloadDirs) = %d (%v), want %d", len(cfg.DownloadDirs), cfg.DownloadDirs, MaxDownloadDirs)
	}
	if cfg.DownloadDirs[0] != "/a" {
		t.Errorf("first dir = %q, want /a", cfg.DownloadDirs[0])
	}
	if cfg.DownloadDir != cfg.DownloadDirs[0] {
		t.Errorf("legacy DownloadDir should track primary, got %q", cfg.DownloadDir)
	}
}

func TestNormalizeEmptyDirsFallsBackToDefault(t *testing.T) {
	cfg := Config{}
	cfg.Normalize()
	if len(cfg.DownloadDirs) != 1 || cfg.DownloadDirs[0] != Default().DownloadDirs[0] {
		t.Errorf("DownloadDirs = %v, want default", cfg.DownloadDirs)
	}
}
