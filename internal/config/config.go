// Package config loads and stores ytdl-tui's user configuration.
//
// The config is a small YAML file under the user's config directory
// (os.UserConfigDir, e.g. ~/.config on Linux, ~/Library/Application Support
// on macOS, %AppData% on Windows), at ytdl-tui/config.yaml. It is created
// with defaults on first run and can be edited from the TUI's settings
// screen or by hand.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxDownloadDirs caps how many download locations a user can configure.
const MaxDownloadDirs = 5

// Config holds the user-editable settings for ytdl-tui.
type Config struct {
	// DownloadDirs is the list of save locations (1..MaxDownloadDirs).
	// DownloadDirs[0] is the default; when there are several, the TUI asks
	// where to save before every download.
	DownloadDirs []string `yaml:"download_dirs"`

	// DownloadDir is the legacy single-folder key. It is still read (and
	// migrated into DownloadDirs) but never written by the app.
	DownloadDir string `yaml:"download_dir,omitempty"`

	FilenameTemplate    string `yaml:"filename_template"`    // yt-dlp output template
	MergeFormat         string `yaml:"merge_format"`         // mp4 | mkv | webm
	AudioFormat         string `yaml:"audio_format"`         // best | m4a | mp3 | opus | flac | wav | vorbis
	PlaylistQuality     string `yaml:"playlist_quality"`     // best | 2160 | 1440 | 1080 | 720 | 480 | 360
	EmbedThumbnail      bool   `yaml:"embed_thumbnail"`      //
	EmbedMetadata       bool   `yaml:"embed_metadata"`       //
	ConcurrentFragments int    `yaml:"concurrent_fragments"` // 1..8
	CookiesFromBrowser  string `yaml:"cookies_from_browser"` // yt-dlp value, e.g. "firefox" or "firefox:/path"
	CookiesFile         string `yaml:"cookies_file"`         // exported cookies.txt (fallback)
}

// Default returns the built-in configuration used on first run.
func Default() Config {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, "Downloads", "YouTube")
	return Config{
		DownloadDirs:        []string{base},
		DownloadDir:         base,
		FilenameTemplate:    "%(title)s [%(id)s].%(ext)s",
		MergeFormat:         "mp4",
		AudioFormat:         "m4a",
		PlaylistQuality:     "1080",
		EmbedThumbnail:      true,
		EmbedMetadata:       true,
		ConcurrentFragments: 4,
	}
}

// DefaultPath returns the standard config file location across platforms.
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
	// Dir fields get zeroed so a file that sets only one of them doesn't
	// inherit the default's value; Normalize restores sane fallbacks.
	cfg.DownloadDirs = nil
	cfg.DownloadDir = ""
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

	// Migrate the legacy single-folder key.
	c.DownloadDirs = append(c.DownloadDirs, c.DownloadDir)

	// Clean the list: expand ~, trim, drop empties, dedupe, cap.
	cleaned := make([]string, 0, len(c.DownloadDirs))
	seen := map[string]bool{}
	for _, d := range c.DownloadDirs {
		d = expandPath(strings.TrimSpace(d))
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		cleaned = append(cleaned, d)
		if len(cleaned) >= MaxDownloadDirs {
			break
		}
	}
	if len(cleaned) == 0 {
		cleaned = []string{def.DownloadDirs[0]}
	}
	c.DownloadDirs = cleaned

	// DownloadDir tracks the primary folder for backward compatibility.
	c.DownloadDir = c.DownloadDirs[0]

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

	// Values are passed straight to yt-dlp: trim, never lowercase (the
	// browser form may embed a filesystem path). ~ is expanded for files.
	c.CookiesFromBrowser = strings.TrimSpace(c.CookiesFromBrowser)
	c.CookiesFile = expandPath(strings.TrimSpace(c.CookiesFile))
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

// ResolveOutput returns the -o value for yt-dlp for the given folder: the
// download dir joined with the filename template.
func (c Config) ResolveOutput(dir string) string {
	if dir == "" {
		dir = c.DownloadDirs[0]
	}
	return filepath.Join(dir, c.FilenameTemplate)
}

func expandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			rest := strings.TrimPrefix(strings.TrimPrefix(p, "~"), "/")
			rest = strings.TrimPrefix(rest, `\`)
			return filepath.Join(home, rest)
		}
	}
	return p
}
