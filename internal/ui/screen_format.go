package ui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/ui/components"
	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
)

// buildFamilyPicker offers codec/container choices derived from the formats.
func (m *Model) buildFamilyPicker() {
	formats := m.info.DownloadableFormats()
	fams := ytdlp.CodecFamilies(formats)

	items := make([]components.Item, 0, len(fams))
	for _, fam := range fams {
		desc := fam.Desc
		if fam.ID != "auto" {
			if res := ytdlp.Resolutions(formats, fam.ID, m.info.Duration); len(res) > 0 {
				desc = fam.Desc + " · " + strconv.Itoa(len(res)) + " resolutions"
			} else {
				continue // no downloadable formats in this family
			}
		}
		items = append(items, components.Item{Title: fam.Label, Desc: desc, Data: fam})
	}
	m.familyPicker = components.NewPicker("Pick a codec / container", items, false)
}

// buildResPicker offers resolutions within the chosen codec family.
func (m *Model) buildResPicker(fam ytdlp.VideoFamily) {
	formats := m.info.DownloadableFormats()
	opts := ytdlp.Resolutions(formats, fam.ID, m.info.Duration)

	items := make([]components.Item, 0, len(opts)+1)
	items = append(items, components.Item{
		Title: "Best of " + fam.Label,
		Desc:  "highest resolution this codec offers",
		Data:  nil,
	})
	for _, o := range opts {
		title := strconv.Itoa(o.Height) + "p"
		if o.FPS > 0 {
			title += strconv.Itoa(o.FPS)
		}
		f := o.Format
		desc := sizeDesc(o.Size) + " · " + f.VideoCodecName() + " · " + f.Ext + " · id " + f.FormatID
		items = append(items, components.Item{Title: title, Desc: desc, Data: o})
	}
	m.resPicker = components.NewPicker("Pick a resolution", items, false)
}

// buildAudioPicker offers audio-only tracks.
func (m *Model) buildAudioPicker() {
	formats := m.info.DownloadableFormats()
	opts := ytdlp.AudioFormats(formats)

	items := make([]components.Item, 0, len(opts))
	for _, o := range opts {
		if o.IsBest {
			conv := ""
			if m.cfg.AudioFormat != "best" {
				conv = " · converts to " + m.cfg.AudioFormat
			}
			items = append(items, components.Item{
				Title: "Best available",
				Desc:  "yt-dlp picks the best audio track" + conv,
				Data:  o,
			})
			continue
		}
		f := o.Format
		title := f.AudioCodecName()
		if f.ABR > 0 {
			title += " · " + strconv.Itoa(int(f.ABR)) + " kbps"
		}
		desc := sizeDesc(f.EstimatedSize(m.info.Duration)) + " · " + f.Ext + " · id " + f.FormatID
		if m.cfg.AudioFormat != "best" {
			desc += " · → " + m.cfg.AudioFormat
		}
		items = append(items, components.Item{Title: title, Desc: desc, Data: o})
	}
	m.audioPicker = components.NewPicker("Pick an audio format", items, false)
}

// buildExpertPicker lists every raw format.
func (m *Model) buildExpertPicker() {
	formats := m.info.DownloadableFormats()
	items := make([]components.Item, 0, len(formats))
	for _, f := range formats {
		title := fmt.Sprintf("[%s] %s", f.FormatID, f.Ext)
		var desc string
		switch {
		case f.IsVideoOnly():
			title += " · " + resLabel(f)
			desc = f.VideoCodecName() + " · video-only · " + sizeDesc(f.EstimatedSize(m.info.Duration)) + " (+ audio)"
		case f.IsAudioOnly():
			title += " · audio"
			desc = f.AudioCodecName()
			if f.ABR > 0 {
				desc += fmt.Sprintf(" · %dkbps", int(f.ABR))
			}
			desc += " · " + sizeDesc(f.EstimatedSize(m.info.Duration))
		default:
			title += " · " + resLabel(f)
			desc = "video+audio · " + sizeDesc(f.EstimatedSize(m.info.Duration))
		}
		items = append(items, components.Item{Title: title, Desc: desc, Data: f})
	}
	m.expertPicker = components.NewPicker("Raw formats — pick exactly one", items, false)
}

func resLabel(f ytdlp.Format) string {
	if f.Height <= 0 {
		if f.Resolution != "" {
			return f.Resolution
		}
		return "?"
	}
	s := strconv.Itoa(f.Height) + "p"
	if f.FPS > 0 {
		s += strconv.Itoa(int(f.FPS))
	}
	return s
}

