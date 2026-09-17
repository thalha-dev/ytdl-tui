package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
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

// TestSettingsSequence replicates the e2e driver's exact keystrokes: every
// toggle must land, including the cookies row whose display label ("none")
// differs from its raw value ("").
func TestSettingsSequence(t *testing.T) {
	cfg := config.Default()
	m := &Model{cfg: &cfg}
	m.settings = newSettingsState(m.cfg)

	down := func() { m.updateSettings("down", tea.KeyMsg{Type: tea.KeyDown}) }
	up := func() { m.updateSettings("up", tea.KeyMsg{Type: tea.KeyUp}) }
	enter := func() { m.updateSettings("enter", tea.KeyMsg{Type: tea.KeyEnter}) }

	down()
	down()
	enter() // merge: mp4 -> mkv
	down()
	down()
	down()
	enter() // thumbnail: on -> off
	down()
	enter() // metadata: on -> off
	down()
	enter() // fragments: 4 -> 8
	up()
	up()
	enter() // thumbnail: off -> on
	down()
	down()
	down()
	enter() // cookies browser: none -> detected
	m.updateSettings("s", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	if cfg.MergeFormat != "mkv" {
		t.Errorf("MergeFormat = %q, want mkv", cfg.MergeFormat)
	}
	if !cfg.EmbedThumbnail || cfg.EmbedMetadata {
		t.Errorf("bool toggles wrong: thumbnail=%v metadata=%v", cfg.EmbedThumbnail, cfg.EmbedMetadata)
	}
	if cfg.ConcurrentFragments != 8 {
		t.Errorf("ConcurrentFragments = %d, want 8", cfg.ConcurrentFragments)
	}
	want := ytdlp.DetectBrowsers()
	if len(want) == 0 {
		t.Skip("no browsers detected on this machine")
	}
	if cfg.CookiesFromBrowser != want[0] {
		t.Errorf("CookiesFromBrowser = %q, want detected %q", cfg.CookiesFromBrowser, want[0])
	}
}
