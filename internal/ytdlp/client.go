package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Probe runs `yt-dlp -J` on the URL. Playlists are flattened (fast; entries
// carry id/title/duration but not per-entry formats). extraArgs (e.g. cookie
// flags) are passed through to yt-dlp.
func Probe(ctx context.Context, url string, extraArgs ...string) (*Info, error) {
	args := []string{"-J", "--no-warnings", "--flat-playlist"}
	args = append(args, extraArgs...)
	args = append(args, url)
	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		if hint, ok := BotCheckHint(errb.String()); ok {
			return nil, errors.New(hint)
		}
		return nil, fmt.Errorf("yt-dlp: %s", lastLines(errb.String(), 5))
	}

	info, err := parseInfo(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parse yt-dlp metadata: %w", err)
	}
	return info, nil
}

// parseInfo decodes a yt-dlp -J JSON document.
func parseInfo(data []byte) (*Info, error) {
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Request describes a single yt-dlp download invocation.
type Request struct {
	URLs           []string // one or more target URLs
	FormatSelector string   // -f value, e.g. "137+bestaudio/137" or "bestaudio"
	OutputTemplate string   // full -o value: dir + filename template
	QualityCap     string   // optional -S res: cap (e.g. "1080"), empty = best
	AudioOnly      bool
	AudioFormat    string // when AudioOnly: best|m4a|mp3|...
	MergeFormat    string // container when merging: mp4|mkv|webm
	EmbedThumbnail bool
	EmbedMetadata  bool
	Concurrency    int      // --concurrent-fragments
	CookieArgs     []string // e.g. --cookies-from-browser firefox:…
}

// EventKind classifies one parsed yt-dlp output line.
type EventKind string

const (
	EventProgress  EventKind = "progress"  // percent/speed/eta/total update
	EventPhase     EventKind = "phase"     // merging / extracting / embedding
	EventItem      EventKind = "item"      // "Downloading item 3 of 25"
	EventDest      EventKind = "dest"      // destination file line
	EventFinalPath EventKind = "finalpath" // --print after_move:filepath line
	EventAlready   EventKind = "already"   // already downloaded
)

// Event is one piece of download progress forwarded to the TUI.
type Event struct {
	Kind     EventKind
	Text     string // detail: file name, phase label, final path...
	Progress Progress
	Item     int // 1-based, for playlists
	Items    int
}

// Download runs yt-dlp, streaming parsed events to onEvent (from the
// process's output goroutines — callers should forward them onto the UI via
// tea.Program.Send). It blocks until yt-dlp exits and returns its error.
func Download(ctx context.Context, req Request, onEvent func(Event)) error {
	args := []string{
		"--newline",
		"--no-warnings",
		"--no-quiet",
		"--no-simulate",
		"--progress-template",
		"download:PROGRESS|%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s|%(progress._total_bytes_estimate_str)s",
		"--print", "after_move:filepath",
		"-o", req.OutputTemplate,
		"--concurrent-fragments", strconv.Itoa(clampInt(req.Concurrency, 1, 8)),
	}

	if req.AudioOnly {
		args = append(args, "-f", req.FormatSelector, "-x", "--audio-format", req.AudioFormat)
	} else {
		args = append(args, "-f", req.FormatSelector)
		if strings.Contains(req.FormatSelector, "+") {
			args = append(args, "--merge-output-format", req.MergeFormat)
		}
		if req.QualityCap != "" && req.QualityCap != "best" {
			args = append(args, "-S", "res:"+req.QualityCap)
		}
	}
	if req.EmbedThumbnail {
		args = append(args, "--embed-thumbnail")
	}
	if req.EmbedMetadata {
		args = append(args, "--embed-metadata")
	}
	args = append(args, req.URLs...)

	args = append(args, req.CookieArgs...)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start yt-dlp: %w", err)
	}

	errTail := &errorBuffer{}
	var wg sync.WaitGroup
	scan := func(r io.Reader, capture *errorBuffer) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if capture != nil {
				capture.addLine(line)
			}
			if ev, ok := ParseLine(line); ok {
				onEvent(ev)
			}
		}
	}
	wg.Add(2)
	go scan(stdout, nil)
	go scan(stderr, errTail)

	waitErr := cmd.Wait()
	wg.Wait() // let scanners drain the pipes before we return

	if ctx.Err() != nil {
		return ctx.Err() // canceled by the user
	}
	if waitErr != nil {
		if hint, ok := BotCheckHint(errTail.String()); ok {
			return errors.New(hint)
		}
		if msg := strings.TrimSpace(errTail.String()); msg != "" {
			return fmt.Errorf("yt-dlp: %s", lastLines(msg, 5))
		}
		return waitErr
	}
	return nil
}

// errorBuffer keeps the tail of stderr so a failed download can show why.
type errorBuffer struct {
	lines []string
	mu    sync.Mutex
}

func (b *errorBuffer) addLine(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if line = strings.TrimSpace(line); line == "" {
		return
	}
	b.lines = append(b.lines, line)
	if len(b.lines) > 16 {
		b.lines = b.lines[len(b.lines)-16:]
	}
}

func (b *errorBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Join(b.lines, "\n")
}

func lastLines(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " · ")
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
