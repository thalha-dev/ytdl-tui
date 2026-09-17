package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/config"
)

func TestBoolTogglesFlipWithoutPanic(t *testing.T) {
	cfg := config.Default()
	m := &Model{cfg: &cfg}
	m.settings = newSettingsState(m.cfg)

	thumb, meta := -1, -1
	for i, f := range m.settings.fields {
		switch f.label {
		case "Embed thumbnail":
			thumb = i
		case "Embed metadata":
			meta = i
		}
	}
	if thumb < 0 || meta < 0 {
		t.Fatal("bool rows missing from settings fields")
	}

	enter := tea.KeyMsg{Type: tea.KeyEnter}

	m.settings.cursor = thumb
	m.updateSettings("enter", enter) // on -> off
	if cfg.EmbedThumbnail {
		t.Error("EmbedThumbnail should have flipped to off")
	}
	m.updateSettings("enter", enter) // off -> on
	if !cfg.EmbedThumbnail {
		t.Error("EmbedThumbnail should have flipped back to on")
	}

	m.settings.cursor = meta
	m.updateSettings("enter", enter) // on -> off
	if cfg.EmbedMetadata {
		t.Error("EmbedMetadata should have flipped to off")
	}
}

func TestCycleFieldsAdvance(t *testing.T) {
	cfg := config.Default()
	m := &Model{cfg: &cfg}
	m.settings = newSettingsState(m.cfg)

	enter := tea.KeyMsg{Type: tea.KeyEnter}

	merge := -1
	for i, f := range m.settings.fields {
		if f.label == "Merge container" {
			merge = i
		}
	}
	m.settings.cursor = merge
	m.updateSettings("enter", enter) // mp4 -> mkv
	if cfg.MergeFormat != "mkv" {
		t.Errorf("MergeFormat = %q, want mkv", cfg.MergeFormat)
	}
	m.updateSettings("enter", enter) // mkv -> webm
	if cfg.MergeFormat != "webm" {
		t.Errorf("MergeFormat = %q, want webm", cfg.MergeFormat)
	}
}

func TestCycleEmptyOptionsIsSafe(t *testing.T) {
	if got := cycle("on", nil); got != "on" {
		t.Errorf("cycle with empty options = %q, want unchanged", got)
	}
}
