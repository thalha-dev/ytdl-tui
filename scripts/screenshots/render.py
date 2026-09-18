#!/usr/bin/env python3
"""Render deterministic README screenshots by driving the real TUI in a PTY.

Uses a fake `yt-dlp` (same directory) that serves canned metadata and
simulates progress, so the shots are identical on every run. pyte emulates
the terminal screen; Pillow paints it.

Usage: python3 scripts/screenshots/render.py   (from the repo root)
"""
import os
import pty
import re
import select
import struct
import sys
import termios
import time
import fcntl

import pyte
from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
BIN = os.path.join(ROOT, "bin", "ytdl-tui")
OUT = os.path.join(ROOT, "docs", "screenshots")
CONFIG = os.path.join(HERE, "config.yaml")
COLS, ROWS = 104, 26

UP, DOWN, ENTER, ESC, TAB = "\x1b[A", "\x1b[B", "\r", "\x1b", "\t"

# xterm-256 palette: 16 basics, 6x6x6 cube, grayscale ramp.
LEVELS = [0, 95, 135, 175, 215, 255]
PALETTE = [
    (0, 0, 0), (205, 0, 0), (0, 205, 0), (205, 205, 0), (0, 0, 238), (205, 0, 205),
    (0, 205, 205), (229, 229, 229), (127, 127, 127), (255, 0, 0), (0, 255, 0),
    (255, 255, 0), (92, 92, 255), (255, 0, 255), (0, 255, 255), (255, 255, 255),
] + [
    (LEVELS[r], LEVELS[g], LEVELS[b]) for r in range(6) for g in range(6) for b in range(6)
] + [(8 + 10 * i, 8 + 10 * i, 8 + 10 * i) for i in range(24)]

BG = (21, 21, 28)        # terminal background
FG = (234, 234, 238)     # default foreground


def color_of(spec, default):
    """pyte stores colors as 'default', '3' (palette idx) or 'ff0033' (truecolor)."""
    if spec in (None, "default", ""):
        return default
    s = spec[3:] if spec.startswith(("fg:", "bg:")) else spec
    if re.fullmatch(r"[0-9a-fA-F]{6}", s):
        return tuple(int(s[i:i + 2], 16) for i in (0, 2, 4))
    try:
        return PALETTE[int(s) % 256]
    except ValueError:
        return default


class Session:
    def __init__(self):
        self.screen = pyte.Screen(COLS, ROWS)
        self.stream = pyte.ByteStream(self.screen)
        self.pid, self.fd = pty.fork()
        if self.pid == 0:
            stub = HERE  # dir containing the fake yt-dlp
            env = dict(os.environ, TERM="xterm-256color", PATH=stub + ":" + os.environ["PATH"])
            os.execvpe(BIN, [BIN, "--config", CONFIG], env)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))
        time.sleep(0.5)
        self.drain()

    def drain(self, quiet=0.4, max_wait=8.0):
        """Read output until the stream has been quiet for a moment.
        Responds to terminal capability queries the way a real terminal
        would, since termenv blocks on them before rendering."""
        deadline = time.time() + max_wait
        last = time.time()
        while time.time() < deadline and time.time() - last < quiet:
            r, _, _ = select.select([self.fd], [], [], 0.05)
            if r:
                try:
                    chunk = os.read(self.fd, 1 << 16)
                except OSError:
                    return
                last = time.time()
                if b"\x1b]11;?" in chunk:  # background color query
                    os.write(self.fd, b"\x1b]11;rgb:1515/1515/1c1c\x1b\\")
                if b"\x1b[6n" in chunk:  # cursor position report
                    os.write(self.fd, b"\x1b[1;1R")
                self.stream.feed(chunk)

    def send(self, keys, pause=0.35):
        os.write(self.fd, keys.encode())
        time.sleep(pause)

    def render(self, path):
        font = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 17)
        cw = int(font.getlength("0"))
        ch = int(font.size * 1.45)
        img = Image.new("RGB", (COLS * cw + 80, ROWS * ch + 80), BG)
        d = ImageDraw.Draw(img)
        for y in range(ROWS):
            for x in range(COLS):
                ch_ = self.screen.buffer[y][x]
                if ch_.data == " " and ch_.bg in (None, "default"):
                    continue
                bg = color_of(ch_.bg, BG)
                if bg != BG:
                    d.rectangle([40 + x * cw, 40 + y * ch, 40 + (x + 1) * cw - 1, 40 + (y + 1) * ch - 1], fill=bg)
                fg = color_of(ch_.fg, FG)
                d.text((40 + x * cw, 38 + y * ch), ch_.data, font=font, fill=fg)
        img.save(path)
        print("wrote", path, os.path.getsize(path), "bytes")


def main():
    os.makedirs(OUT, exist_ok=True)
    s = Session()

    def shot(name):
        s.render(os.path.join(OUT, name))

    # 1. URL screen with a pasted link
    s.send("https://www.youtube.com/watch?v=278IRQ6HSi4", pause=0.8)
    shot("01-paste.png")

    # 2. menu after the probe
    s.send(ENTER, pause=0.4)
    s.drain(quiet=0.8)
    shot("02-menu.png")

    # 3. codec picker narrowed by the fzf filter
    s.send(ENTER, pause=0.4)
    s.drain(quiet=0.5)
    s.send("vp9", pause=0.6)
    shot("03-codec-filter.png")

    # 4. resolution rows with sizes
    s.send("\x7f" * 3, pause=0.3)
    s.send(DOWN + ENTER, pause=0.4)
    s.drain(quiet=0.6)
    shot("04-resolution.png")

    # 5. save-to picker (two folders configured)
    s.send(DOWN * 4 + ENTER, pause=0.4)
    s.drain(quiet=0.6)
    shot("05-save-to.png")

    # 6. mid-download progress
    s.send(ENTER, pause=0.4)
    time.sleep(4.2)
    s.drain(quiet=0.2, max_wait=2.0)
    shot("06-download.png")

    # 7. finished
    s.drain(quiet=1.2, max_wait=20.0)
    shot("07-done.png")

    # 8. settings
    s.send(ESC, pause=0.4)
    s.send("\x13", pause=0.4)  # ctrl+s
    s.drain(quiet=0.6)
    shot("08-settings.png")

    # 9. folder manager
    s.send(ENTER, pause=0.4)
    s.drain(quiet=0.6)
    shot("09-folders.png")

    try:
        os.write(s.fd, b"\x1b")  # esc back to settings
        time.sleep(0.3)
        os.kill(s.pid, 9)
    except OSError:
        pass


if __name__ == "__main__":
    main()
