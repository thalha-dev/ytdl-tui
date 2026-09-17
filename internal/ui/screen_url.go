package ui

import (
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
)

// updateURL handles keys on the URL screen.
func (m *Model) updateURL(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.probing {
		if key == "esc" {
			if m.probeCancel != nil {
				m.probeCancel()
			}
			m.probeSeq++ // stale results are ignored
			m.probing = false
		}
		return m, nil
	}

	switch key {
	case "enter":
		raw := strings.TrimSpace(m.urlInput.Value())
		if !validURL(raw) {
			m.urlErr = "That doesn't look like a URL — paste an http(s) link to a video or playlist."
			return m, nil
		}
		return m, m.startProbe(raw)

	case "ctrl+s":
		m.settings = newSettingsState(m.cfg)
		m.screen = screenSettings
		return m, nil

	case "esc":
		if m.urlInput.Value() == "" {
			return m, tea.Quit
		}
		m.urlInput.SetValue("")
		m.urlErr = ""
		return m, nil

	default:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		m.urlErr = ""
		return m, cmd
	}
}

func validURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (m *Model) viewURL() string {
	var b strings.Builder
	b.WriteString(styles.CardTitle.Render("▍ What should we download?") + "\n\n")
	b.WriteString("  " + m.urlInput.View() + "\n")

	if m.probing {
		b.WriteString("\n  " + m.spinner.View() + " " + styles.PhaseText.Render("fetching video info…") + "\n")
	} else {
		b.WriteString("\n  " + styles.HelpDesc.Render("Videos, playlists, audio — anything yt-dlp understands.") + "\n")
	}

	if m.urlErr != "" {
		b.WriteString("\n  " + styles.ErrorText.Render("✗ "+truncate(m.urlErr, m.width-6)) + "\n")
	}
	return b.String()
}
