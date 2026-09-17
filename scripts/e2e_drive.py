#!/usr/bin/env python3
"""E2E driver for ytdl-tui.

Spawns the built TUI binary in a PTY and drives it purely with keystrokes,
asserting on what renders. Run `go build -o bin/ytdl-tui ./cmd/ytdl-tui`
first, then: python3 scripts/e2e_drive.py
"""
import fcntl
import os
import pty
import re
import select
import shutil
import signal
import struct
import subprocess
import sys
import termios
import time

BASE = "/tmp/ytdl-tui-e2e"
BIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "bin", "ytdl-tui")
CFG = os.path.join(BASE, "config.yaml")
DL = os.path.join(BASE, "downloads")
VIDEO_URL = "https://www.youtube.com/watch?v=278IRQ6HSi4"
PLAYLIST_URL = "https://www.youtube.com/playlist?list=PLbpi6ZahtOH6Blw3RGYpWkSByi_T7Rygb"

ANSI = re.compile(r"\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][0-9A-B]|\x1b.")

UP, DOWN, ENTER, ESC, TAB, CTRL_S, CTRL_C = "\x1b[A", "\x1b[B", "\r", "\x1b", "\t", "\x13", "\x03"
BS = "\x7f"

DEFAULT_CONFIG = """download_dir: {dl}
filename_template: '%(title)s [%(id)s].%(ext)s'
merge_format: mp4
audio_format: m4a
playlist_quality: "1080"
embed_thumbnail: true
embed_metadata: true
concurrent_fragments: 4
""".format(dl=DL)


class TUI:
    def __init__(self):
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            env = dict(os.environ, TERM="xterm-256color")
            os.execvpe(BIN, [BIN, "--config", CFG], env)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", 36, 110, 0, 0))
        self.raw = b""
        time.sleep(0.4)

    def text(self):
        return ANSI.sub("", self.raw.decode("utf-8", "replace"))

    def drain(self):
        """Read any pending output into the buffer."""
        while True:
            r, _, _ = select.select([self.fd], [], [], 0.05)
            if not r:
                return
            try:
                chunk = os.read(self.fd, 1 << 16)
            except OSError:
                return
            if not chunk:
                return
            self.raw += chunk

    def frame(self):
        """The currently visible screen. Bubbletea's renderer repaints the
        full alt-screen on resize, so bounce the PTY height, drop history,
        and capture only the fresh repaint."""
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", 35, 110, 0, 0))
        time.sleep(0.15)
        self.drain()
        self.raw = b""  # discard history: only fresh repaints from here
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", 36, 110, 0, 0))
        time.sleep(0.25)
        self.drain()
        return self.text()

    def send(self, keys, pause=0.2):
        os.write(self.fd, keys.encode())
        time.sleep(pause)

    def expect(self, pattern, timeout=30):
        rx = re.compile(pattern, re.S)
        deadline = time.time() + timeout
        while time.time() < deadline:
            if rx.search(self.text()):
                return True
            r, _, _ = select.select([self.fd], [], [], 0.1)
            if r:
                try:
                    chunk = os.read(self.fd, 1 << 16)
                except OSError:
                    break
                if not chunk:
                    break
                self.raw += chunk
        return False

    def snapshot(self, name):
        path = os.path.join(BASE, name + ".transcript.txt")
        with open(path, "w") as f:
            f.write(self.text())
        return path

    def quit(self):
        try:
            os.write(self.fd, CTRL_C.encode())
            time.sleep(0.5)
            os.kill(self.pid, signal.SIGKILL)
        except (ProcessLookupError, OSError):
            pass
        try:
            os.close(self.fd)
        except OSError:
            pass


class Check(Exception):
    pass


def require(cond, msg, tui=None):
    if not cond:
        if tui:
            tui.snapshot("FAIL-last-state")
        raise Check(msg)


def scenario_single():
    """URL -> Video -> fzf filter test -> H.264 -> 360p -> download."""
    t = TUI()
    try:
        require(t.expect(r"What should we download", 15), "URL screen did not render", t)
        t.send(VIDEO_URL + ENTER)
        require(t.expect(r"What do you want to download", 60), "video menu never appeared", t)
        txt = t.text()
        require("drone" in txt.lower(), "video title missing from menu", t)
        t.snapshot("single-1-menu")

        # fzf filtering on the codec screen: typing vp9 must hide H.264
        t.send(ENTER)  # Video
        require(t.expect(r"Pick a codec", 10), "codec screen never appeared", t)
        t.send("vp9", pause=0.5)
        fr = t.frame()
        require("VP9" in fr and "H.264" not in fr, "fzf filter did not narrow to VP9", t)
        t.send(BS * 3, pause=0.4)  # clear filter
        t.send(DOWN + ENTER)       # MP4 · H.264
        require(t.expect(r"Pick a resolution", 10), "resolution screen never appeared", t)
        fr = t.frame()
        require("1080p" in fr, "1080p row missing", t)
        require("≈" in fr, "sizes missing on resolution rows", t)
        t.snapshot("single-2-resolution")

        # rows: Best-of, 1080p, 720p, 480p, 360p, ...
        t.send(DOWN * 4 + ENTER)  # 360p
        require(t.expect(r"▍ Download", 10), "download screen never appeared", t)
        require(t.expect(r"✓ saved to", 240), "download never finished", t)
        fr = t.frame()
        t.snapshot("single-3-done")
        require("360p" in fr, "chosen resolution not reflected", t)
        files = os.listdir(DL)
        require(len(files) == 1, "expected exactly 1 downloaded file, got %r" % files, t)
        require("[278IRQ6HSi4]" in files[0], "filename template not applied: %s" % files[0], t)
        print("PASS single video: downloaded", files[0])
    finally:
        t.quit()


