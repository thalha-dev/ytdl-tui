package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
)

type settingKind int

const (
	setInput settingKind = iota
	setToggle
	setAction // enter opens a sub-screen (e.g. folder list)
)

type settingField struct {
	kind    settingKind
	label   string
	hint    string
	input   textinput.Model
	options []string // for toggles: cycle order
	apply   func(m *Model, val string)
	value   func(m *Model) string
	display func(m *Model) string // optional pretty label; cycling uses value
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

// cookieBrowserOptions is the cycle for the cookies row: none → detected
// browsers → the common names yt-dlp supports, deduped.
func cookieBrowserOptions() []string {
	out := []string{""}
	seen := map[string]bool{"": true}
	for _, v := range ytdlp.DetectBrowsers() {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	for _, name := range []string{"firefox", "chrome", "brave", "edge", "chromium", "vivaldi", "opera", "safari"} {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

func cookieValueLabel(v string) string {
	if v == "" {
		return "none"
	}
	return v
}

func cookieBrowserHint() string {
	detected := ytdlp.DetectBrowsers()
	if len(detected) == 0 {
		return "no browser cookie stores found — export a cookies.txt and use Cookies file"
	}
	short := detected[0]
	if i := strings.Index(short, ":"); i >= 0 {
		short = short[:i] + ":<profile>"
	}
	return "detected: " + short + " · safari needs Full Disk Access"
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
			kind:  setAction,
			label: "Download folders",
			hint:  "enter to manage · first is the default · up to 5",
			value: func(m *Model) string {
				n := len(m.cfg.DownloadDirs)
				if n == 1 {
					return m.cfg.DownloadDirs[0]
				}
				return fmt.Sprintf("%d folders · default %s", n, m.cfg.DownloadDirs[0])
			},
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
		{
			kind:    setToggle,
			label:   "Cookies (browser)",
			hint:    cookieBrowserHint(),
			options: cookieBrowserOptions(),
			apply:   func(m *Model, val string) { m.cfg.CookiesFromBrowser = val },
			value:   func(m *Model) string { return m.cfg.CookiesFromBrowser },
			display: func(m *Model) string { return cookieValueLabel(m.cfg.CookiesFromBrowser) },
		},
		{
			kind:  setInput,
			label: "Cookies file",
			hint:  "exported cookies.txt — used only when no browser is set",
			apply: func(m *Model, val string) { m.cfg.CookiesFile = val },
			value: func(m *Model) string { return m.cfg.CookiesFile },
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
	case "up", "ctrl+k", "ctrl+p":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "ctrl+j", "ctrl+n":
		if s.cursor < len(s.fields)-1 {
			s.cursor++
		}
	case "enter", " ":
		f := &s.fields[s.cursor]
		switch f.kind {
		case setInput:
			s.editing = true
			return m, f.input.Focus()
		case setAction:
			m.enterFolders()
			return m, nil
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
	for _, dir := range m.cfg.DownloadDirs {
		if err := createDir(dir); err != nil {
			return m.setStatus("cannot create download folder: "+err.Error(), true)
		}
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
			if f.display != nil {
				v = f.display(m)
			}
			st := styles.RowDesc
			if v != "off" && v != "best" && v != "none" {
				st = styles.Checkmark
			}
			value = st.Render(truncate(v, maxInt(m.width-42, 12)))
		case setAction:
			value = styles.Checkmark.Render(truncate(f.value(m), maxInt(m.width-42, 12)))
		}

		line := cursor + styles.InfoLabel.Render(f.label) + "  " + value
		if i == s.cursor {
			pad := m.width - 6 - lipgloss.Width(cursor+styles.InfoLabel.Render(f.label)) - lipgloss.Width(f.input.Value())
			if f.kind != setInput {
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
