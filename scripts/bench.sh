#!/bin/sh
# End-to-end timing of vidgen across codecs and segment counts.
#
# usage: scripts/bench.sh [duration_s] [resolution] [codec...]
#   defaults: 6 1920x1080 libx264 libx265 mpeg4 libvpx-vp9 libsvtav1
#
# Builds the binary into a temp dir, renders into it, prints wall time, packet
# count (must equal fps*duration) and file size per (codec, segments) pair.
set -eu

duration="${1:-6}"
res="${2:-1920x1080}"
[ "$#" -gt 2 ] && shift 2 || shift "$#"
codecs="${*:-libx264 libx265 mpeg4 libvpx-vp9 libsvtav1}"

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

(cd "$root" && go build -o "$work/vidgen" .)

for codec in $codecs; do
    ext=mp4
    case "$codec" in libvpx*) ext=webm ;; esac

    for seg in 1 2 4 8; do
        out="$work/out.$ext"
        rm -f "$out"
        printf '{"output":"%s","duration":%s,"resolution":"%s","codec":"%s","segments":%s}' \
            "$out" "$duration" "$res" "$codec" "$seg" >"$work/cfg.json"

        start="$(date +%s%N)"
        rc=0
        "$work/vidgen" -c "$work/cfg.json" >/dev/null 2>&1 || rc=$?
        end="$(date +%s%N)"

        packets="$(ffprobe -v error -count_packets -select_streams v \
            -show_entries stream=nb_read_packets -of csv=p=0 "$out" 2>/dev/null || echo "?")"
        size="$(stat -c %s "$out" 2>/dev/null || echo "?")"
        ms="$(((end - start) / 1000000))"

        echo "$codec segments=$seg rc=$rc time=${ms}ms packets=$packets size=$size"
    done
done
