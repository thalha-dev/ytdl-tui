// Package config loads and stores ytdl-tui's user configuration.
//
// The config is a small YAML file at $XDG_CONFIG_HOME/ytdl-tui/config.yaml
// (usually ~/.config/ytdl-tui/config.yaml). It is created with defaults on
// first run and can be edited from the TUI's settings screen or by hand.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the user-editable settings for ytdl-tui.
type Config struct {
	DownloadDir         string `yaml:"download_dir"`
	FilenameTemplate    string `yaml:"filename_template"`
	MergeFormat         string `yaml:"merge_format"`     // mp4 | mkv | webm
	AudioFormat         string `yaml:"audio_format"`     // best | m4a | mp3 | opus | flac | wav | vorbis
	PlaylistQuality     string `yaml:"playlist_quality"` // best | 2160 | 1440 | 1080 | 720 | 480 | 360
	EmbedThumbnail      bool   `yaml:"embed_thumbnail"`
	EmbedMetadata       bool   `yaml:"embed_metadata"`
	ConcurrentFragments int    `yaml:"concurrent_fragments"` // 1..8
}

// Default returns the built-in configuration used on first run.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DownloadDir:         filepath.Join(home, "Downloads", "YouTube"),
		FilenameTemplate:    "%(title)s [%(id)s].%(ext)s",
		MergeFormat:         "mp4",
		AudioFormat:         "m4a",
		PlaylistQuality:     "1080",
		EmbedThumbnail:      true,
		EmbedMetadata:       true,
		ConcurrentFragments: 4,
	}
}

// DefaultPath returns the standard config file location.
func DefaultPath() string {
	if base, err := os.UserConfigDir(); err == nil && base != "" {
		return filepath.Join(base, "ytdl-tui", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ytdl-tui", "config.yaml")
}

// Load reads the config at path. When the file does not exist it is created
// with defaults. Blank or invalid fields fall back to their defaults, so the
// app can always run.
func Load(path string) (Config, error) {
	cfg := Default()
	missing := false

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Default(), fmt.Errorf("parse %s: %w", path, err)
		}
	case os.IsNotExist(err):
		missing = true
	default:
		return Default(), fmt.Errorf("read %s: %w", path, err)
	}

	cfg.Normalize()

	if missing {
		if err := cfg.Save(path); err != nil {
			return cfg, fmt.Errorf("write default config: %w", err)
		}
	}
	return cfg, nil
}

// Normalize fills blank/invalid fields with defaults and expands ~ in paths.
func (c *Config) Normalize() {
	def := Default()

	c.DownloadDir = expandPath(strings.TrimSpace(c.DownloadDir))
	if c.DownloadDir == "" {
		c.DownloadDir = def.DownloadDir
	}

	c.FilenameTemplate = strings.TrimSpace(c.FilenameTemplate)
	if c.FilenameTemplate == "" {
		c.FilenameTemplate = def.FilenameTemplate
	}

	switch c.MergeFormat {
	case "mp4", "mkv", "webm":
	default:
		c.MergeFormat = def.MergeFormat
	}

	switch c.AudioFormat {
	case "best", "m4a", "mp3", "opus", "flac", "wav", "vorbis":
	default:
		c.AudioFormat = def.AudioFormat
	}

	switch c.PlaylistQuality {
	case "best", "2160", "1440", "1080", "720", "480", "360":
	default:
		c.PlaylistQuality = def.PlaylistQuality
	}

	if c.ConcurrentFragments < 1 || c.ConcurrentFragments > 8 {
		c.ConcurrentFragments = def.ConcurrentFragments
	}
}

// Save writes the config as YAML, creating parent directories as needed.
func (c Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ResolveOutput returns the -o value for yt-dlp: the download dir joined
// with the filename template.
func (c Config) ResolveOutput() string {
	return filepath.Join(c.DownloadDir, c.FilenameTemplate)
}

func expandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/"))
		}
	}
	return p
}
