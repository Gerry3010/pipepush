#!/usr/bin/env sh
# Rasterize the icon SVG (single source of truth) into web/public/.
# The generated PNGs + favicon copy are gitignored build artifacts — this runs
# automatically before every `npm run build` (see package.json "prebuild") and
# inside the Docker web stage. Edit web/assets/icon.svg, never the PNGs.
set -eu

cd "$(dirname "$0")/.."   # -> web/
SVG="assets/icon.svg"
OUT="public"

if ! command -v rsvg-convert >/dev/null 2>&1; then
  echo "error: rsvg-convert not found — install librsvg" >&2
  echo "  Arch/Manjaro: pacman -S librsvg | Debian: apt-get install librsvg2-bin | Alpine: apk add rsvg-convert" >&2
  exit 1
fi

mkdir -p "$OUT"
rsvg-convert -w 192 -h 192 "$SVG" -o "$OUT/icon-192.png"
rsvg-convert -w 512 -h 512 "$SVG" -o "$OUT/icon-512.png"
rsvg-convert -w 180 -h 180 "$SVG" -o "$OUT/apple-touch-icon.png"
cp "$SVG" "$OUT/favicon.svg"

echo "icons generated -> $OUT/{icon-192,icon-512,apple-touch-icon}.png, favicon.svg"
