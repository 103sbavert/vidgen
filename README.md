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
| `text`          | string\|null | `null`          | Static text overlay. If `null`, shows video metadata (resolution, fps, format, codec, duration).                                                                                       |
| `font_size`     | int\|null    | `null`          | Font size in pixels. If `null`, auto-scaled to `height / 12` (e.g. 60px at 720p).                                                                                                      |
| `font_color`    | string       | `"black"`       | Text color — HTML name or `#RRGGBB`.                                                                                                                                                   |
| `gradient_type` | string       | `"radial"`      | One of: `linear`, `radial`, `circular`, `spiral`, `square`. Only `radial` is implemented so far.                                                                                       |
| `linear_angle`  | int          | `0`             | Rotation of the `linear` gradient in degrees (0 = horizontal, 90 = vertical). Has no effect on other gradient types.                                                                   |
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
> `text`, `font_size`, `font_color`, `linear_angle`, `speed` and `seed` are
> accepted and validated but not used by any engine yet.

![palette](palette.png)

### Performance

The default `native` engine renders frames in Go and streams raw `yuv420p`
(BT.709, limited range, tagged as such) into ffmpeg, which only encodes. A
gradient depends only on distance-from-center and time, so a distance map is
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

### Project layout

| Package        | Role                                                                                     |
| -------------- | ---------------------------------------------------------------------------------------- |
| `config`       | Load, default, validate the JSON config; codec table; output path.                      |
| `gradient`     | Engine-independent gradient math (`Radial.ColorAt` is the canonical definition).        |
| `render`       | Native frame generation: distance `Field`, per-frame color `Table`, ordered `Stream`.   |
| `ffmpeg`       | `Encode`: spawns ffmpeg, plans segments, atomic output. `ffmpeg/geq` is the slow engine. |
| `progress`     | Terminal progress line fed by a per-frame callback.                                      |

Adding a gradient type: add its math under `gradient`, a `render` source that
reuses `Field.Fill` and `Stream`, and a case in `ffmpeg.newSource` (plus a geq
builder if the reference engine should support it).

### Gradient types

| Type       | Shape                    | Animation                            |
| ---------- | ------------------------ | ------------------------------------ |
| `linear`   | Parallel color bands     | Bands slide along the gradient slope |
| `radial`   | Soft rings from center   | Rings expand outward                 |
| `circular` | Tighter concentric rings | Rings expand outward (3× frequency)  |
| `spiral`   | Archimedean spiral       | Spiral rotates around center         |
| `square`   | Concentric rectangles    | Rectangles expand outward            |

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
