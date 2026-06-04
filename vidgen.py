#!/usr/bin/env python3
"""
vidgen.py — Generate a video with animated gradient background and static text overlay.

Usage:
    python vidgen.py                    # uses all defaults, outputs out.mp4
    python vidgen.py -c config.json     # uses config file (missing/null keys use defaults)

JSON config keys (all optional, null uses default):
    output          str    null             Output file; null = auto-named {res}_{fps}fps_{bitrate}bps_{duration}s.{ext}
    resolution      str    "1280x720"       WxH
    framerate       int    30               FPS
    bitrate         str    "2M"             Video bitrate (e.g. "500k", "4M")
    codec           str    "libx264"        FFmpeg video codec
    duration        float  10               Seconds
    text            str    null             Static text overlay; if null, shows video metadata
    nb_colors       int    4                Colors in gradient (2-8)
    speed           float  0.08             Animation speed (color-cycles per second)
    gradient_type   str    "linear"         linear / radial / circular / spiral / square
    linear_angle    float  0                Degrees for linear gradient (0=horizontal, 90=vertical); ignored for other types
    seed            null                    Reserved (colors are explicit; ignored)
    font_size       int    null             Font size in pixels; null = auto-scale from height (height/12)
    font_color      str    "black"          Text color (HTML name or #RRGGBB)
    colors          list   [...]            Hex color strings (without #)
"""

import argparse
import json
import math
import os
import subprocess
import sys
import tempfile
from collections import deque
from concurrent.futures import ThreadPoolExecutor

import numpy as np
from PIL import Image, ImageDraw, ImageFont

_WORKERS = max(1, int((os.cpu_count() or 1) * 0.75))
PARALLEL_WINDOW = _WORKERS * 2  # max frames rendered ahead of FFmpeg writer


GRADIENT_TYPES = ("linear", "radial", "circular", "spiral", "square")
FONT_PATH = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "fonts", "NotoSans-Regular.ttf"
)

CODEC_EXT = {
    "libx264": "mp4",
    "libx265": "mp4",
    "h264": "mp4",
    "hevc": "mp4",
    "libvpx-vp9": "webm",
    "libvpx": "webm",
    "vp9": "webm",
    "vp8": "webm",
    "libaom-av1": "mkv",
    "av1": "mkv",
    "prores": "mov",
    "prores_ks": "mov",
    "dnxhd": "mxf",
    "huffyuv": "avi",
    "rawvideo": "avi",
}

# 4-color pastel palette: powder blue, blush peach, soft mauve, seafoam mint
DEFAULT_COLORS = ["546B41", "99AD7A", "DCCCAC", "FFF8EC"]

DEFAULTS = {
    "output": None,
    "resolution": "1920x1080",
    "framerate": 24,
    "bitrate": "8M",
    "codec": "libx264",
    "duration": 30,
    "text": None,
    "nb_colors": 4,
    "speed": 0.08,
    "gradient_type": "linear",
    "linear_angle": 0,
    "seed": None,
    "font_size": None,
    "font_color": "black",
    "colors": ["546B41", "99AD7A", "DCCCAC", "FFF8EC"],
}


def load_config(path):
    with open(path) as f:
        raw = json.load(f)
    cfg = dict(DEFAULTS)
    for k, v in raw.items():
        if k in cfg:
            # Allow explicit null for fields that use null as a meaningful default
            if v is None and k in ("text", "output"):
                cfg[k] = None
            elif v is not None:
                cfg[k] = v
    return cfg


def parse_args():
    p = argparse.ArgumentParser(
        description="Generate a video with animated gradient background and text overlay.",
    )
    p.add_argument(
        "-c",
        "--config",
        metavar="FILE",
        help="Path to JSON config file. Missing or null keys use defaults.",
    )
    return p.parse_args()


def hex_to_rgb(h):
    h = h.lstrip("#").lstrip("0x").lstrip("0X")
    return tuple(int(h[i : i + 2], 16) for i in (0, 2, 4))


