// Package ytdlp wraps the yt-dlp binary: metadata probing, format
// inspection and option building, and download execution with parsed
// progress events.
package ytdlp

import "strings"

// Format is a single yt-dlp format entry (the subset of fields the TUI uses).
type Format struct {
	FormatID       string  `json:"format_id"`
	FormatNote     string  `json:"format_note"`
	Ext            string  `json:"ext"`
	Resolution     string  `json:"resolution"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	FPS            float64 `json:"fps"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	ABR            float64 `json:"abr"`
	TBR            float64 `json:"tbr"`
	Filesize       int64   `json:"filesize"`
	FilesizeApprox int64   `json:"filesize_approx"`
}

func (f Format) IsVideo() bool     { return f.VCodec != "" && f.VCodec != "none" }
func (f Format) IsAudio() bool     { return f.ACodec != "" && f.ACodec != "none" }
func (f Format) IsVideoOnly() bool { return f.IsVideo() && !f.IsAudio() }
func (f Format) IsAudioOnly() bool { return f.IsAudio() && !f.IsVideo() }

// IsStoryboard reports whether the format is a storyboard sprite sheet
// (mhtml previews YouTube reports as formats), which is never downloadable.
func (f Format) IsStoryboard() bool {
	return f.Ext == "mhtml" || strings.HasPrefix(f.FormatID, "sb")
}

// IsDRCVariant reports duplicate DRC variants YouTube lists alongside the
// identical base format ("139-drc" for "139").
func (f Format) IsDRCVariant() bool { return strings.HasSuffix(f.FormatID, "-drc") }

// BaseID strips the -drc suffix used for duplicate DRC variants.
func (f Format) BaseID() string { return strings.TrimSuffix(f.FormatID, "-drc") }

// Size returns the exact filesize if known, otherwise the reported estimate.
func (f Format) Size() int64 {
	if f.Filesize > 0 {
		return f.Filesize
	}
	return f.FilesizeApprox
}

// EstimatedSize returns the filesize, or an estimate derived from the total
// bitrate and the clip duration (in seconds). Returns 0 when unknown.
func (f Format) EstimatedSize(duration float64) int64 {
	if s := f.Size(); s > 0 {
		return s
	}
	if f.TBR > 0 && duration > 0 {
		return int64(f.TBR * 1000.0 / 8.0 * duration)
	}
	return 0
}

// VideoCodecName maps the raw codec string to a friendly name.
func (f Format) VideoCodecName() string {
	switch {
	case strings.HasPrefix(f.VCodec, "avc"):
		return "H.264"
	case strings.HasPrefix(f.VCodec, "vp0"), strings.HasPrefix(f.VCodec, "vp9"):
		return "VP9"
	case strings.HasPrefix(f.VCodec, "av01"):
		return "AV1"
	case strings.HasPrefix(f.VCodec, "hev"), strings.HasPrefix(f.VCodec, "h265"):
		return "HEVC"
	default:
		return f.VCodec
	}
}

// AudioCodecName maps the raw audio codec string to a friendly name.
func (f Format) AudioCodecName() string {
	switch {
	case strings.HasPrefix(f.ACodec, "opus"):
		return "Opus"
	case strings.HasPrefix(f.ACodec, "mp4a"):
		return "AAC"
	case strings.HasPrefix(f.ACodec, "vorbis"):
		return "Vorbis"
	case strings.HasPrefix(f.ACodec, "mp3"):
		return "MP3"
	case strings.HasPrefix(f.ACodec, "flac"):
		return "FLAC"
	default:
		return f.ACodec
	}
}

// Progress is a snapshot of a running download.
type Progress struct {
	Percent float64 // 0..100
	Speed   string  // human-readable, e.g. "3.51MiB/s"
	ETA     string  // e.g. "00:12"
	Total   string  // human-readable total size, e.g. "12.25MiB"
}

// Info is the yt-dlp JSON metadata document (subset). For playlists the
// Entries slice is populated (flat entries: id/title/duration, no formats).
type Info struct {
	Type          string   `json:"_type"`
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Channel       string   `json:"channel"`
	Uploader      string   `json:"uploader"`
	Duration      float64  `json:"duration"`
	ViewCount     int64    `json:"view_count"`
	UploadDate    string   `json:"upload_date"`
	WebpageURL    string   `json:"webpage_url"`
	ExtractorKey  string   `json:"extractor_key"`
	Formats       []Format `json:"formats"`
	Entries       []*Info  `json:"entries"`
	PlaylistCount int      `json:"playlist_count"`
}

// IsPlaylist reports whether the probed URL is a playlist.
func (i *Info) IsPlaylist() bool { return i != nil && i.Type == "playlist" }

// DisplayChannel returns the channel name, falling back to the uploader.
func (i *Info) DisplayChannel() string {
	if i.Channel != "" {
		return i.Channel
	}
	return i.Uploader
}

// DownloadableFormats returns the formats that make sense to offer the
// user: actual media tracks, without storyboards or DRC duplicates.
func (i *Info) DownloadableFormats() []Format {
	out := make([]Format, 0, len(i.Formats))
	for _, f := range i.Formats {
		if f.IsStoryboard() || f.IsDRCVariant() {
			continue
		}
		if !f.IsVideo() && !f.IsAudio() {
			continue
		}
		out = append(out, f)
	}
	return out
}
