# vidgen

Generates synthetic test videos with animated gradient backgrounds and a text
overlay.

> [!IMPORTANT]
> This branch is under a re-write in Go.

## Requirements

- Go 1.26+
- FFmpeg (must be on `PATH`)

## Usage

```bash
go run .                # all defaults
go run . -c config.json # load config from file
```

The only CLI option is `-c`/`--config`, pointing to a JSON file. All keys are
optional — missing or `null` values fall back to defaults. See `sample.json`
for a ready-to-copy template.

## Config reference

| Key             | Type         | Default         | Description                                                                                                                                                                            |
| --------------- | ------------ | --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `output`        | string\|null | `null`          | Output path. If `null`, auto-named `{gradient_type}_{res}_{fps}fps_{bitrate}bps_{duration}s.{ext}`¹ where ext is inferred from codec. Taken auto-names gain `_1`, `_2`, … until free.* |
| `resolution`    | string       | `"1920x1080"`   | Frame size as `WxH`.                                                                                                                                                                   |
| `framerate`     | int          | `24`            | Frames per second.                                                                                                                                                                     |
| `bitrate`       | int          | `2097152`       | Video bitrate in bits per second.                                                                                                                                                      |
| `codec`         | string       | `"libx264"`     | FFmpeg video codec. See codec table below.                                                                                                                                             |
| `duration`      | int          | `30`            | Length in seconds.                                                                                                                                                                     |
| `text`          | string\|null | `null`          | Static text overlay, centered; `\n` starts a new line. If `null`, shows video metadata (resolution, fps, format, codec, duration). An empty string `""` draws no text.³                 |
| `font_size`     | int\|null    | `null`          | Font size in pixels. If `null`, auto-scaled: `height / 12` (min 24) for custom text, `height / 28` (min 14) for the metadata block.                                                     |
| `font_color`    | string       | `"black"`       | Text color: `RRGGBB` hex (optional `#`) or one of `black`, `white`, `red`, `green`, `blue` (soft shades, not pure).                                                                    |
| `gradient_type` | string       | `"radial"`      | One of: `linear`, `radial`, `circular`, `spiral`, `square`. `linear` and `radial` are implemented so far.                                                                              |
| `linear_angle`  | int          | `0`             | Direction of the `linear` gradient in degrees, 0–360 (0 = left→right, 90 = top→bottom, 180 = right→left, 270 = bottom→top). Has no effect on other gradient types.                    |
| `colors`        | string[]     | pastel palette² | List of hex color strings (without `#`).                                                                                                                                               |
| `nb_colors`     | int          | `4`             | How many colors to use from `colors`. Cycles if fewer than requested.                                                                                                                  |
| `speed`         | float        | `0.08`          | Animation speed in color-cycles per second.                                                                                                                                            |
| `seed`          | —            | `null`          | Reserved, currently unused.                                                                                                                                                            |
| `engine`        | string       | `"native"`      | `native`: frames rendered in Go, piped to ffmpeg for encoding (fast). `geq`: single ffmpeg `geq` filter (slow reference path).                                                         |
| `workers`       | int          | `0`             | Render goroutines for the native engine. `0` = number of CPUs.                                                                                                                         |
| `segments`      | int          | `0`             | Parallel encode segments for the native engine. `0` = auto, `1` = off. See [Performance](#performance).                                                                                |
| `preset`        | string\|null | `null`          | Passed to ffmpeg as `-preset` when set (e.g. `veryfast` for libx264/libx265).                                                                                                          |

¹ An explicit `output` is used as-is; if that file already exists vidgen stops
with an error instead of overwriting it. Path expansion (environment
variables, output directories) is not implemented yet.

² Default palette: `546B41`, `99AD7A`, `DCCCAC`, `FFF8EC`

> [!NOTE]
³ Text is rendered with the bundled Noto Sans Regular, anti-aliased, and drawn
identically by both engines (see [Text overlay](#text-overlay)).

> [!NOTE]
> `speed` and `seed` are accepted and validated but not used by any engine
> yet.

![palette](palette.png)

### Performance

The default `native` engine renders frames in Go and streams raw `yuv420p`
(BT.709, limited range, tagged as such) into ffmpeg, which only encodes. A
gradient's color depends only on a per-pixel position (distance from the
center, or distance along a direction) and time, so the position map is
computed once and each frame needs just a small color lookup table plus a
table lookup per pixel, spread over all cores. A 30 s 1080p24 clip went from
about 3.5 min with `geq` to about 3 s (libx264, 16 cores); encoding is now the
bottleneck.

`segments` additionally splits the clip into independent parts, each with its
own render pipeline and ffmpeg process, then joins them with a lossless
stream-copy concat. This helps encoders that scale poorly across threads and
is chosen automatically for `libvpx`, `libvpx-vp9` and `libaom-av1`
(3-5x faster). It does nothing for `libx264`, `libsvtav1` or `mpeg4`, which
already saturate the CPU, and gives ~25% for `libx265` (opt in with
`"segments": 2`). Every segment starts with a keyframe and restarts rate
control, so the achieved bitrate and file size differ somewhat from a single
pass.

Output is written to a hidden `.name.partial.ext` file and renamed on success,
so an interrupted run (Ctrl-C) leaves no truncated file.

`scripts/bench.sh [duration] [resolution] [codec...]` times each codec at 1, 2,
4 and 8 segments.

### Text overlay

Text is laid out and rasterized once (`caption` package, bundled Noto Sans via
`fonts`) into an anti-aliased coverage mask. The native engine blends that mask
straight into each `yuv420p` frame, which touches only the pixels the text
covers and costs well under 1 ms per frame. The `geq` engine writes the same
mask as a transparent PNG and lets ffmpeg's `overlay` filter composite it, so
both engines draw the same glyphs in the same place.

With `text: null` the overlay is a five-line metadata block:

```
1920 x 1080 px
24 fps
MP4
libx264
30s
```

### Project layout

| Package        | Role                                                                                     |
| -------------- | ---------------------------------------------------------------------------------------- |
| `config`       | Load, default, validate the JSON config; codec table; output path.                      |
| `gradient`     | Engine-independent gradient math: `Shape` (radial, linear) + `Bands` colors (`Bands.ColorAt` is the canonical definition). |
| `render`       | Native frame generation: position `Field`, per-frame color `Table`, text `Overlay`, ordered `Stream`. |
| `caption`      | Text layout and rasterization into a coverage mask.                                      |
| `fonts`        | The embedded typeface.                                                                   |
| `ffmpeg`       | `Encode`: spawns ffmpeg, plans segments, atomic output. `ffmpeg/geq` is the slow engine. |
| `progress`     | Terminal progress line fed by a per-frame callback.                                      |

Adding a gradient type: add a `Shape` and a case in `gradient.New`, plus a
matching expression in `ffmpeg/geq`'s `shapeExpr`. The native renderer
(`render.Banded`), the pipeline and the encoder need no changes; a test in
`ffmpeg` compares both engines frame by frame.

### Gradient types

Implemented gradients share one animation: the bands start at unequal widths
(the first color widest) and drift toward equal widths over the clip, with soft
S-curve edges.

| Type       | Shape                    | Animation                            | Status      |
| ---------- | ------------------------ | ------------------------------------ | ----------- |
| `linear`   | Parallel stripes, first color at the start edge (see `linear_angle`) | Stripes drift toward equal width | implemented |
| `radial`   | Soft rings from center, first color outermost | Rings drift toward equal width | implemented |
| `circular` | Tighter concentric rings (3× frequency) | planned | not yet |
| `spiral`   | Archimedean spiral | planned | not yet |
| `square`   | Concentric rectangles | planned | not yet |

### Codec → container mapping

| Codec                                                                                                                                                                                                             | Container | Extension |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | --------- |
| `libx264`, `libx265`, `h264_nvenc`, `h264_amf`, `h264_qsv`, `h264_videotoolbox`, `hevc_nvenc`, `hevc_amf`, `hevc_qsv`, `hevc_videotoolbox`, `libaom-av1`, `libsvtav1`, `av1_nvenc`, `av1_amf`, `av1_qsv`, `mpeg4` | MP4       | `.mp4`    |
| `libvpx`, `libvpx-vp9`, `vp9_qsv`                                                                                                                                                                                 | WebM      | `.webm`   |
| `mpeg2video`, `mpeg2_qsv`                                                                                                                                                                                         | MPEG-PS   | `.mpg`    |
| `libxvid`, `mjpeg`, `mjpeg_qsv`                                                                                                                                                                                   | AVI       | `.avi`    |
| `prores`, `prores_ks`                                                                                                                                                                                             | QuickTime | `.mov`    |
| `dnxhd`                                                                                                                                                                                                           | MXF       | `.mxf`    |
| `libtheora`                                                                                                                                                                                                       | Ogg       | `.ogv`    |
| `libwebp`, `libwebp_anim`                                                                                                                                                                                         | WebP      | `.webp`   |

Unknown codecs are rejected at config load.
