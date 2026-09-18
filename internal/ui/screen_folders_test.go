package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
)

func keyStr(s string) tea.KeyMsg {
	if r := []rune(s); len(r) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: r}
	}
	return tea.KeyMsg{Type: tea.KeyEnter}
}

func TestFoldersManagerAddEditDelete(t *testing.T) {
	cfg := config.Default()
	m := &Model{cfg: &cfg, width: 120}
	m.enterFolders()

	if m.screen != screenFolders {
		t.Fatalf("screen = %v, want screenFolders", m.screen)
	}

	// add a second folder
	m.updateFolders("a", keyStr("a"))
	if !m.folders.editing || m.folders.editingIdx != -1 {
		t.Fatal("add mode not entered")
	}
	m.folders.input.SetValue("/tmp/yt-two")
	m.updateFolders("enter", keyStr("enter"))
	if len(cfg.DownloadDirs) != 2 || cfg.DownloadDirs[1] != "/tmp/yt-two" {
		t.Fatalf("DownloadDirs = %v, want 2 entries ending /tmp/yt-two", cfg.DownloadDirs)
	}
	if cfg.DownloadDir != cfg.DownloadDirs[0] {
		t.Errorf("primary should stay DownloadDirs[0], got %q", cfg.DownloadDir)
	}

	// duplicate add is rejected
	m.updateFolders("a", keyStr("a"))
	m.folders.input.SetValue("/tmp/yt-two")
	m.updateFolders("enter", keyStr("enter"))
	if len(cfg.DownloadDirs) != 2 {
		t.Errorf("duplicate folder should be dropped, got %v", cfg.DownloadDirs)
	}

	// edit the second entry
	m.folders.cursor = 1
	m.updateFolders("e", keyStr("e"))
	if !m.folders.editing || m.folders.editingIdx != 1 {
		t.Fatal("edit mode not entered")
	}
	m.folders.input.SetValue("/tmp/yt-renamed")
	m.updateFolders("enter", keyStr("enter"))
	if cfg.DownloadDirs[1] != "/tmp/yt-renamed" {
		t.Errorf("edit failed: %v", cfg.DownloadDirs)
	}

	// delete down to one folder, then the last delete is refused
	m.updateFolders("d", keyStr("d"))
	if len(cfg.DownloadDirs) != 1 {
		t.Fatalf("delete failed: %v", cfg.DownloadDirs)
	}
	_, cmd := m.updateFolders("d", keyStr("d"))
	if len(cfg.DownloadDirs) != 1 || cmd == nil {
		t.Error("deleting the last folder must be refused with a toast")
	}

	// esc returns to settings
	m.updateFolders("esc", keyStr("esc"))
	if m.screen != screenSettings {
		t.Errorf("esc should return to settings, got %v", m.screen)
	}
}

func TestFoldersCapAtFive(t *testing.T) {
	cfg := config.Default()
	cfg.DownloadDirs = []string{"/1", "/2", "/3", "/4", "/5"}
	cfg.Normalize()
	m := &Model{cfg: &cfg, width: 120}
	m.enterFolders()

	m.updateFolders("a", keyStr("a"))
	_, cmd := m.updateFolders("a", keyStr("a")) // add refused at cap, no editing state
	if m.folders.editing || cmd == nil {
		t.Error("adding a 6th folder must be refused with a toast")
	}
	if len(cfg.DownloadDirs) != config.MaxDownloadDirs {
		t.Errorf("DownloadDirs = %d, want %d", len(cfg.DownloadDirs), config.MaxDownloadDirs)
	}
}

func TestSaveToRouting(t *testing.T) {
	mkModel := func(dirs ...string) *Model {
		cfg := config.Default()
		cfg.DownloadDirs = dirs
		cfg.DownloadDir = "" // don't let the default leak into the list
		cfg.Normalize()
		return &Model{cfg: &cfg, width: 120}
	}

	// single folder: straight to download
	m := mkModel("/tmp/one")
	m.screen = screenResolution
	cmd := m.offerSaveTo(ytdlp.Request{URLs: []string{"u"}}, "t", "s")
	if m.screen != screenDownload || cmd == nil {
		t.Fatalf("single folder should launch immediately, screen=%v", m.screen)
	}
	if m.dl.dir != "/tmp/one" {
		t.Errorf("dl.dir = %q, want /tmp/one", m.dl.dir)
	}

	// multiple folders: save-to picker first
	m = mkModel("/tmp/one", "/tmp/two")
	m.screen = screenResolution
	cmd = m.offerSaveTo(ytdlp.Request{URLs: []string{"u"}}, "t", "s")
	if m.screen != screenSaveTo || cmd != nil {
		t.Fatalf("multiple folders should show the picker, screen=%v", m.screen)
	}
	if m.saveToBack != screenResolution {
		t.Errorf("saveToBack = %v, want screenResolution", m.saveToBack)
	}

	// esc returns to the origin screen
	m.updateSaveTo(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenResolution {
		t.Errorf("esc should return to %v, got %v", screenResolution, m.screen)
	}

	// picking the second folder launches into it
	m.screen = screenSaveTo
	m.updateSaveTo(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSaveTo(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenDownload {
		t.Fatalf("screen = %v, want screenDownload", m.screen)
	}
	if m.dl.dir != "/tmp/two" {
		t.Errorf("dl.dir = %q, want /tmp/two", m.dl.dir)
	}
	if m.pendingReq != nil {
		t.Error("pending request should be cleared after launch")
	}
}

func TestSettingsCtrlJKNavigation(t *testing.T) {
	cfg := config.Default()
	m := &Model{cfg: &cfg}
	m.settings = newSettingsState(&cfg)

	m.updateSettings("ctrl+j", tea.KeyMsg{Type: tea.KeyCtrlJ})
	if m.settings.cursor != 1 {
		t.Errorf("ctrl+j should move down, cursor = %d", m.settings.cursor)
	}
	m.updateSettings("ctrl+k", tea.KeyMsg{Type: tea.KeyCtrlK})
	if m.settings.cursor != 0 {
		t.Errorf("ctrl+k should move up, cursor = %d", m.settings.cursor)
	}

	// the first row opens the folder manager
	m.updateSettings("enter", tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenFolders {
		t.Errorf("enter on Download folders should open the manager, screen = %v", m.screen)
	}
}

func TestSettingsViewShowsFoldersRow(t *testing.T) {
	cfg := config.Default()
	cfg.DownloadDirs = []string{"/tmp/one", "/tmp/two"}
	cfg.DownloadDir = ""
	cfg.Normalize()
	m := &Model{cfg: &cfg, width: 120}
	m.settings = newSettingsState(&cfg)

	view := m.viewSettings()
	if !strings.Contains(view, "Download folders") {
		t.Error("settings view should show the Download folders row")
	}
	if !strings.Contains(view, "2 folders") {
		t.Error("settings view should show the folder count")
	}
}
