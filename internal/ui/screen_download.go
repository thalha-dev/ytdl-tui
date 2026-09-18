package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
)

// dlState is the download screen's state.
type dlState struct {
	running  bool
	done     bool
	err      string
	canceled bool

	cancel        context.CancelFunc
	cancelConfirm bool

	title    string
	subtitle string

	bar     progress.Model
	percent float64
	speed   string
	eta     string
	total   string

	item  int
	items int

	phase    string
	dest     string
	dir      string
	log      []string
	files    int
	fullPath string
}

func newDLState(cancel context.CancelFunc, barWidth int) dlState {
	bar := progress.New(
		progress.WithScaledGradient("#FF0033", "#FF8A6B"),
		progress.WithoutPercentage(),
	)
	if barWidth > 0 {
		bar.Width = barWidth
	}
	return dlState{running: true, cancel: cancel, bar: bar}
}

func appendLog(log []string, line string) []string {
	log = append(log, line)
	if len(log) > 4 {
		log = log[len(log)-4:]
	}
	return log
}

func (d *dlState) applyEvent(ev ytdlp.Event) {
	switch ev.Kind {
	case ytdlp.EventProgress:
		d.percent = ev.Progress.Percent
		if ev.Progress.Speed != "" {
			d.speed = ev.Progress.Speed
		}
		if ev.Progress.ETA != "" {
			d.eta = ev.Progress.ETA
		}
		if ev.Progress.Total != "" {
			d.total = ev.Progress.Total
		}
	case ytdlp.EventPhase:
		d.phase = ev.Text
	case ytdlp.EventItem:
		d.item, d.items = ev.Item, ev.Items
	case ytdlp.EventDest:
		d.dest = ev.Text
		d.log = appendLog(d.log, "↓ "+ev.Text)
	case ytdlp.EventAlready:
		d.files++
		d.log = appendLog(d.log, "= "+ev.Text+" (already downloaded)")
	case ytdlp.EventFinalPath:
		d.fullPath = ev.Text
		d.files++
	}
}

func (d *dlState) finish(err error) {
	d.running = false
	d.done = true
	if err != nil {
		if err == context.Canceled {
			d.canceled = true
		} else {
			d.err = err.Error()
		}
	}
}

// updateDownload handles keys on the download screen.
func (m *Model) updateDownload(key string) (tea.Model, tea.Cmd) {
	d := &m.dl

	switch key {
	case "c":
		if d.running {
			if d.cancelConfirm {
				d.cancelConfirm = false
				return m, func() tea.Msg {
					if d.cancel != nil {
						d.cancel()
					}
					return nil
				}
			}
			d.cancelConfirm = true
			return m, m.setStatus("Press c again to cancel the download", false)
		}

	case "o":
		if d.done && d.err == "" {
			dir := d.dir
			if dir == "" {
				dir = m.cfg.DownloadDirs[0]
			}
			if d.files == 1 && d.fullPath != "" {
				dir = filepath.Dir(d.fullPath)
			}
			var open *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				open = exec.Command("open", dir)
			case "windows":
				open = exec.Command("explorer", dir)
			default:
				open = exec.Command("xdg-open", dir)
			}
			return m, tea.ExecProcess(open, func(error) tea.Msg { return nil })
		}

	case "enter", "esc":
		if !d.running {
			m.screen = screenURL
			m.info = nil
			return m, m.urlInput.Focus()
		}
	}
	return m, nil
}

func (m *Model) viewDownload() string {
	d := &m.dl
	var b strings.Builder

	b.WriteString(styles.CardTitle.Render("▍ Download") + "\n\n")
	b.WriteString("  " + styles.InfoValue.Render(truncate(d.title, m.width-8)) + "\n")
	if d.subtitle != "" {
		b.WriteString("  " + styles.RowDesc.Render(d.subtitle) + "\n")
	}
	b.WriteString("\n")

	if d.done {
		switch {
		case d.canceled:
			b.WriteString("  " + styles.PhaseText.Render("download canceled") + "\n")
		case d.err != "":
			b.WriteString("  " + styles.ErrorText.Render("✗ download failed") + "\n")
			b.WriteString("  " + styles.RowDesc.Render(truncate(d.err, m.width-6)) + "\n")
		default:
			if d.files > 1 {
				b.WriteString("  " + styles.SuccessText.Render(fmt.Sprintf("✓ %d files saved to ", d.files)) +
					styles.InfoValue.Render(d.dir) + "\n")
			} else if d.fullPath != "" {
				b.WriteString("  " + styles.SuccessText.Render("✓ saved to ") + styles.InfoValue.Render(d.fullPath) + "\n")
			} else {
				b.WriteString("  " + styles.SuccessText.Render("✓ saved to ") + styles.InfoValue.Render(d.dir) + "\n")
			}
		}
		if len(d.log) > 0 {
			b.WriteString("\n  " + styles.LogText.Render(strings.Join(d.log, "\n  ")) + "\n")
		}
		return b.String()
	}

	// live progress
	b.WriteString("  " + d.bar.ViewAs(d.percent/100) + "\n")

	stat := fmt.Sprintf("%5.1f%%", d.percent)
	details := []string{}
	if d.speed != "" {
		details = append(details, d.speed)
	}
	if d.eta != "" {
		details = append(details, "ETA "+d.eta)
	}
	if d.total != "" {
		details = append(details, d.total)
	}
	b.WriteString("  " + styles.BigPercent.Render(stat))
	if len(details) > 0 {
		b.WriteString("   " + styles.RowDesc.Render(strings.Join(details, "  ·  ")))
	}
	b.WriteString("\n")

	if d.items > 0 {
		b.WriteString("  " + styles.RowDesc.Render(fmt.Sprintf("item %d of %d", maxInt(d.item, 1), d.items)) + "\n")
	}

	phase := d.phase
	if phase == "" {
		phase = "Downloading"
	}
	if d.percent >= 100 && d.phase == "" {
		phase = "Processing"
	}
	b.WriteString("  " + m.spinner.View() + " " + styles.PhaseText.Render(phase+"…") + "\n")

	if d.dest != "" {
		b.WriteString("  " + styles.LogText.Render(truncate(d.dest, m.width-8)) + "\n")
	}
	if d.cancelConfirm {
		b.WriteString("\n  " + styles.ErrorText.Render("press c again to cancel") + "\n")
	}

	return b.String()
}
