package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
)

// helpView renders the keyboard shortcut overlay.
func (m *Model) helpView() string {
	type row struct{ key, desc string }

	sections := []struct {
		title string
		rows  []row
	}{
		{"Global", []row{
			{"ctrl+c", "quit (cancels a running download)"},
			{"?", "toggle this help"},
		}},
		{"URL screen", []row{
			{"enter", "fetch info for the pasted link"},
			{"ctrl+s", "open settings"},
			{"esc", "clear input · quit when empty"},
		}},
		{"Pickers", []row{
			{"a–z", "type to fuzzy filter (fzf ranking)"},
			{"↑↓ / ctrl+j k", "move cursor"},
			{"enter", "select"},
			{"tab", "toggle selection (multi-select)"},
			{"ctrl+a", "select all / none (multi-select)"},
			{"esc", "clear filter, then go back"},
		}},
		{"Download", []row{
			{"c", "cancel (press twice)"},
			{"o", "open folder (when finished)"},
			{"enter", "back to the URL screen"},
		}},
		{"Settings", []row{
			{"↑↓", "navigate fields"},
			{"enter", "edit / cycle value"},
			{"s", "save"},
			{"esc", "back without saving"},
		}},
	}

	var b strings.Builder
	b.WriteString(styles.CardTitle.Render("▍ Keyboard") + "\n\n")
	for i, sec := range sections {
		b.WriteString("  " + styles.Checkmark.Render(sec.title) + "\n")
		for _, r := range sec.rows {
			k := styles.HelpKey.Render(r.key)
			b.WriteString("    " + k + strings.Repeat(" ", maxInt(16-lipgloss.Width(r.key), 2)) +
				styles.HelpDesc.Render(r.desc) + "\n")
		}
		if i < len(sections)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n  " + styles.HelpDesc.Render("press any key to close") + "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.RedBorder).
		Padding(0, 2).
		Render(b.String())
}
