// Package components holds reusable Bubble Tea widgets for ytdl-tui.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thalha-dev/ytdl-tui/internal/fzf"
	"github.com/thalha-dev/ytdl-tui/internal/ui/styles"
)

// Item is one selectable row in a Picker.
type Item struct {
	Title string // primary text (filterable)
	Desc  string // dim secondary text (also filterable)
	Data  any    // caller payload
}

// Picker is a keyboard-driven fuzzy list backed by fzf's ranking engine.
// It supports single and multi selection and incremental type-to-filter.
//
// Pickers are not standalone tea.Models: screens embed them and forward
// messages so global keys stay in one place.
type Picker struct {
	Prompt     string // e.g. "Pick a codec"
	Hint       string // dim one-liner under the prompt
	Multi      bool
	MaxVisible int // rows shown at once; 0 = no windowing

	items   []Item
	matches []fzf.Match
	cursor  int
	query   string
	checked map[int]bool // item index -> selected (multi mode)
	width   int

	// Pending results consumed by the owning screen via ConsumeSelect /
	// ConsumeCancel.
	selectPending bool
	cancelPending bool
}

// NewPicker creates a picker for the given items.
func NewPicker(prompt string, items []Item, multi bool) Picker {
	p := Picker{
		Prompt:     prompt,
		Multi:      multi,
		MaxVisible: 12,
		checked:    map[int]bool{},
	}
	p.SetItems(items)
	return p
}

// SetItems replaces the item list and resets filtering.
func (p *Picker) SetItems(items []Item) {
	p.items = items
	p.query = ""
	p.cursor = 0
	p.checked = map[int]bool{}
	p.refilter()
}

// SetWidth stores the render width for row alignment.
func (p *Picker) SetWidth(w int) { p.width = w }

// Query returns the current filter text.
func (p *Picker) Query() string { return p.query }

// MatchCount returns how many items currently match the filter.
func (p *Picker) MatchCount() int { return len(p.matches) }

// CheckedCount returns how many items are ticked (multi mode).
func (p *Picker) CheckedCount() int { return len(p.checked) }

// ConsumeSelected reports whether Enter was pressed and returns the
// selected items (in list order for singles; insertion order for multis).
func (p *Picker) ConsumeSelected() ([]Item, bool) {
	if !p.selectPending {
		return nil, false
	}
	p.selectPending = false
	if !p.Multi {
		if !p.cursorValid() {
			return nil, false
		}
		return []Item{p.items[p.matches[p.cursor].Index]}, true
	}
	idxs := make([]int, 0, len(p.checked))
	for i := range p.checked {
		idxs = append(idxs, i)
	}
	items := make([]Item, 0, len(idxs))
	for _, i := range idxs {
		items = append(items, p.items[i])
	}
	return items, true
}

// ConsumeCancel reports whether Esc was pressed with an empty filter.
func (p *Picker) ConsumeCancel() bool {
	if !p.cancelPending {
		return false
	}
	p.cancelPending = false
	return true
}

// Update handles key messages. Unrecognized keys are ignored so the owning
// screen can process them too.
func (p *Picker) Update(msg tea.Msg) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return
	}
	switch key.String() {
	case "up", "ctrl+k", "ctrl+p":
		p.move(-1)
	case "down", "ctrl+j", "ctrl+n":
		p.move(1)
	case "pgup", "ctrl+u":
		p.move(-8)
	case "pgdown", "ctrl+d":
		p.move(8)
	case "home":
		p.cursor = 0
	case "end":
		p.cursor = len(p.matches) - 1
	case "backspace", "ctrl+h":
		if r := []rune(p.query); len(r) > 0 {
			p.query = string(r[:len(r)-1])
			p.refilter()
		}
	case "tab":
		if p.Multi && p.cursorValid() {
			idx := p.matches[p.cursor].Index
			p.checked[idx] = !p.checked[idx]
			p.move(1)
		}
	case "ctrl+a":
		if p.Multi {
			all := len(p.checked) == len(p.items) && len(p.items) > 0
			p.checked = map[int]bool{}
			if !all {
				for i := range p.items {
					p.checked[i] = true
				}
			}
		}
	case "enter":
		if p.Multi && len(p.checked) == 0 && p.cursorValid() {
			p.checked[p.matches[p.cursor].Index] = true
		}
		p.selectPending = true
	case "esc":
		if p.query != "" {
			p.query = ""
			p.refilter()
		} else {
			p.cancelPending = true
		}
	default:
		// Printable input. Typed-fast or pasted text arrives as ONE
		// multi-rune KeyMsg, so append every rune, not just single keys.
		switch key.Type {
		case tea.KeyRunes:
			p.appendRunes(string(key.Runes))
		case tea.KeySpace:
			p.appendRunes(" ")
		}
	}
}

func (p *Picker) appendRunes(s string) {
	p.query += s
	p.refilter()
}