def compute_base(width, height, gradient_type, linear_angle):
    """Precompute static spatial phase map (H×W float64). Phase is added per frame."""
    # Normalized coords in [0, 1]
    x = np.linspace(0, 1, width, endpoint=False)
    y = np.linspace(0, 1, height, endpoint=False)
    XX, YY = np.meshgrid(x, y)

    t = np.zeros_like(XX)

    if gradient_type == "linear":
        a = math.radians(linear_angle)
        cos_a, sin_a = math.cos(a), math.sin(a)
        t = XX * cos_a + YY * sin_a
        # Normalize so full color cycle spans the frame exactly once
        span = abs(cos_a) + abs(sin_a)
        t = t / span if span > 0 else t

    elif gradient_type == "radial":
        # Distance from center → rings expand outward as phase increases
        DX, DY = XX - 0.5, YY - 0.5
        t = np.sqrt(DX**2 + DY**2) / (0.5 * math.sqrt(2))

    elif gradient_type == "circular":
        # Tighter concentric rings (3 cycles center→edge)
        DX, DY = XX - 0.5, YY - 0.5
        t = (np.sqrt(DX**2 + DY**2) / (0.5 * math.sqrt(2))) * 3

    elif gradient_type == "spiral":
        # Archimedean spiral: r + angle → spins as phase increases
        DX, DY = XX - 0.5, YY - 0.5
        r = np.sqrt(DX**2 + DY**2) / (0.5 * math.sqrt(2))
        theta = (np.arctan2(DY, DX) / (2 * math.pi)) % 1
        t = r + theta

    elif gradient_type == "square":
        # Chebyshev distance from center → concentric squares expand outward
        DX = np.abs(XX - 0.5) * 2  # [0, 1]
        DY = np.abs(YY - 0.5) * 2  # [0, 1]
        t = np.maximum(DX, DY)

    return t


def phase_to_rgb(t_mod1, colors_arr):
    """Map H×W phase array [0,1) to H×W×3 uint8 RGB."""
    n = len(colors_arr)
    scaled = t_mod1 * n
    idx = scaled.astype(np.int32) % n
    frac = (scaled - np.floor(scaled))[..., np.newaxis]  # H×W×1
    c0 = colors_arr[idx]
    c1 = colors_arr[(idx + 1) % n]
    return np.clip(c0 * (1 - frac) + c1 * frac, 0, 255).astype(np.uint8)


def render_via_stdin(proc, render_frame, total_frames):
    """Stream frames directly to FFmpeg via stdin."""
    try:
        with ThreadPoolExecutor(max_workers=_WORKERS) as pool:
            pending = deque()
            for i in range(total_frames):
                if len(pending) >= PARALLEL_WINDOW:
                    proc.stdin.write(pending.popleft().result().tobytes())
                pending.append(pool.submit(render_frame, i))
            for fut in pending:
                proc.stdin.write(fut.result().tobytes())
    except BrokenPipeError:
        pass
    finally:
        proc.stdin.close()

    return proc.wait()


def render_via_tempfiles(cfg, render_frame, total_frames, framerate):
    """Render frames to temp directory, then encode with FFmpeg."""
    print("Warning: stdin unavailable, using temp files", file=sys.stderr)

    with tempfile.TemporaryDirectory() as tmp_dir:
        print(f"Rendering {total_frames} frames to {tmp_dir}...", file=sys.stderr)

        with ThreadPoolExecutor(max_workers=_WORKERS) as pool:
            futures = [pool.submit(render_frame, i) for i in range(total_frames)]
            for i, fut in enumerate(futures):
                img = fut.result()
                img.save(os.path.join(tmp_dir, f"frame_{i:05d}.png"))

        print("Encoding video from temp files...", file=sys.stderr)
        ffmpeg_cmd_files = [
            "ffmpeg",
            "-y",
            "-framerate",
            str(framerate),
            "-i",
            os.path.join(tmp_dir, "frame_%05d.png"),
            "-vcodec",
            cfg["codec"],
            "-b:v",
            cfg["bitrate"],
            "-t",
            str(cfg["duration"]),
            cfg["output"],
        ]
        result = subprocess.run(ffmpeg_cmd_files)
        return result.returncode


