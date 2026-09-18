package ytdlp

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CookieArgs builds yt-dlp's authentication flags from the config:
// `--cookies-from-browser <value>` (e.g. "firefox:/path/to/profile") takes
// precedence, then `--cookies <file>`, otherwise nothing.
func CookieArgs(fromBrowser, file string) []string {
	if fromBrowser != "" {
		return []string{"--cookies-from-browser", fromBrowser}
	}
	if file != "" {
		return []string{"--cookies", file}
	}
	return nil
}

var botCheckMarkers = []string{
	"sign in to confirm",
	"confirm you're not a bot",
	"--cookies-from-browser",
}

// BotCheckHint detects YouTube's "Sign in to confirm you're not a bot"
// error and returns a user-facing hint pointing at the settings screen.
func BotCheckHint(stderr string) (string, bool) {
	low := strings.ToLower(stderr)
	for _, m := range botCheckMarkers {
		if strings.Contains(low, m) {
			hint := "YouTube flagged this IP for a bot check — retrying in a few minutes often just works."
			if detected := DetectBrowsers(); len(detected) > 0 {
				hint += fmt.Sprintf(" To use your account: sign in to YouTube in your browser, then ctrl+s → Cookies (browser) → %s → s.",
					detected[0])
			} else {
				hint += " To use your account: ctrl+s → Cookies (browser) or Cookies file."
			}
			return hint, true
		}
	}
	return "", false
}

// DetectBrowsers returns yt-dlp-ready `--cookies-from-browser` values for
// every browser with a usable cookie store on this machine, best first.
// Firefox forks (Zen) yield "firefox:<default-profile-path>".
func DetectBrowsers() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var out []string
	seen := map[string]bool{}
	add := func(v string) {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}

	// Firefox forks keep a Firefox-style profiles.ini, but under their own
	// directory — yt-dlp only knows the plain "firefox" path, so we hand it
	// the resolved profile: firefox:<path>.
	for _, base := range firefoxBaseDirs(home) {
		if prof, ok := defaultFirefoxProfile(base); ok {
			add("firefox:" + prof)
		}
	}

	for _, c := range chromiumCandidates(home) {
		if dirExists(c.path) {
			add(c.name)
		}
	}

	if runtime.GOOS == "darwin" {
		if _, err := os.Stat(filepath.Join(home, "Library", "Cookies", "Cookies.binarycookies")); err == nil {
			add("safari") // needs Full Disk Access for the terminal
		}
	}

	return out
}

func firefoxBaseDirs(home string) []string {
	switch runtime.GOOS {
	case "darwin":
		appSupport := filepath.Join(home, "Library", "Application Support")
		return []string{
			filepath.Join(appSupport, "zen"),     // Zen Browser
			filepath.Join(appSupport, "Firefox"), // plain Firefox
		}
	case "windows":
		if cfg, err := os.UserConfigDir(); err == nil { // %AppData%
			return []string{
				filepath.Join(cfg, "zen"), // Zen Browser
				filepath.Join(cfg, "Mozilla", "Firefox"),
			}
		}
		return nil
	default: // linux and friends
		return []string{
			filepath.Join(home, ".zen"),
			filepath.Join(home, ".mozilla", "firefox"),
		}
	}
}

func chromiumCandidates(home string) []struct {
	name string
	path string
} {
	appSupport := filepath.Join(home, "Library", "Application Support")
	xdg := filepath.Join(home, ".config")

	var dirs []struct{ name, path string }
	for _, d := range []struct {
		name                           string
		darwin, linuxPath, windowsPath string
	}{
		{"chrome", filepath.Join(appSupport, "Google", "Chrome"), filepath.Join(xdg, "google-chrome"), filepath.Join(localAppData(), "Google", "Chrome", "User Data")},
		{"brave", filepath.Join(appSupport, "BraveSoftware", "Brave-Browser"), filepath.Join(xdg, "BraveSoftware", "Brave-Browser"), filepath.Join(localAppData(), "BraveSoftware", "Brave-Browser", "User Data")},
		{"edge", filepath.Join(appSupport, "Microsoft Edge"), filepath.Join(xdg, "microsoft-edge"), filepath.Join(localAppData(), "Microsoft", "Edge", "User Data")},
		{"vivaldi", filepath.Join(appSupport, "Vivaldi"), filepath.Join(xdg, "vivaldi"), filepath.Join(localAppData(), "Vivaldi", "User Data")},
		{"chromium", filepath.Join(appSupport, "Chromium"), filepath.Join(xdg, "chromium"), filepath.Join(localAppData(), "Chromium", "User Data")},
		{"opera", filepath.Join(appSupport, "com.operasoftware.Opera"), filepath.Join(xdg, "opera"), filepath.Join(localAppData(), "Opera Software", "Opera Stable")},
	} {
		p := d.linuxPath
		switch runtime.GOOS {
		case "darwin":
			p = d.darwin
		case "windows":
			p = d.windowsPath
		}
		if p != "" {
			dirs = append(dirs, struct{ name, path string }{d.name, p})
		}
	}
	return dirs
}

// localAppData returns %LOCALAPPDATA% (Windows only; empty elsewhere).
func localAppData() string {
	return os.Getenv("LOCALAPPDATA")
}

// defaultFirefoxProfile resolves the profile a Firefox fork actually uses:
// an [Install*] section's Default= wins, then Default=1 profiles, then the
// first listed profile. Only returns a profile that has a cookies.sqlite.
func defaultFirefoxProfile(base string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(base, "profiles.ini"))
	if err != nil {
		return "", false
	}

	var (
		installDefault string
		sectionDefault string
		firstPath      string
		inInstall      bool
	)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "["):
			inInstall = strings.HasPrefix(line, "[Install")
		case strings.HasPrefix(line, "Default=") && inInstall && installDefault == "":
			installDefault = strings.TrimPrefix(line, "Default=")
		case strings.HasPrefix(line, "Default=") && !inInstall:
			sectionDefault = strings.TrimPrefix(line, "Default=")
		case strings.HasPrefix(line, "Path="):
			p := strings.TrimPrefix(line, "Path=")
			if firstPath == "" {
				firstPath = p
			}
			if sectionDefault == "1" {
				sectionDefault = p // remember the Default=1 profile path
			}
		}
	}

	pick := installDefault
	if pick == "" {
		pick = sectionDefault
	}
	if pick == "" {
		pick = firstPath
	}
	if pick == "" {
		return "", false
	}
	if !filepath.IsAbs(pick) {
		pick = filepath.Join(base, pick)
	}
	if _, err := os.Stat(filepath.Join(pick, "cookies.sqlite")); err != nil {
		return "", false // profile never used — no cookies to export
	}
	return pick, true
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