func (p *Picker) move(delta int) {
	n := len(p.matches)
	if n == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= n {
		p.cursor = n - 1
	}
}

func (p *Picker) cursorValid() bool {
	return len(p.matches) > 0 && p.cursor >= 0 && p.cursor < len(p.matches)
}

func (p *Picker) refilter() {
	texts := make([]string, len(p.items))
	for i, it := range p.items {
		texts[i] = it.Title + " " + it.Desc
	}
	p.matches = fzf.Filter(p.query, texts)
	if p.cursor >= len(p.matches) {
		p.cursor = len(p.matches) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// View renders the picker. width is the available content width.
func (p Picker) View(width int) string {
	var b strings.Builder

	counter := fmt.Sprintf("%d/%d", len(p.matches), len(p.items))
	if p.Multi {
		counter += fmt.Sprintf(" · %d selected", len(p.checked))
	}

	header := styles.Prompt.Render(p.Prompt)
	if p.query != "" {
		header += "  " + styles.Query.Render(p.query) + "▏"
	}
	header += "  " + styles.Counter.Render(counter)
	b.WriteString(header + "\n")

	if p.Hint != "" {
		b.WriteString(styles.HelpDesc.Render(p.Hint) + "\n")
	}

	if len(p.matches) == 0 {
		b.WriteString(styles.NoMatch.Render("no matches for \""+p.query+"\"") + "\n")
		return b.String()
	}

	lo, hi := p.visibleWindow()
	// Column-align descriptions.
	tw := 0
	for _, m := range p.matches[lo:hi] {
		if w := lipgloss.Width(p.items[m.Index].Title); w > tw {
			tw = w
		}
	}
	if tw > width-12 {
		tw = max(width-12, 0)
	}

	innerW := width - 4
	if innerW < 20 {
		innerW = 20
	}

	if lo > 0 {
		b.WriteString(styles.HelpDesc.Render(fmt.Sprintf("  ↑ %d more", lo)) + "\n")
	}
	for i := lo; i < hi; i++ {
		m := p.matches[i]
		item := p.items[m.Index]
		b.WriteString(p.renderRow(i, item, i == p.cursor, tw, innerW))
		b.WriteString("\n")
	}
	if hi < len(p.matches) {
		b.WriteString(styles.HelpDesc.Render(fmt.Sprintf("  ↓ %d more", len(p.matches)-hi)))
	}
	return b.String()
}

func (p Picker) renderRow(i int, item Item, cursor bool, titleW, innerW int) string {
	var check string
	if p.Multi {
		if p.checked[fzfIndexOf(p, i)] {
			check = styles.Checkmark.Render("● ")
		} else {
			check = styles.HelpDesc.Render("○ ")
		}
	}

	title, desc := item.Title, item.Desc
	if pad := titleW - lipgloss.Width(title); pad > 0 {
		title += strings.Repeat(" ", pad)
	}
	line := check + title
	if desc != "" {
		line += "  " + desc
	}

	if over := lipgloss.Width(line) - innerW; over > 0 {
		line = truncateVisible(line, lipgloss.Width(line)-over)
	}

	if cursor {
		pad := innerW - lipgloss.Width(line)
		if pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		return styles.CursorRow.Render(line)
	}
	// Split already-styled check from plain text so desc stays dim.
	if p.Multi {
		plain := strings.TrimPrefix(line, check)
		if strings.Contains(plain, "  ") {
			parts := strings.SplitN(plain, "  ", 2)
			t, d := parts[0], strings.TrimPrefix(parts[1], " ")
			return check + styles.Row.Render(t) + "  " + styles.RowDesc.Render(d)
		}
		return check + styles.Row.Render(plain)
	}
	if strings.Contains(line, "  ") {
		parts := strings.SplitN(line, "  ", 2)
		t, d := parts[0], strings.TrimPrefix(parts[1], " ")
		return styles.Row.Render(t) + "  " + styles.RowDesc.Render(d)
	}
	return styles.Row.Render(line)
}

func fzfIndexOf(p Picker, matchPos int) int {
	return p.matches[matchPos].Index
}

func (p Picker) visibleWindow() (lo, hi int) {
	n := len(p.matches)
	max := p.MaxVisible
	if max <= 0 || n <= max {
		return 0, n
	}
	lo = p.cursor - max/2
	if lo < 0 {
		lo = 0
	}
	if lo > n-max {
		lo = n - max
	}
	return lo, lo + max
}

// truncateVisible cuts a styled line to w visible cells.
func truncateVisible(s string, w int) string {
	if w <= 0 {
		return ""
	}
	var b strings.Builder
	cw := 0
	inAnsi := false
	for _, r := range s {
		if r == '\x1b' {
			inAnsi = true
			b.WriteRune(r)
			continue
		}
		if inAnsi {
			b.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inAnsi = false
			}
			continue
		}
		if cw >= w {
			continue
		}
		b.WriteRune(r)
		cw++
	}
	return b.String()
}
