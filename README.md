# ▶ ytdl-tui

A keyboard-driven terminal UI for **yt-dlp**. Paste a link, pick what you
want with fuzzy search, watch the progress. Built with Bubble Tea + Lip
Gloss + Bubbles, and [fzf](https://github.com/junegunn/fzf)'s actual
matching engine for every filter.

![UI](https://img.shields.io/badge/theme-YouTube%20red-FF0033)

## Features

- **Paste any link** — single videos, `youtu.be` short links, playlists,
  anything yt-dlp understands.
- **Guided download flow** — media type → codec/container → resolution.
  Every step is a fuzzy-filtered picker.
- **Sizes up front** — resolution rows show the estimated download size
  (video + merged audio track) whenever yt-dlp can calculate it.
- **Audio extraction** — pick a specific audio track (Opus / AAC / Vorbis,
  by bitrate) or let yt-dlp choose, converted to your configured format.
- **Expert mode** — browse the raw yt-dlp format list (`id · ext · res ·
  codec · size`) and download exactly one format.
- **Playlists** — download everything, audio-only everything, or fuzzy
  search and select specific entries (tab to mark, ctrl+a for all/none).
- **Live progress** — red gradient bar with percent, speed, ETA, item
  x/y for playlists, and the current phase (merging, embedding…).
- **Config file** — download folder, filename template, merge container,
  audio format, playlist quality cap, thumbnail/metadata embedding, and
  concurrent fragments — editable in the TUI's settings screen.

## Install

Requirements: [Go](https://go.dev) 1.21+, `yt-dlp` on your `$PATH`
(merging/embedding also wants `ffmpeg`).

```sh
git clone https://github.com/thalha-dev/ytdl-tui
cd ytdl-tui
make install        # builds to ./bin and copies to /usr/local/bin
```

or plain: `go build -o ytdl-tui ./cmd/ytdl-tui`

## Usage

```sh
ytdl-tui                     # config at ~/.config/ytdl-tui/config.yaml
ytdl-tui --config ./my.yaml  # custom config path
ytdl-tui --version
```

Paste a link, press `enter`, choose, done. Files land in your configured
download folder (default `~/Downloads/YouTube`).

## Keys

| Screen | Keys |
| --- | --- |
| Everywhere | `ctrl+c` quit · `?` help |
| URL | `enter` fetch · `ctrl+s` settings · `esc` clear/quit |
| Pickers | type to **fuzzy filter** (fzf ranking) · `↑↓` move · `enter` select · `esc` back |
| Multi-select | `tab` mark · `ctrl+a` all/none · `enter` download selection |
| Download | `c` cancel (twice) · `o` open folder · `enter` new download |
| Settings | `↑↓` navigate · `enter` edit/cycle · `s` save · `esc` back |

## Configuration

`~/.config/ytdl-tui/config.yaml` (created on first run, editable in-app
via `ctrl+s`):

```yaml
download_dir: ~/Downloads/YouTube
filename_template: '%(title)s [%(id)s].%(ext)s'
merge_format: mp4          # mp4 | mkv | webm
audio_format: m4a          # best | m4a | mp3 | opus | flac | wav | vorbis
playlist_quality: "1080"   # best | 2160 | 1440 | 1080 | 720 | 480 | 360
embed_thumbnail: true
embed_metadata: true
concurrent_fragments: 4    # 1..8
cookies_from_browser: ""   # e.g. firefox, chrome, or firefox:/path/to/profile
cookies_file: ""           # exported cookies.txt — used when no browser is set
```

## YouTube bot-check ("Sign in to confirm you're not a bot")

YouTube occasionally blocks unauthenticated yt-dlp requests based on IP
reputation — bursts of downloads make it more likely, and it usually
clears on its own. Permanent fix: authenticate yt-dlp with your browser's
cookies.

- In the TUI: `ctrl+s` → **Cookies (browser)** → cycle to your browser
  (`enter`) → `s` to save. Installed browsers are auto-detected, including
  Firefox forks like Zen (passed as `firefox:<profile path>`).
- Cookies help most when that browser is **signed in to YouTube**.
- Or export a `cookies.txt` and set **Cookies file**.
- When the bot-check hits, the error message points you at these settings.
- The check is IP-reputation based: it flaps, and retrying a few minutes
  later often just works.
- Safari requires Full Disk Access for your terminal to read cookies.

## Project layout

```
cmd/ytdl-tui/          entrypoint (flags, config bootstrap)
internal/
  config/              YAML config: load/save/normalize
  fzf/                 wrapper over junegunn/fzf's matching engine
  ytdlp/               yt-dlp wrapper: probe, formats, download, progress
  ui/                  root model, screens, key handling
  ui/components/       reusable fzf-backed picker
  ui/styles/           lipgloss theme (YouTube-red palette)
scripts/e2e_drive.py   PTY end-to-end driver (drives the real binary)
```

## Development

```sh
make build    # ./bin/ytdl-tui
make test     # go vet + unit tests (network tests skipped with -short)
make e2e      # drive the real TUI through every flow (needs network)
make run      # build & launch
```

## License

MIT
