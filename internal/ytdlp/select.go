package ytdlp

import "sort"

// VideoFamily groups downloadable formats by video codec for the
// codec-selection screen.
type VideoFamily struct {
	ID    string // "h264" | "vp9" | "av1" | "hevc" | "auto"
	Label string // "MP4 · H.264"
	Desc  string // "Most compatible — plays everywhere"
}

// ResolutionOption is one selectable row on the resolution screen.
type ResolutionOption struct {
	Format Format // the concrete best format for this row
	Height int
	FPS    int
	Size   int64 // estimated download size incl. the audio track merged in
}

// AudioOption is one selectable row on the audio screen.
type AudioOption struct {
	Format Format
	IsBest bool // synthetic "let yt-dlp pick the best audio" row
}

// familyByCodec maps our family IDs to the codec prefixes they match and
// their display metadata.
var familyByCodec = []struct {
	id     string
	prefix string // raw vcodec prefix
	label  string
	desc   string
}{
	{"h264", "avc", "MP4 · H.264", "Most compatible — plays everywhere"},
	{"vp9", "vp0", "WEBM · VP9", "Efficient and widely supported"},
	{"av1", "av01", "MP4 · AV1", "Smallest files, newest codec"},
	{"hevc", "hev", "MP4 · HEVC", "High efficiency, limited player support"},
}

// CodecFamilies derives the codec choices available in formats, ordered
// from most to least compatible. An "auto" option is always first.
func CodecFamilies(formats []Format) []VideoFamily {
	counts := map[string]int{}
	for _, f := range formats {
		if !f.IsVideo() {
			continue
		}
		for _, fam := range familyByCodec {
			if hasPrefix(f.VCodec, fam.prefix) {
				counts[fam.id]++
				break
			}
		}
	}

	out := []VideoFamily{{
		ID:    "auto",
		Label: "Auto — let yt-dlp decide",
		Desc:  "Best video + best audio, highest resolution",
	}}
	for _, fam := range familyByCodec {
		if counts[fam.id] == 0 {
			continue
		}
		out = append(out, VideoFamily{
			ID:    fam.id,
			Label: fam.label,
			Desc:  fam.desc,
		})
	}
	return out
}

// Resolutions lists the distinct (height, fps) options within a codec
// family, highest first. Size includes the best audio track that yt-dlp
// merges in. With familyID "auto" it returns a single option using the
// generic best selector.
func Resolutions(formats []Format, familyID string, duration float64) []ResolutionOption {
	if familyID == "auto" {
		return nil // handled by the caller: generic "bv*+ba/b" selector
	}

	var prefix string
	for _, fam := range familyByCodec {
		if fam.id == familyID {
			prefix = fam.prefix
		}
	}

	audio := bestAudioSize(formats)
	best := map[int]ResolutionOption{} // key: height*1000+fps

	for _, f := range formats {
		if !f.IsVideoOnly() || !hasPrefix(f.VCodec, prefix) || f.Height <= 0 {
			continue
		}
		fps := int(f.FPS)
		key := f.Height*1000 + fps
		cur, ok := best[key]
		if !ok || f.TBR > cur.Format.TBR {
			size := f.EstimatedSize(duration)
			if audio > 0 && size > 0 {
				size += audio
			}
			best[key] = ResolutionOption{Format: f, Height: f.Height, FPS: fps, Size: size}
		}
	}

	out := make([]ResolutionOption, 0, len(best))
	for _, opt := range best {
		out = append(out, opt)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Height != out[b].Height {
			return out[a].Height > out[b].Height
		}
		return out[a].FPS > out[b].FPS
	})
	return out
}

// AudioFormats lists downloadable audio-only tracks, best (highest bitrate)
// first, plus a synthetic "best" row at the front.
func AudioFormats(formats []Format) []AudioOption {
	type track struct {
		f Format
	}
	seen := map[string]bool{}
	var tracks []track
	for _, f := range formats {
		if !f.IsAudioOnly() {
			continue
		}
		id := f.BaseID()
		if seen[id] {
			continue
		}
		seen[id] = true
		tracks = append(tracks, track{f})
	}
	sort.SliceStable(tracks, func(a, b int) bool {
		if tracks[a].f.ABR != tracks[b].f.ABR {
			return tracks[a].f.ABR > tracks[b].f.ABR
		}
		return tracks[a].f.TBR > tracks[b].f.TBR
	})

	out := []AudioOption{{IsBest: true}}
	for _, t := range tracks {
		out = append(out, AudioOption{Format: t.f})
	}
	return out
}

// FormatSelector builds yt-dlp's -f expression for a chosen format:
// "<id>+bestaudio/<id>" for video-only tracks, the plain id otherwise.
func FormatSelector(f Format) string {
	if f.IsVideoOnly() {
		return f.FormatID + "+bestaudio/" + f.FormatID
	}
	return f.FormatID
}

// bestAudioSize returns the size of the largest audio-only track, used to
// estimate the merged download size.
func bestAudioSize(formats []Format) int64 {
	var best int64
	for _, f := range formats {
		if !f.IsAudioOnly() {
			continue
		}
		if s := f.Size(); s > best {
			best = s
		}
	}
	return best
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
