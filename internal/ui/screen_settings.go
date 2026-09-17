package ui

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
)

type settingKind int

const (
	setInput settingKind = iota
	setToggle
)

type settingField struct {
	kind    settingKind
	label   string
	hint    string
	input   textinput.Model
	options []string // for toggles: cycle order
	apply   func(m *Model, val string)
	value   func(m *Model) string
}

// settingsState is the settings screen's state.
type settingsState struct {
	fields  []settingField
	cursor  int
	editing bool
}

var (
	mergeFormats   = []string{"mp4", "mkv", "webm"}
	audioFormats   = []string{"best", "m4a", "mp3", "opus", "flac", "wav", "vorbis"}
	playlistCaps   = []string{"best", "2160", "1440", "1080", "720", "480", "360"}
	fragmentCounts = []string{"1", "2", "4", "8"}
	boolOptions    = []string{"on", "off"}
)

func boolLabel(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func cycle(cur string, options []string) string {
	if len(options) == 0 {
		return cur
	}
	for i, o := range options {
		if o == cur {
			return options[(i+1)%len(options)]
		}
	}
	return options[0]
}

func newTextInput(val string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(val)
	ti.CharLimit = 512
	ti.Width = 60
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Red)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.RedBright)
	return ti
}

func newSettingsState(cfg *config.Config) settingsState {
	s := settingsState{}
	s.fields = []settingField{
		{
			kind:  setInput,
			label: "Download folder",
			hint:  "where files are saved (~ is expanded)",
			apply: func(m *Model, val string) { m.cfg.DownloadDir = val },
			value: func(m *Model) string { return m.cfg.DownloadDir },
		},
		{
			kind:  setInput,
			label: "Filename template",
			hint:  "yt-dlp output template, e.g. %(title)s.%(ext)s",
			apply: func(m *Model, val string) { m.cfg.FilenameTemplate = val },
			value: func(m *Model) string { return m.cfg.FilenameTemplate },
		},
		{
			kind:    setToggle,
			label:   "Merge container",
			hint:    "container when joining video + audio",
			options: mergeFormats,
			apply:   func(m *Model, val string) { m.cfg.MergeFormat = val },
			value:   func(m *Model) string { return m.cfg.MergeFormat },
		},
		{
			kind:    setToggle,
			label:   "Audio format",
			hint:    "target format for audio extraction",
			options: audioFormats,
			apply:   func(m *Model, val string) { m.cfg.AudioFormat = val },
			value:   func(m *Model) string { return m.cfg.AudioFormat },
		},
		{
			kind:    setToggle,
			label:   "Playlist quality",
			hint:    "resolution cap when downloading playlists",
			options: playlistCaps,
			apply:   func(m *Model, val string) { m.cfg.PlaylistQuality = val },
			value:   func(m *Model) string { return m.cfg.PlaylistQuality },
		},
		{
			kind:    setToggle,
			label:   "Embed thumbnail",
			hint:    "write the cover image into the file",
			options: boolOptions,
			apply:   func(m *Model, val string) { m.cfg.EmbedThumbnail = val == "on" },
			value:   func(m *Model) string { return boolLabel(m.cfg.EmbedThumbnail) },
		},
		{
			kind:    setToggle,
			label:   "Embed metadata",
			hint:    "write title/artist tags into the file",
			options: boolOptions,
			apply:   func(m *Model, val string) { m.cfg.EmbedMetadata = val == "on" },
			value:   func(m *Model) string { return boolLabel(m.cfg.EmbedMetadata) },
		},
		{
			kind:    setToggle,
			label:   "Concurrent fragments",
			hint:    "parallel HLS/DASH fragment downloads",
			options: fragmentCounts,
			apply: func(m *Model, val string) {
				if n, err := strconv.Atoi(val); err == nil {
					m.cfg.ConcurrentFragments = n
				}
			},
			value: func(m *Model) string { return strconv.Itoa(m.cfg.ConcurrentFragments) },
		},
	}
	for i := range s.fields {
		if s.fields[i].kind == setInput {
			s.fields[i].input = newTextInput(s.fields[i].value(&Model{cfg: cfg}))
		}
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// updateSettings handles keys on the settings screen.
func (m *Model) updateSettings(key string, msg tea.Msg) (tea.Model, tea.Cmd) {
	s := &m.settings

	if s.editing {
		switch key {
		case "enter", "esc":
			s.editing = false
			s.fields[s.cursor].input.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			s.fields[s.cursor].input, cmd = s.fields[s.cursor].input.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "up", "ctrl+p":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "ctrl+n":
		if s.cursor < len(s.fields)-1 {
			s.cursor++
		}
	case "enter", " ":
		f := &s.fields[s.cursor]
		switch f.kind {
		case setInput:
			s.editing = true
			return m, f.input.Focus()
		case setToggle:
			f.apply(m, cycle(f.value(m), f.options))
		}
	case "s", "ctrl+s":
		return m, m.saveSettings()
	case "esc":
		m.screen = screenURL
		return m, m.urlInput.Focus()
	}
	return m, nil
}

func (m *Model) saveSettings() tea.Cmd {
	s := &m.settings
	for _, f := range s.fields {
		if f.kind == setInput {
			f.apply(m, f.input.Value())
		}
	}
	m.cfg.Normalize()
	if err := m.cfg.Save(m.cfgPath); err != nil {
		return m.setStatus("failed to save: "+err.Error(), true)
	}
	if err := os.MkdirAll(m.cfg.DownloadDir, 0o755); err != nil {
		return m.setStatus("cannot create download folder: "+err.Error(), true)
	}
	m.screen = screenURL
	cmd := m.setStatus("Settings saved → "+m.cfgPath, false)
	return tea.Batch(cmd, m.urlInput.Focus())
}

func (m *Model) viewSettings() string {
	s := &m.settings
	var b strings.Builder
	b.WriteString(styles.CardTitle.Render("▍ Settings") + "\n")
	b.WriteString("  " + styles.HelpDesc.Render("config file: "+m.cfgPath) + "\n\n")

	for i, f := range s.fields {
		cursor := "  "
		if i == s.cursor {
			cursor = "› "
		}

		var value string
		switch f.kind {
		case setInput:
			if s.editing && i == s.cursor {
				value = f.input.View()
			} else {
				v := f.input.Value()
				value = styles.Row.Render(truncate(v, m.width-40))
			}
		case setToggle:
			v := f.value(m)
			st := styles.RowDesc
			if v == "on" || (v != "off" && v != "best") {
				st = styles.Checkmark
			}
			value = st.Render(v)
		}

		line := cursor + styles.InfoLabel.Render(f.label) + "  " + value
		if i == s.cursor {
			pad := m.width - 6 - lipgloss.Width(cursor+styles.InfoLabel.Render(f.label)) - lipgloss.Width(f.input.Value())
			if f.kind == setToggle {
				pad = 0
			}
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			line = styles.CursorRow.Render(line)
		}
		b.WriteString(line + "\n")
		if i == s.cursor && f.hint != "" {
			b.WriteString("    " + styles.HelpDesc.Render(f.hint) + "\n")
		}
	}
	return b.String()
}