def main():
    args = parse_args()
    cfg = load_config(args.config) if args.config else dict(DEFAULTS)

    if not os.path.isfile(FONT_PATH):
        print(f"Error: font not found at {FONT_PATH}", file=sys.stderr)
        sys.exit(1)

    try:
        w_str, h_str = cfg["resolution"].lower().split("x")
        width, height = int(w_str), int(h_str)
    except ValueError:
        print(
            f"Error: invalid resolution '{cfg['resolution']}' — expected WxH (e.g. 1280x720)",
            file=sys.stderr,
        )
        sys.exit(1)

    # Auto-generate output filename if not specified
    if not cfg["output"]:
        res = cfg["resolution"]
        fps = cfg["framerate"]
        br = cfg["bitrate"]
        dur = int(cfg["duration"])
        ext = CODEC_EXT.get(cfg["codec"], "mp4")
        cfg["output"] = f"{res}_{fps}fps_{br}bps_{dur}s.{ext}"

    nb = cfg["nb_colors"]
    palette = cfg["colors"]
    colors_hex = (palette * ((nb // len(palette)) + 1))[:nb]
    colors_arr = np.array([hex_to_rgb(c) for c in colors_hex], dtype=np.float32)

    font_size = cfg["font_size"] if cfg["font_size"] else max(24, height // 12)
    small_size = max(14, height // 28)
    font = ImageFont.truetype(FONT_PATH, font_size)
    small_font = ImageFont.truetype(FONT_PATH, small_size)
    framerate = cfg["framerate"]
    duration = cfg["duration"]
    total_frames = int(framerate * duration)
    speed = cfg["speed"]

    # Determine overlay text
    custom_text = cfg["text"]
    if custom_text is not None:
        # Single centered text block
        overlay_lines = []
    else:
        # Metadata overlay: one line per stat, rendered in small font
        output_path = cfg["output"]
        fmt = os.path.splitext(output_path)[1].lstrip(".") or "unknown"
        overlay_lines = [
            f"{width} x {height} px",
            f"{framerate} fps",
            fmt.upper(),
            cfg["codec"],
            f"{int(duration)}s",
        ]

    # Precompute static spatial component once; add scalar phase per frame
    t_base = compute_base(
        width, height, cfg["gradient_type"], cfg.get("linear_angle", 0)
    )

    # Precompute text position(s)
    dummy_img = Image.new("RGB", (width, height))
    dummy_draw = ImageDraw.Draw(dummy_img)
    font_color = cfg["font_color"]

    if custom_text is not None:
        bbox = dummy_draw.textbbox((0, 0), custom_text, font=font)
        text_x = (width - (bbox[2] - bbox[0])) // 2
        text_y = (height - (bbox[3] - bbox[1])) // 2
        text_positions = [(custom_text, font, text_x, text_y)]
    else:
        # Stack metadata lines centered, with small_font
        line_height = dummy_draw.textbbox((0, 0), "Ag", font=small_font)[3] + 6
        total_h = line_height * len(overlay_lines)
        start_y = (height - total_h) // 2
        text_positions = []
        for idx, line in enumerate(overlay_lines):
            bbox = dummy_draw.textbbox((0, 0), line, font=small_font)
            lx = (width - (bbox[2] - bbox[0])) // 2
            ly = start_y + idx * line_height
            text_positions.append((line, small_font, lx, ly))

    ffmpeg_cmd = [
        "ffmpeg",
        "-y",
        "-f",
        "rawvideo",
        "-pixel_format",
        "rgb24",
        "-video_size",
        f"{width}x{height}",
        "-framerate",
        str(framerate),
        "-i",
        "pipe:0",
        "-vcodec",
        cfg["codec"],
        "-b:v",
        cfg["bitrate"],
        "-t",
        str(duration),
        cfg["output"],
    ]

    proc = subprocess.Popen(ffmpeg_cmd, stdin=subprocess.PIPE)

    def render_frame(i):
        phase = (i / framerate) * speed
        t = (t_base + phase) % 1
        rgb_arr = phase_to_rgb(t, colors_arr)
        img = Image.fromarray(rgb_arr, "RGB")
        draw = ImageDraw.Draw(img)
        for line, fnt, lx, ly in text_positions:
            draw.text((lx, ly), line, font=fnt, fill=font_color)
        return img

    if proc.stdin:
        exit_code = render_via_stdin(proc, render_frame, total_frames)
    else:
        proc.kill()
        exit_code = render_via_tempfiles(cfg, render_frame, total_frames, framerate)

    sys.exit(exit_code)


if __name__ == "__main__":
    main()
