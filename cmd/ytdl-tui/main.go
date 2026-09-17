// Command ytdl-tui is a keyboard-driven TUI for yt-dlp: paste a link, pick
// codec & resolution (fzf-style), watch the download, done.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"

	"github.com/thalha-dev/ytdl-tui/internal/config"
	"github.com/thalha-dev/ytdl-tui/internal/ui"
)

// version is set via -ldflags at build time.
var version = "dev"

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: "+config.DefaultPath()+")")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("ytdl-tui", version)
		return
	}

	path := *cfgPath
	if path == "" {
		path = config.DefaultPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ytdl-tui: %v\n", err)
		os.Exit(1)
	}

	// Make sure the download folder exists before the first download.
	if err := os.MkdirAll(cfg.DownloadDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "ytdl-tui: cannot create download folder: %v\n", err)
		os.Exit(1)
	}

	// Optional debug logging: YTDL_TUI_DEBUG=path/to/log
	if debug := os.Getenv("YTDL_TUI_DEBUG"); debug != "" {
		if f, err := tea.LogToFile(debug, "ytdl-tui"); err == nil {
			defer f.Close()
		}
	}

	if err := ui.Run(cfg, path, version); err != nil {
		fmt.Fprintf(os.Stderr, "ytdl-tui: %v\n", err)
		os.Exit(1)
	}
}
