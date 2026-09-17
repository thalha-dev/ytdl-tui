// Package ui implements the ytdl-tui Bubble Tea application: the root
// model, screen routing, key handling and the shared header/footer chrome.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ui/components"
	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
)

type screen int

const (
	screenURL        screen = iota
	screenMenu              // video card + media menu, or playlist menu
	screenFamily            // codec / container picker
	screenResolution        // resolution picker
	screenAudio             // audio format picker
	screenExpert            // raw yt-dlp format picker
	screenPlaylist          // multi-select playlist entries
	screenDownload
	screenSettings
)

// Model is the root Bubble Tea model.
type Model struct {
	program *tea.Program // used to forward yt-dlp output events
	cfg     *config.Config
	cfgPath string
	version string

	screen screen
	width  int
	height int

	urlInput textinput.Model
	spinner  spinner.Model
	probing  bool
	urlErr   string

	probeSeq    int // bumped on cancel so late probe results are ignored
	probeCancel context.CancelFunc
	info        *ytdlp.Info

	menuPicker     components.Picker
	familyPicker   components.Picker
	resPicker      components.Picker
	audioPicker    components.Picker
	expertPicker   components.Picker
	playlistPicker components.Picker

	dl       dlState
	settings settingsState

	chosenFamily ytdlp.VideoFamily
	chosenRes    ytdlp.ResolutionOption
	chosenAudio  ytdlp.AudioOption
	chosenExpert ytdlp.Format

	status      string
	statusIsErr bool
	showHelp    bool
}

// --- messages ---

type probeDoneMsg struct {
	info *ytdlp.Info
	err  error
	seq  int
}

type dlEventMsg struct{ ev ytdlp.Event }

type dlDoneMsg struct{ err error }

type clearStatusMsg struct{}

// NewModel builds the root model.
func NewModel(cfg config.Config, cfgPath, version string) *Model {
	ti := textinput.New()
	ti.Placeholder = "Paste a YouTube link (video or playlist)…"
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Faint)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Red)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(styles.RedBright)
	ti.Focus()
	ti.CharLimit = 1024
	ti.Width = 80

	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.Red)),
	)

	m := &Model{
		cfg:      &cfg,
		cfgPath:  cfgPath,
		version:  version,
		urlInput: ti,
		spinner:  sp,
		settings: newSettingsState(&cfg),
		dl:       dlState{bar: progress.New(progress.WithScaledGradient("#FF0033", "#FF8A6B"), progress.WithoutPercentage())},
	}
	return m
}

// Run starts the TUI and blocks until the user quits.
func Run(cfg config.Config, cfgPath, version string) error {
	m := NewModel(cfg, cfgPath, version)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p
	_, err := p.Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.dl.bar.Width = clampInt(msg.Width-16, 20, 64)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case probeDoneMsg:
		if msg.seq != m.probeSeq {
			return m, nil // stale result from a canceled probe
		}
		m.probing = false
		m.probeCancel = nil
		if msg.err != nil {
			m.urlErr = msg.err.Error()
			return m, nil
		}
		m.onProbed(msg.info)
		return m, nil

	case dlEventMsg:
		m.dl.applyEvent(msg.ev)
		return m, nil

	case dlDoneMsg:
		m.dl.finish(msg.err)
		return m, nil

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		if m.dl.running && m.dl.cancel != nil {
			m.dl.cancel()
		}
		return m, tea.Quit
	}
	if key == "?" && m.screen != screenURL && m.screen != screenSettings {
		m.showHelp = !m.showHelp
		return m, nil
	}
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	switch m.screen {
	case screenURL:
		return m.updateURL(key, msg)
	case screenMenu:
		return m.updateMenu(msg)
	case screenFamily, screenResolution, screenAudio, screenExpert, screenPlaylist:
		return m.updatePickers(msg)
	case screenDownload:
		return m.updateDownload(key)
	case screenSettings:
		return m.updateSettings(key, msg)
	}
	return m, nil
}

// --- probing ---

func (m *Model) startProbe(url string) tea.Cmd {
	m.urlErr = ""
	m.probing = true
	seq := m.probeSeq + 1
	m.probeSeq = seq
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	m.probeCancel = cancel
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		info, err := ytdlp.Probe(ctx, url)
		return probeDoneMsg{info: info, err: err, seq: seq}
	})
}

func (m *Model) onProbed(info *ytdlp.Info) {
	m.info = info

	// A "playlist" with exactly one entry is just a video: re-probe it.
	if info.IsPlaylist() && len(info.Entries) == 1 {
		u := info.Entries[0].WebpageURL
		if u == "" && info.Entries[0].ID != "" {
			u = "https://www.youtube.com/watch?v=" + info.Entries[0].ID
		}
		if u != "" {
			m.startProbe(u)
			return
		}
	}

	if info.IsPlaylist() {
		m.buildPlaylistMenu()
	} else {
		m.buildVideoMenu()
	}
	m.screen = screenMenu
}

// --- downloads ---

func (m *Model) launch(req ytdlp.Request, title, subtitle string) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.dl = newDLState(cancel, m.dl.bar.Width)
	m.dl.title = title
	m.dl.subtitle = subtitle
	m.screen = screenDownload

	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		err := ytdlp.Download(ctx, req, func(ev ytdlp.Event) {
			if m.program != nil {
				m.program.Send(dlEventMsg{ev: ev})
			}
		})
		return dlDoneMsg{err: err}
	})
}

