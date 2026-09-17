// Package styles centralises the lipgloss theme: a clean dark UI in
// YouTube-inspired shades of red.
package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette — YouTube reds on a near-black canvas.
var (
	Red       = lipgloss.Color("#FF0033") // primary brand red
	RedBright = lipgloss.Color("#FF5F5F") // highlights, keys, cursor
	RedDeep   = lipgloss.Color("#C4112D") // secondary accent
	RedDark   = lipgloss.Color("#3D1420") // selected-row background
	RedBorder = lipgloss.Color("#4A1A26") // card borders
	RedDim    = lipgloss.Color("#8A2438") // secondary text accent

	Text    = lipgloss.Color("#F2EEEF") // primary text
	Muted   = lipgloss.Color("#A79EA2") // secondary text
	Faint   = lipgloss.Color("#5E5458") // hints, footers
	White   = lipgloss.Color("#FFFFFF")
	Success = lipgloss.Color("#7FD98B") // success only
)

var (
	// Logo is the red play-button badge in the header.
	Logo = lipgloss.NewStyle().Bold(true).Foreground(White).Background(Red).Padding(0, 1)

	// AppName is the product name next to the logo.
	AppName = lipgloss.NewStyle().Bold(true).Foreground(Text)

	// HeaderMeta is dim info on the header row (version, download dir).
	HeaderMeta = lipgloss.NewStyle().Foreground(Faint)

	// Card wraps content sections in a rounded red-tinted border.
	Card = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(RedBorder).
		Padding(0, 1)

	// CardTitle prefixes a card ("▍Title" in red).
	CardTitle = lipgloss.NewStyle().Bold(true).Foreground(Red)

	// Prompt is the picker's question line.
	Prompt = lipgloss.NewStyle().Bold(true).Foreground(Text)

	// Query is the typed filter text.
	Query = lipgloss.NewStyle().Foreground(RedBright).Bold(true)

	// Counter is the "3/54 items" indicator.
	Counter = lipgloss.NewStyle().Foreground(Faint)

	// CursorRow is the highlighted (cursor) row.
	CursorRow = lipgloss.NewStyle().Bold(true).Foreground(White).Background(RedDark)

	// Row is a normal row.
	Row = lipgloss.NewStyle().Foreground(Text)

	// RowDesc is the dim description part of a row.
	RowDesc = lipgloss.NewStyle().Foreground(Muted)

	// CursorDesc keeps descriptions readable on the highlighted row.
	CursorDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5A9B4"))

	// Checkmark is the selected marker in multi-pickers.
	Checkmark = lipgloss.NewStyle().Bold(true).Foreground(RedBright)

	// NoMatch is the "no results" line.
	NoMatch = lipgloss.NewStyle().Foreground(Faint).Italic(true)

	// InfoValue is a bold value on the video card.
	InfoValue = lipgloss.NewStyle().Bold(true).Foreground(Text)

	// InfoLabel is a muted label on the video card.
	InfoLabel = lipgloss.NewStyle().Foreground(Muted)

	// ErrorText is user-facing errors.
	ErrorText = lipgloss.NewStyle().Bold(true).Foreground(RedBright)

	// SuccessText is user-facing success messages.
	SuccessText = lipgloss.NewStyle().Bold(true).Foreground(Success)

	// StatusText is transient toasts.
	StatusText = lipgloss.NewStyle().Foreground(RedDim)

	// PhaseText is the download-phase line.
	PhaseText = lipgloss.NewStyle().Foreground(RedDim).Italic(true)

	// LogText is the dim download log tail.
	LogText = lipgloss.NewStyle().Foreground(Faint)

	// HelpKey is a key name in the footer.
	HelpKey = lipgloss.NewStyle().Bold(true).Foreground(RedBright)

	// HelpDesc is a key description in the footer.
	HelpDesc = lipgloss.NewStyle().Foreground(Faint)

	// BigPercent is the large percentage during download.
	BigPercent = lipgloss.NewStyle().Bold(true).Foreground(RedBright)
)

// Help renders a "key description" pair for footers.
func Help(key, desc string) string {
	return HelpKey.Render(key) + " " + HelpDesc.Render(desc)
}

// HelpLine joins pairs with a dim separator.
func HelpLine(pairs ...[2]string) string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, Help(p[0], p[1]))
	}
	sep := HelpDesc.Render("  ·  ")
	return strings.Join(out, sep)
}