// updatePickers routes keys to the picker for the active screen and
// processes selections.
func (m *Model) updatePickers(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenFamily:
		m.familyPicker.Update(msg)
		if items, ok := m.familyPicker.ConsumeSelected(); ok {
			fam := items[0].Data.(ytdlp.VideoFamily)
			m.chosenFamily = fam
			if fam.ID == "auto" {
				return m, m.startVideoDownload("bv*+ba/b", "")
			}
			m.buildResPicker(fam)
			if m.resPicker.MatchCount() == 0 {
				return m, m.startVideoDownload("bv*+ba/b", "")
			}
			m.screen = screenResolution
			return m, nil
		}
		if m.familyPicker.ConsumeCancel() {
			m.screen = screenMenu
		}
		return m, nil

	case screenResolution:
		m.resPicker.Update(msg)
		if items, ok := m.resPicker.ConsumeSelected(); ok {
			if items[0].Data == nil {
				// "Best of <codec>": highest listed option
				best := ytdlp.Resolutions(m.info.DownloadableFormats(), m.chosenFamily.ID, m.info.Duration)
				if len(best) > 0 {
					m.chosenRes = best[0]
					return m, m.startVideoDownload(ytdlp.FormatSelector(best[0].Format), "")
				}
				return m, m.startVideoDownload("bv*+ba/b", "")
			}
			opt := items[0].Data.(ytdlp.ResolutionOption)
			m.chosenRes = opt
			return m, m.startVideoDownload(ytdlp.FormatSelector(opt.Format), "")
		}
		if m.resPicker.ConsumeCancel() {
			m.screen = screenFamily
		}
		return m, nil

	case screenAudio:
		m.audioPicker.Update(msg)
		if items, ok := m.audioPicker.ConsumeSelected(); ok {
			opt := items[0].Data.(ytdlp.AudioOption)
			m.chosenAudio = opt
			selector := "bestaudio/best"
			if !opt.IsBest {
				selector = opt.Format.FormatID
			}
			return m, m.startAudioDownload(selector)
		}
		if m.audioPicker.ConsumeCancel() {
			m.screen = screenMenu
		}
		return m, nil

	case screenExpert:
		m.expertPicker.Update(msg)
		if items, ok := m.expertPicker.ConsumeSelected(); ok {
			f := items[0].Data.(ytdlp.Format)
			m.chosenExpert = f
			return m, m.startVideoDownload(ytdlp.FormatSelector(f), "")
		}
		if m.expertPicker.ConsumeCancel() {
			m.screen = screenMenu
		}
		return m, nil

	case screenPlaylist:
		m.playlistPicker.Update(msg)
		if items, ok := m.playlistPicker.ConsumeSelected(); ok {
			entries := make([]*ytdlp.Info, 0, len(items))
			for _, it := range items {
				if e, ok := it.Data.(*ytdlp.Info); ok {
					entries = append(entries, e)
				}
			}
			return m, m.startPlaylistDownload(false, entries)
		}
		if m.playlistPicker.ConsumeCancel() {
			m.screen = screenMenu
		}
		return m, nil
	}
	return m, nil
}

// --- launchers ---

func (m *Model) startVideoDownload(selector, qualityCap string) tea.Cmd {
	req := m.baseRequest([]string{m.info.WebpageURL})
	req.FormatSelector = selector
	req.QualityCap = qualityCap

	subtitle := describeFormat(m, selector)
	return m.offerSaveTo(req, m.info.Title, subtitle)
}

func (m *Model) startAudioDownload(selector string) tea.Cmd {
	req := m.baseRequest([]string{m.info.WebpageURL})
	req.FormatSelector = selector
	req.AudioOnly = true

	sub := "audio only"
	if m.cfg.AudioFormat != "best" {
		sub += " · → " + m.cfg.AudioFormat
	}
	return m.offerSaveTo(req, m.info.Title, sub)
}

// startPlaylistDownload launches a playlist download; when entries is nil
// the whole playlist is delegated to yt-dlp, otherwise only the picked URLs.
func (m *Model) startPlaylistDownload(audioOnly bool, entries []*ytdlp.Info) tea.Cmd {
	urls := []string{m.info.WebpageURL}
	n := len(m.info.Entries)
	if entries != nil {
		urls = make([]string, 0, len(entries))
		for _, e := range entries {
			if u := playlistEntryURL(e); u != "" {
				urls = append(urls, u)
			}
		}
		n = len(entries)
	}

	req := m.baseRequest(urls)
	title := m.info.Title
	var sub string
	if audioOnly {
		req.FormatSelector = "bestaudio/best"
		req.AudioOnly = true
		sub = fmt.Sprintf("%d tracks · audio → %s", n, m.cfg.AudioFormat)
	} else {
		req.FormatSelector = "bv*+ba/b"
		req.QualityCap = m.cfg.PlaylistQuality
		capQ := m.cfg.PlaylistQuality
		if capQ == "best" {
			capQ = "best"
		} else {
			capQ = "≤ " + capQ + "p"
		}
		sub = fmt.Sprintf("%d videos · %s · %s", n, capQ, m.cfg.MergeFormat)
	}
	return m.offerSaveTo(req, title, sub)
}

// describeFormat renders the chosen format for the download header.
func describeFormat(m *Model, selector string) string {
	f := m.chosenRes.Format
	if f.FormatID == "" {
		f = m.chosenExpert
	}
	if f.FormatID == "" {
		return selector
	}
	parts := []string{}
	if f.IsVideo() {
		if l := resLabel(f); l != "?" {
			parts = append(parts, l)
		}
		parts = append(parts, f.VideoCodecName())
		parts = append(parts, f.Ext)
	} else {
		parts = append(parts, f.AudioCodecName(), f.Ext)
	}
	if s := humanBytes(m.chosenRes.Size); s != "" && m.chosenRes.Size > 0 {
		parts = append(parts, "≈ "+s)
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " · "
		}
		out += p
	}
	return out
}
