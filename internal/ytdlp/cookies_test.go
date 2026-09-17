package ytdlp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCookieArgs(t *testing.T) {
	if got := CookieArgs("", ""); got != nil {
		t.Errorf("CookieArgs(\"\",\"\") = %v, want nil", got)
	}
	got := CookieArgs("firefox:/tmp/zen profile", "")
	want := []string{"--cookies-from-browser", "firefox:/tmp/zen profile"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("CookieArgs browser = %v, want %v", got, want)
	}
	got = CookieArgs("", "~/cookies.txt")
	if len(got) != 2 || got[0] != "--cookies" || got[1] != "~/cookies.txt" {
		t.Errorf("CookieArgs file = %v", got)
	}
	got = CookieArgs("chrome", "ignored.txt")
	if got[1] != "chrome" {
		t.Errorf("browser should take precedence, got %v", got)
	}
}

func TestBotCheckHint(t *testing.T) {
	_, ok := BotCheckHint("ERROR: [youtube] abc: Sign in to confirm you're not a bot. Use --cookies-from-browser")
	if !ok {
		t.Fatal("bot-check error not detected")
	}
	if _, ok := BotCheckHint("ERROR: unsupported URL"); ok {
		t.Error("unrelated error must not match")
	}
	if _, ok := BotCheckHint(""); ok {
		t.Error("empty stderr must not match")
	}
}

func TestDefaultFirefoxProfile(t *testing.T) {
	base := t.TempDir()
	ini := `[General]
StartWithLastProfile=1

[Profile1]
Name=Unused
IsRelative=1
Path=Profiles/unused123
Default=1

[Profile0]
Name=Default (release)
IsRelative=1
Path=Profiles/waqf.used

[InstallABC]
Default=Profiles/waqf.used
Locked=1
`
	if err := os.WriteFile(filepath.Join(base, "profiles.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	used := filepath.Join(base, "Profiles", "waqf.used")
	if err := os.MkdirAll(used, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(used, "cookies.sqlite"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := defaultFirefoxProfile(base)
	if !ok {
		t.Fatal("profile with cookies.sqlite not detected")
	}
	if got != used {
		t.Errorf("got %q, want %q (install section wins over Default=1)", got, used)
	}
}

func TestDefaultFirefoxProfileSkipsEmptyProfiles(t *testing.T) {
	base := t.TempDir()
	ini := `[Profile0]
Name=Default
IsRelative=1
Path=Profiles/fresh
Default=1
`
	if err := os.WriteFile(filepath.Join(base, "profiles.ini"), []byte(ini), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "Profiles", "fresh"), 0o755); err != nil {
		t.Fatal(err)
	}
	// no cookies.sqlite -> not offered
	if _, ok := defaultFirefoxProfile(base); ok {
		t.Error("profile without cookies.sqlite should be skipped")
	}
}

func TestDetectBrowsersSmoke(t *testing.T) {
	got := DetectBrowsers()
	seen := map[string]bool{}
	for _, v := range got {
		if v == "" {
			t.Error("empty detection value")
		}
		if seen[v] {
			t.Errorf("duplicate detection value %q", v)
		}
		seen[v] = true
	}
	// On the dev machine Zen is installed; make sure the value is well-formed
	// if detected at all (fragile to assert presence in CI).
	for _, v := range got {
		if v == "firefox:" || strings.HasSuffix(v, ":") {
			t.Errorf("malformed firefox value %q", v)
		}
	}
}
