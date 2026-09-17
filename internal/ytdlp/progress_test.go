package ytdlp

import (
	"testing"
)

func TestParseProgressTemplateLine(t *testing.T) {
	ev, ok := ParseLine("PROGRESS| 45.2%| 3.51MiB/s|00:12| 12.25MiB")
	if !ok || ev.Kind != EventProgress {
		t.Fatalf("got %+v ok=%v", ev, ok)
	}
	p := ev.Progress
	if p.Percent != 45.2 {
		t.Errorf("Percent = %v", p.Percent)
	}
	if p.Speed != "3.51MiB/s" || p.ETA != "00:12" || p.Total != "12.25MiB" {
		t.Errorf("Speed/ETA/Total = %q/%q/%q", p.Speed, p.ETA, p.ETA)
	}
}

func TestParseProgressUnknownFields(t *testing.T) {
	ev, ok := ParseLine("PROGRESS|  12.0%| Unknown B/s|Unknown| N/A")
	if !ok {
		t.Fatal("expected a progress event")
	}
	if ev.Progress.Speed != "" || ev.Progress.ETA != "" || ev.Progress.Total != "" {
		t.Errorf("unknown placeholders should clean to empty: %+v", ev.Progress)
	}
}

func TestParseItemLine(t *testing.T) {
	ev, ok := ParseLine("[download] Downloading item 3 of 25")
	if !ok || ev.Kind != EventItem || ev.Item != 3 || ev.Items != 25 {
		t.Fatalf("got %+v ok=%v", ev, ok)
	}
}

func TestParseDestinationLine(t *testing.T) {
	ev, ok := ParseLine("[download] Destination: /tmp/x/My Video [abc123].f137.mp4")
	if !ok || ev.Kind != EventDest || ev.Text != "My Video [abc123].f137.mp4" {
		t.Fatalf("got %+v ok=%v", ev, ok)
	}
}

func TestParsePhaseLines(t *testing.T) {
	cases := map[string]string{
		"[Merger] Merging formats into \"/tmp/x/out.mp4\"":    "Merging video + audio",
		"[ExtractAudio] Destination: /tmp/x/out.m4a":          "Extracting audio",
		"[EmbedThumbnail] ffmpeg: Adding thumbnail to \"/x\"": "Embedding thumbnail",
		"[Metadata] Adding metadata to \"/x\"":                "Embedding metadata",
	}
	for line, want := range cases {
		ev, ok := ParseLine(line)
		if !ok || ev.Kind != EventPhase || ev.Text != want {
			t.Errorf("ParseLine(%q) = %+v ok=%v, want phase %q", line, ev, ok, want)
		}
	}
}

func TestParseFinalPath(t *testing.T) {
	ev, ok := ParseLine("/Users/me/Downloads/YouTube/Title [id].mp4")
	if !ok || ev.Kind != EventFinalPath || ev.Text != "/Users/me/Downloads/YouTube/Title [id].mp4" {
		t.Fatalf("got %+v ok=%v", ev, ok)
	}
}

func TestParseClassicFallback(t *testing.T) {
	ev, ok := ParseLine("[download]  12.3% of    3.00MiB at    1.50MiB/s ETA 00:02")
	if !ok || ev.Kind != EventProgress {
		t.Fatalf("got %+v ok=%v", ev, ok)
	}
	if ev.Progress.Percent != 12.3 || ev.Progress.Total != "3.00MiB" || ev.Progress.ETA != "00:02" {
		t.Errorf("classic parse wrong: %+v", ev.Progress)
	}
}

func TestParseAlreadyDownloaded(t *testing.T) {
	ev, ok := ParseLine("[download] Clip One [abc].mp4 has already been downloaded")
	if !ok || ev.Kind != EventAlready || ev.Text != "Clip One [abc].mp4" {
		t.Fatalf("got %+v ok=%v", ev, ok)
	}
}

func TestParseIgnoresNoise(t *testing.T) {
	for _, line := range []string{
		"",
		"[youtube] Extracting URL",
		"WARNING: something or other",
		"[info] 278IRQ6HSi4: Downloading webpage",
	} {
		if _, ok := ParseLine(line); ok {
			t.Errorf("line %q should be ignored", line)
		}
	}
}
