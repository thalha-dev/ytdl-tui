package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/ui/components"
	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
)

// menu option payloads.
type menuVideo struct{}
type menuAudio struct{}
type menuExpert struct{}
type menuPlaylistAll struct{}
type menuPlaylistAudio struct{}
type menuPlaylistSelect struct{}

// buildVideoMenu populates the media-type menu for a single video.
func (m *Model) buildVideoMenu() {
	items := []components.Item{}

	if m.maxResLabel() != "" {
		items = append(items, components.Item{
			Title: "Video",
			Desc:  "pick codec & resolution · up to " + m.maxResLabel(),
			Data:  menuVideo{},
		})
	}
	if m.hasAudioFormats() {
		conv := ""
		if m.cfg.AudioFormat != "best" {
			conv = " → " + m.cfg.AudioFormat
		}
		items = append(items, components.Item{
			Title: "Audio only",
			Desc:  "extract the audio track" + conv,
			Data:  menuAudio{},
		})
	}
	items = append(items, components.Item{
		Title: "All formats (expert)",
		Desc:  "browse every raw yt-dlp format",
		Data:  menuExpert{},
	})

	m.menuPicker = components.NewPicker("What do you want to download?", items, false)
}

// buildPlaylistMenu populates the menu shown for playlists.
func (m *Model) buildPlaylistMenu() {
	n := len(m.info.Entries)
	capQ := m.cfg.PlaylistQuality
	if capQ == "best" {
		capQ = "best available"
	} else {
		capQ = "≤ " + capQ + "p"
	}

	items := []components.Item{
		{
			Title: "Download entire playlist",
			Desc:  fmt.Sprintf("%d videos · %s · merged %s", n, capQ, m.cfg.MergeFormat),
			Data:  menuPlaylistAll{},
		},
		{
			Title: "Audio only (entire playlist)",
			Desc:  fmt.Sprintf("%d tracks · extracts to %s", n, m.cfg.AudioFormat),
			Data:  menuPlaylistAudio{},
		},
		{
			Title: "Select videos",
			Desc:  "fuzzy search and pick specific entries",
			Data:  menuPlaylistSelect{},
		},
	}
	m.menuPicker = components.NewPicker("This is a playlist", items, false)
}

func (m *Model) hasAudioFormats() bool {
	for _, f := range m.info.DownloadableFormats() {
		if f.IsAudioOnly() {
			return true
		}
	}
	return false
}

// updateMenu handles keys on the info/menu screen.
func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.menuPicker.Update(msg)

	if items, ok := m.menuPicker.ConsumeSelected(); ok {
		switch items[0].Data.(type) {
		case menuVideo:
			m.buildFamilyPicker()
			m.screen = screenFamily
			return m, nil
		case menuAudio:
			m.buildAudioPicker()
			m.screen = screenAudio
			return m, nil
		case menuExpert:
			m.buildExpertPicker()
			m.screen = screenExpert
			return m, nil
		case menuPlaylistAll:
			return m, m.startPlaylistDownload(false, nil)
		case menuPlaylistAudio:
			return m, m.startPlaylistDownload(true, nil)
		case menuPlaylistSelect:
			m.buildPlaylistPicker()
			m.screen = screenPlaylist
			return m, nil
		}
	}
	if m.menuPicker.ConsumeCancel() {
		m.screen = screenURL
		m.info = nil
		return m, m.urlInput.Focus()
	}
	return m, nil
}

func (m *Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(styles.CardTitle.Render("▍ "+truncate(m.info.Title, m.width-8)) + "\n")
	b.WriteString("  " + styles.RowDesc.Render(m.videoMetaLine()) + "\n")
	if m.info.WebpageURL != "" {
		b.WriteString("  " + styles.HelpDesc.Render(truncate(m.info.WebpageURL, m.width-6)) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.menuPicker.View(m.width - 6))
	return b.String()
}
