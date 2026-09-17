package ytdlp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) *Info {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "probe.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	info, err := parseInfo(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return info
}

func TestFixtureIsVideoWithFormats(t *testing.T) {
	info := loadFixture(t)
	if info.IsPlaylist() {
		t.Fatal("fixture should be a single video")
	}
	if len(info.DownloadableFormats()) == 0 {
		t.Fatal("fixture has no downloadable formats")
	}
	if info.Title == "" {
		t.Fatal("fixture title empty")
	}
}

func TestDownloadableFormatsFiltersNoise(t *testing.T) {
	info := loadFixture(t)
	for _, f := range info.DownloadableFormats() {
		if f.IsStoryboard() {
			t.Errorf("storyboard %s leaked through", f.FormatID)
		}
		if f.IsDRCVariant() {
			t.Errorf("drc variant %s leaked through", f.FormatID)
		}
		if !f.IsVideo() && !f.IsAudio() {
			t.Errorf("format %s is neither video nor audio", f.FormatID)
		}
	}
}

func TestCodecFamilies(t *testing.T) {
	info := loadFixture(t)
	fams := CodecFamilies(info.DownloadableFormats())

	if fams[0].ID != "auto" {
		t.Errorf("first family should be auto, got %s", fams[0].ID)
	}
	ids := map[string]bool{}
	for _, f := range fams {
		ids[f.ID] = true
	}
	for _, want := range []string{"h264", "vp9", "av1"} {
		if !ids[want] {
			t.Errorf("family %s missing from %v", want, ids)
		}
	}
}

func TestResolutionsSortedAndSized(t *testing.T) {
	info := loadFixture(t)
	formats := info.DownloadableFormats()
	opts := Resolutions(formats, "h264", info.Duration)

	if len(opts) == 0 {
		t.Fatal("no h264 resolutions")
	}
	for i := 1; i < len(opts); i++ {
		if opts[i].Height > opts[i-1].Height {
			t.Errorf("resolutions not sorted desc: %v", opts)
		}
	}
	if opts[0].Height != 1080 {
		t.Errorf("top h264 resolution = %d, want 1080", opts[0].Height)
	}
	if opts[0].Size <= 0 {
		t.Error("top resolution should have a calculable size")
	}
	// merged size must include the audio track
	videoOnly := opts[0].Format.EstimatedSize(info.Duration)
	if opts[0].Size < videoOnly {
		t.Errorf("merged size %d smaller than video-only %d", opts[0].Size, videoOnly)
	}
}

func TestResolutionsUnknownFamily(t *testing.T) {
	info := loadFixture(t)
	if opts := Resolutions(info.DownloadableFormats(), "hevc", info.Duration); len(opts) != 0 {
		t.Errorf("hevc not in fixture; want 0 options, got %d", len(opts))
	}
}

func TestAudioFormatsOrdering(t *testing.T) {
	info := loadFixture(t)
	opts := AudioFormats(info.DownloadableFormats())

	if len(opts) < 2 {
		t.Fatalf("want best + several audio rows, got %d", len(opts))
	}
	if !opts[0].IsBest {
		t.Error("first audio option should be the synthetic best")
	}
	for i := 2; i < len(opts); i++ {
		if opts[i].Format.ABR > opts[i-1].Format.ABR {
			t.Errorf("audio options not sorted by abr desc: %v", opts)
		}
	}
	// fixture has 130 kbps opus as the best real audio track
	if got := opts[1].Format.AudioCodecName(); got != "Opus" {
		t.Errorf("best audio = %s, want Opus", got)
	}
}

func TestFormatSelector(t *testing.T) {
	videoOnly := Format{FormatID: "137", VCodec: "avc1.640028", ACodec: "none"}
	if got := FormatSelector(videoOnly); got != "137+bestaudio/137" {
		t.Errorf("video-only selector = %q", got)
	}
	audioOnly := Format{FormatID: "251", VCodec: "none", ACodec: "opus"}
	if got := FormatSelector(audioOnly); got != "251" {
		t.Errorf("audio-only selector = %q", got)
	}
	combined := Format{FormatID: "18", VCodec: "avc1", ACodec: "mp4a"}
	if got := FormatSelector(combined); got != "18" {
		t.Errorf("combined selector = %q", got)
	}
}

func TestEstimatedSizeFromBitrate(t *testing.T) {
	f := Format{TBR: 1000} // 1000 kbps for 8 seconds => ~1 MB
	got := f.EstimatedSize(8)
	if got != 1_000_000 {
		t.Errorf("EstimatedSize = %d, want 1000000", got)
	}
	zero := Format{}
	if zero.EstimatedSize(10) != 0 {
		t.Error("no bitrate and no size should estimate to 0")
	}
}

func TestParseInfo(t *testing.T) {
	info, err := parseInfo([]byte(`{"_type":"playlist","title":"list","entries":[{"id":"a","title":"A","duration":10.5}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsPlaylist() || len(info.Entries) != 1 || info.Entries[0].ID != "a" {
		t.Fatalf("playlist parse wrong: %+v", info)
	}
	if info.Entries[0].Duration != 10.5 {
		t.Errorf("entry duration = %v", info.Entries[0].Duration)
	}
}

func TestParseInfoRejectsGarbage(t *testing.T) {
	if _, err := parseInfo([]byte("not json at all")); err == nil {
		t.Fatal("expected an error for non-JSON output")
	}
}

func TestVideoCodecNames(t *testing.T) {
	cases := map[string]string{
		"avc1.640028": "H.264",
		"vp09.00.40":  "VP9",
		"av01.0.08M":  "AV1",
		"hev1.1.6":    "HEVC",
	}
	for in, want := range cases {
		f := Format{VCodec: in}
		if got := f.VideoCodecName(); got != want {
			t.Errorf("VideoCodecName(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestLastErrorTail(t *testing.T) {
	long := strings.Repeat("line\n", 20) + "ERROR: could not download"
	tail := lastLines(long, 2)
	if !strings.Contains(tail, "could not download") {
		t.Errorf("tail lost the error: %q", tail)
	}
	if strings.Count(tail, "line") > 2 {
		t.Errorf("tail kept too many lines: %q", tail)
	}
}
