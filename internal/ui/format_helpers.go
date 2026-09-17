package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/thalha-dev/ytdl-tui/internal/ytdlp"
)

// formatDuration renders seconds as "0:27" / "1:02:33".
func formatDuration(secs float64) string {
	if secs <= 0 {
		return ""
	}
	total := int(secs)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// formatViews renders a view count like "1,307,404".
func formatViews(n int64) string {
	if n <= 0 {
		return ""
	}
	return humanize.Comma(n) + " views"
}

// formatUploadDate renders yt-dlp's "20060102" as "01 Apr 2023".
func formatUploadDate(s string) string {
	if len(s) != 8 {
		return ""
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return ""
	}
	return t.Format("02 Jan 2006")
}

// humanBytes renders a byte count like "12.2 MB" or "" for 0.
func humanBytes(n int64) string {
	if n <= 0 {
		return ""
	}
	return humanize.Bytes(uint64(n))
}

// videoMetaLine renders "Channel · 0:27 · 1,307,404 views · 01 Apr 2023".
func (m *Model) videoMetaLine() string {
	parts := []string{}
	if ch := m.info.DisplayChannel(); ch != "" {
		parts = append(parts, ch)
	}
	if d := formatDuration(m.info.Duration); d != "" {
		parts = append(parts, d)
	}
	if v := formatViews(m.info.ViewCount); v != "" {
		parts = append(parts, v)
	}
	if d := formatUploadDate(m.info.UploadDate); d != "" {
		parts = append(parts, d)
	}
	return strings.Join(parts, " · ")
}

// maxResLabel returns the highest available video resolution, e.g. "1080p".
func (m *Model) maxResLabel() string {
	best := 0
	for _, f := range m.info.DownloadableFormats() {
		if f.IsVideo() && f.Height > best {
			best = f.Height
		}
	}
	if best == 0 {
		return ""
	}
	return strconv.Itoa(best) + "p"
}

// sizeDesc renders "≈ 12.2 MB" or "size N/A".
func sizeDesc(n int64) string {
	if s := humanBytes(n); s != "" {
		return "≈ " + s
	}
	return "size N/A"
}

// playlistEntryURL builds a watch URL for a flat playlist entry.
func playlistEntryURL(e *ytdlp.Info) string {
	if e.WebpageURL != "" {
		return e.WebpageURL
	}
	return "https://www.youtube.com/watch?v=" + e.ID
}
