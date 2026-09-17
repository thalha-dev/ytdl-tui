package ui

import (
	"fmt"

	"github.com/thalha-dev/ytdl-tui/internal/ui/components"
)

// buildPlaylistPicker creates the multi-select picker over playlist entries.
func (m *Model) buildPlaylistPicker() {
	items := make([]components.Item, 0, len(m.info.Entries))
	for _, e := range m.info.Entries {
		title := e.Title
		if title == "" {
			title = "[untitled · " + e.ID + "]"
		}
		desc := formatDuration(e.Duration)
		if ch := e.DisplayChannel(); ch != "" {
			if desc != "" {
				desc += " · "
			}
			desc += ch
		}
		items = append(items, components.Item{Title: title, Desc: desc, Data: e})
	}

	m.playlistPicker = components.NewPicker(
		fmt.Sprintf("%d videos — mark the ones to download", len(m.info.Entries)),
		items,
		true,
	)
	m.playlistPicker.Hint = "type to search · tab toggles · ctrl+a all/none · enter downloads the selection"
}
