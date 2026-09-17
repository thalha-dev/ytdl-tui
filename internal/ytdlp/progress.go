package ytdlp

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const progressPrefix = "PROGRESS|"

var (
	itemRe    = regexp.MustCompile(`\[download\] Downloading item (\d+) of (\d+)`)
	destRe    = regexp.MustCompile(`\[download\] Destination: (.+)`)
	alreadyRe = regexp.MustCompile(`\[download\] (.+) has already been downloaded`)
	phaseRe   = regexp.MustCompile(`^\[(Merger|ExtractAudio|EmbedThumbnail|Metadata|VideoRemuxer|VideoConvertor|FixupM4a|FixupM3u8|VideoRecoder)\]`)
	// Fallback for yt-dlp builds that ignore the progress template:
	// "[download]  45.2% of    3.00MiB at    1.50MiB/s ETA 00:02"
	classicRe = regexp.MustCompile(`^\[download\]\s+([\d.]+)% of\s+~?\s*([\d.]+)(\w+)(?:\s+at\s+(\S+))?(?:\s+ETA\s+(\S+))?`)
)

var phaseLabels = map[string]string{
	"Merger":         "Merging video + audio",
	"ExtractAudio":   "Extracting audio",
	"EmbedThumbnail": "Embedding thumbnail",
	"Metadata":       "Embedding metadata",
	"VideoRemuxer":   "Remuxing",
	"VideoConvertor": "Converting",
	"FixupM4a":       "Fixing up file",
	"FixupM3u8":      "Fixing up file",
	"VideoRecoder":   "Re-encoding",
}

// ParseLine converts one yt-dlp output line into an Event. ok=false for
// lines the UI does not care about.
func ParseLine(line string) (Event, bool) {
	s := strings.TrimSpace(strings.TrimRight(line, "\r"))
	if s == "" {
		return Event{}, false
	}

	switch {
	case strings.HasPrefix(s, progressPrefix):
		return parseProgressLine(s[len(progressPrefix):]), true

	case strings.HasPrefix(s, "[download] Downloading item "):
		if m := itemRe.FindStringSubmatch(s); m != nil {
			item, _ := strconv.Atoi(m[1])
			items, _ := strconv.Atoi(m[2])
			return Event{Kind: EventItem, Item: item, Items: items}, true
		}

	case strings.HasPrefix(s, "[download] Destination: "):
		if m := destRe.FindStringSubmatch(s); m != nil {
			return Event{Kind: EventDest, Text: filepath.Base(strings.TrimSpace(m[1]))}, true
		}

	case strings.HasSuffix(s, " has already been downloaded"):
		if m := alreadyRe.FindStringSubmatch(s); m != nil {
			return Event{Kind: EventAlready, Text: filepath.Base(strings.TrimSpace(m[1]))}, true
		}

	case phaseRe.MatchString(s):
		if m := phaseRe.FindStringSubmatch(s); m != nil {
			return Event{Kind: EventPhase, Text: phaseLabels[m[1]]}, true
		}

	case strings.HasPrefix(s, "[download]") && strings.Contains(s, "% of"):
		return parseClassicLine(s), true
	}

	// --print after_move:filepath output: a bare absolute path on its own.
	if filepath.IsAbs(s) && !strings.HasPrefix(s, "[") {
		return Event{Kind: EventFinalPath, Text: s}, true
	}

	return Event{}, false
}

func parseProgressLine(payload string) Event {
	fields := strings.Split(payload, "|")
	p := Progress{}
	if len(fields) > 0 {
		p.Percent, _ = strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(fields[0]), "%")), 64)
	}
	if len(fields) > 1 {
		p.Speed = cleanField(fields[1])
	}
	if len(fields) > 2 {
		p.ETA = cleanField(fields[2])
	}
	if len(fields) > 3 {
		p.Total = cleanField(fields[3])
	}
	return Event{Kind: EventProgress, Progress: p}
}

func parseClassicLine(s string) Event {
	m := classicRe.FindStringSubmatch(s)
	if m == nil {
		return Event{}
	}
	pct, _ := strconv.ParseFloat(m[1], 64)
	speed := m[4]
	if speed == "Unknown" {
		speed = ""
	}
	return Event{
		Kind: EventProgress,
		Progress: Progress{
			Percent: pct,
			Total:   m[2] + m[3],
			Speed:   speed,
			ETA:     m[5],
		},
	}
}

// cleanField normalizes a template field: "  N/A", "Unknown B/s" etc.
func cleanField(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "", "N/A", "Unknown", "Unknown B/s":
		return ""
	}
	return s
}