def scenario_audio():
    """URL -> Audio only -> Best available -> saved .m4a."""
    t = TUI()
    try:
        require(t.expect(r"What should we download", 15), "URL screen", t)
        t.send(VIDEO_URL + ENTER)
        require(t.expect(r"What do you want to download", 60), "video menu", t)
        t.send(DOWN + ENTER)  # Audio only
        require(t.expect(r"Pick an audio format", 10), "audio screen", t)
        t.snapshot("audio-1-formats")
        t.send(ENTER)  # Best available
        require(t.expect(r"✓ saved to", 240), "audio download never finished", t)
        t.snapshot("audio-2-done")
        files = os.listdir(DL)
        require(any(f.endswith(".m4a") for f in files), "no .m4a produced: %r" % files, t)
        print("PASS audio: downloaded", [f for f in files if f.endswith(".m4a")][0])
    finally:
        t.quit()


def scenario_settings():
    """ctrl+s -> cycle values on every toggle kind -> save -> file updated."""
    t = TUI()
    try:
        require(t.expect(r"What should we download", 15), "URL screen", t)
        t.send(CTRL_S)
        require(t.expect(r"▍ Settings", 10), "settings screen", t)
        t.snapshot("settings-1")
        # fields: 0 dir, 1 template, 2 merge, 3 audio, 4 quality,
        #         5 thumbnail, 6 metadata, 7 fragments
        t.send(DOWN * 2 + ENTER)      # merge: mp4 -> mkv
        t.send(DOWN * 3 + ENTER)      # thumbnail: on -> off
        t.send(DOWN + ENTER)          # metadata: on -> off
        t.send(DOWN + ENTER)          # fragments: 4 -> 8 (cycle 1,2,4,8)
        t.send(UP * 2 + ENTER)        # thumbnail: off -> on  (regression: bool cycle)
        t.send("s", pause=0.5)
        require(t.expect(r"Settings saved", 10), "save toast", t)
        cfg = open(CFG).read()
        require("merge_format: mkv" in cfg, "config not updated: %r" % cfg, t)
        require("embed_thumbnail: true" in cfg, "thumbnail should be on: %r" % cfg, t)
        require("embed_metadata: false" in cfg, "metadata should be off: %r" % cfg, t)
        require("concurrent_fragments: 8" in cfg, "fragments should be 8: %r" % cfg, t)
        require(t.expect(r"What should we download", 5), "did not return to URL screen", t)
        print("PASS settings: toggles + save persisted")
    finally:
        t.quit()


def scenario_playlist():
    """Playlist URL -> Select videos -> tab one entry -> download."""
    cfg = open(CFG).read()
    cfg = re.sub(r"playlist_quality:.*", 'playlist_quality: "360"', cfg)
    open(CFG, "w").write(cfg)

    before = set(os.listdir(DL))
    t = TUI()
    try:
        require(t.expect(r"What should we download", 15), "URL screen", t)
        t.send(PLAYLIST_URL + ENTER)
        require(t.expect(r"This is a playlist", 120), "playlist menu never appeared", t)
        t.snapshot("playlist-1-menu")
        t.send(DOWN * 2 + ENTER)  # Select videos
        require(t.expect(r"mark the ones to download", 30), "playlist picker", t)
        t.send(TAB + ENTER)  # select first entry, confirm
        require(t.expect(r"✓ (saved to|\d+ files saved to)", 300), "playlist download never finished", t)
        t.snapshot("playlist-2-done")
        new = set(os.listdir(DL)) - before
        require(len(new) == 1, "expected 1 new file, got %r" % new, t)
        print("PASS playlist: downloaded", new.pop())
    finally:
        t.quit()


SCENARIOS = {
    "single": scenario_single,
    "audio": scenario_audio,
    "settings": scenario_settings,
    "playlist": scenario_playlist,
}


def main():
    if not os.path.exists(BIN):
        print("binary missing — run: go build -o bin/ytdl-tui ./cmd/ytdl-tui", file=sys.stderr)
        return 1
    if os.path.exists(BASE):
        shutil.rmtree(BASE)
    os.makedirs(DL)
    with open(CFG, "w") as f:
        f.write(DEFAULT_CONFIG)

    names = sys.argv[1:] or list(SCENARIOS)
    failures = 0
    for name in names:
        print("== scenario:", name, flush=True)
        try:
            SCENARIOS[name]()
        except Check as e:
            failures += 1
            print("FAIL %s: %s" % (name, e), flush=True)
        except Exception as e:  # noqa: BLE001
            failures += 1
            print("ERROR %s: %r" % (name, e), flush=True)
    print("== %d/%d scenarios passed" % (len(names) - failures, len(names)))
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
