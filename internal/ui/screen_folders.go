package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
)

// foldersState manages the download folder list. Changes apply to the live
// config immediately; persisting to disk happens via the settings screen's
// save (s), like every other setting.
type foldersState struct {
	cursor     int
	editing    bool
	editingIdx int // -1 when adding a new folder
	input      textinput.Model
}

func (m *Model) enterFolders() {
	ti := newTextInput("")
	m.folders = foldersState{editingIdx: -1, input: ti}
	m.screen = screenFolders
}

func (m *Model) updateFolders(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	f := &m.folders

	if f.editing {
		switch key {
		case "enter":
			val := f.input.Value()
			if val == "" {
				return m, m.setStatus("path cannot be empty", true)
			}
			if f.editingIdx >= 0 {
				m.cfg.DownloadDirs[f.editingIdx] = val
			} else {
				if len(m.cfg.DownloadDirs) >= config.MaxDownloadDirs {
					return m, m.setStatus(fmt.Sprintf("up to %d folders — delete one first", config.MaxDownloadDirs), true)
				}
				m.cfg.DownloadDirs = append(m.cfg.DownloadDirs, val)
				f.cursor = len(m.cfg.DownloadDirs) - 1
			}
			m.cfg.Normalize()
			f.editing = false
			f.editingIdx = -1
			return m, m.checkFolders()
		case "esc":
			f.editing = false
			f.editingIdx = -1
			return m, nil
		default:
			var cmd tea.Cmd
			f.input, cmd = f.input.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "up", "ctrl+k", "ctrl+p":
		if f.cursor > 0 {
			f.cursor--
		}
	case "down", "ctrl+j", "ctrl+n":
		if f.cursor < len(m.cfg.DownloadDirs)-1 {
			f.cursor++
		}
	case "a":
		if len(m.cfg.DownloadDirs) >= config.MaxDownloadDirs {
			return m, m.setStatus(fmt.Sprintf("up to %d folders — delete one first", config.MaxDownloadDirs), true)
		}
		f.editing = true
		f.editingIdx = -1
		f.input.SetValue("")
		return m, f.input.Focus()
	case "e":
		f.editing = true
		f.editingIdx = f.cursor
		f.input.SetValue(m.cfg.DownloadDirs[f.cursor])
		return m, f.input.Focus()
	case "d", "x", "delete":
		if len(m.cfg.DownloadDirs) <= 1 {
			return m, m.setStatus("keep at least one folder", true)
		}
		m.cfg.DownloadDirs = append(
			m.cfg.DownloadDirs[:f.cursor],
			m.cfg.DownloadDirs[f.cursor+1:]...,
		)
		m.cfg.Normalize()
		if f.cursor >= len(m.cfg.DownloadDirs) {
			f.cursor = len(m.cfg.DownloadDirs) - 1
		}
		return m, nil
	case "esc":
		m.screen = screenSettings
		return m, nil
	}
	return m, nil
}

// checkFolders creates every configured folder, surfacing the first failure.
func (m *Model) checkFolders() tea.Cmd {
	for _, dir := range m.cfg.DownloadDirs {
		if err := createDir(dir); err != nil {
			return m.setStatus("cannot create folder: "+err.Error(), true)
		}
	}
	return nil
}

func (m *Model) viewFolders() string {
	f := &m.folders
	var b strings.Builder
	b.WriteString(styles.CardTitle.Render("▍ Download folders") + "\n")
	b.WriteString("  " + styles.HelpDesc.Render(fmt.Sprintf(
		"the TUI asks where to save when there is more than one · up to %d", config.MaxDownloadDirs)) + "\n\n")

	for i, dir := range m.cfg.DownloadDirs {
		cursor := "  "
		if i == f.cursor && !f.editing {
			cursor = "› "
		}
		tag := ""
		if i == 0 {
			tag = "  " + styles.HelpDesc.Render("(default)")
		}
		line := cursor + styles.InfoLabel.Render(fmt.Sprintf("%d.", i+1)) + " " +
			styles.Row.Render(truncate(dir, m.width-18)) + tag
		if i == f.cursor && !f.editing {
			line = styles.CursorRow.Render(line)
		}
		b.WriteString(line + "\n")
	}

	if f.editing {
		label := "add"
		if f.editingIdx >= 0 {
			label = "edit"
		}
		b.WriteString("\n  " + styles.InfoLabel.Render(label+": ") + f.input.View() + "\n")
		b.WriteString("  " + styles.HelpDesc.Render("enter confirm · esc cancel") + "\n")
	}
	return b.String()
}