func (m *Model) baseRequest(urls []string) ytdlp.Request {
	return ytdlp.Request{
		URLs:           urls,
		OutputTemplate: m.cfg.ResolveOutput(),
		MergeFormat:    m.cfg.MergeFormat,
		AudioFormat:    m.cfg.AudioFormat,
		EmbedThumbnail: m.cfg.EmbedThumbnail,
		EmbedMetadata:  m.cfg.EmbedMetadata,
		Concurrency:    m.cfg.ConcurrentFragments,
	}
}

// setStatus shows a transient toast.
func (m *Model) setStatus(s string, isErr bool) tea.Cmd {
	m.status = s
	m.statusIsErr = isErr
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

// --- view chrome ---

func (m *Model) header() string {
	left := lipgloss.JoinHorizontal(lipgloss.Center,
		styles.Logo.Render("▶"),
		" ", styles.AppName.Render("yt-dlp tui"),
		"  ", styles.HeaderMeta.Render("v"+m.version),
	)
	right := styles.HeaderMeta.Render("⬇ " + truncate(m.cfg.DownloadDir, maxInt(m.width-46, 12)))
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *Model) footer() string {
	switch m.screen {
	case screenURL:
		if m.probing {
			return styles.HelpLine([2]string{"esc", "cancel"})
		}
		return styles.HelpLine(
			[2]string{"enter", "fetch formats"},
			[2]string{"ctrl+s", "settings"},
			[2]string{"esc", "clear / quit"},
		)
	case screenMenu:
		return styles.HelpLine(
			[2]string{"type", "filter"},
			[2]string{"↑↓", "move"},
			[2]string{"enter", "select"},
			[2]string{"esc", "new link"},
			[2]string{"?", "help"},
		)
	case screenPlaylist:
		return styles.HelpLine(
			[2]string{"type", "filter"},
			[2]string{"tab", "select"},
			[2]string{"ctrl+a", "all/none"},
			[2]string{"enter", "download selected"},
			[2]string{"esc", "back"},
		)
	case screenFamily, screenResolution, screenAudio, screenExpert:
		return styles.HelpLine(
			[2]string{"type", "filter"},
			[2]string{"↑↓", "move"},
			[2]string{"enter", "select"},
			[2]string{"esc", "back"},
		)
	case screenDownload:
		if m.dl.running {
			return styles.HelpLine([2]string{"c", "cancel"}, [2]string{"ctrl+c", "quit"})
		}
		if m.dl.err == "" {
			return styles.HelpLine(
				[2]string{"enter", "new download"},
				[2]string{"o", "open folder"},
				[2]string{"ctrl+c", "quit"},
			)
		}
		return styles.HelpLine([2]string{"enter", "back"}, [2]string{"ctrl+c", "quit"})
	case screenSettings:
		return styles.HelpLine(
			[2]string{"↑↓", "navigate"},
			[2]string{"enter", "edit / toggle"},
			[2]string{"s", "save"},
			[2]string{"esc", "back"},
		)
	}
	return ""
}

func (m *Model) View() string {
	if m.width == 0 {
		return "loading…"
	}

	var content string
	if m.showHelp {
		content = m.helpView()
	} else {
		switch m.screen {
		case screenURL:
			content = m.viewURL()
		case screenMenu:
			content = m.viewMenu()
		case screenFamily:
			content = m.viewPicker(&m.familyPicker)
		case screenResolution:
			content = m.viewPicker(&m.resPicker)
		case screenAudio:
			content = m.viewPicker(&m.audioPicker)
		case screenExpert:
			content = m.viewPicker(&m.expertPicker)
		case screenPlaylist:
			content = m.viewPicker(&m.playlistPicker)
		case screenDownload:
			content = m.viewDownload()
		case screenSettings:
			content = m.viewSettings()
		}
	}

	if m.status != "" {
		st := styles.StatusText
		if m.statusIsErr {
			st = styles.ErrorText
		}
		content += "\n\n" + st.Render(truncate(m.status, m.width-4))
	}

	// Pin the footer to the bottom line.
	footer := m.footer()
	lines := strings.Count(content, "\n") + 1 + 2 // + header + blank
	if pad := m.height - lines - 1; pad > 0 {
		content += strings.Repeat("\n", pad+1)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(),
		"",
		lipgloss.NewStyle().Padding(0, 2).Render(content),
		lipgloss.NewStyle().Padding(0, 2).Render(footer),
	)
}

// viewPicker renders a full-screen picker with the context title above it.
func (m *Model) viewPicker(p *components.Picker) string {
	var b strings.Builder
	if m.info != nil {
		b.WriteString(styles.CardTitle.Render("▍ "+truncate(m.info.Title, m.width-8)) + "\n")
		b.WriteString(styles.RowDesc.Render(m.contextSubtitle()) + "\n\n")
	}
	b.WriteString(p.View(m.width - 6))
	return b.String()
}

// contextSubtitle describes what is being picked.
func (m *Model) contextSubtitle() string {
	if m.info == nil {
		return ""
	}
	if m.info.IsPlaylist() {
		return fmt.Sprintf("%d videos", len(m.info.Entries))
	}
	return m.videoMetaLine()
}

// --- small helpers ---

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// truncate cuts a plain string to w cells, appending an ellipsis.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}
