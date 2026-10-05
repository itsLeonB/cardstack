#!/usr/bin/env bash
# Builds assets/series/<series code>.webp from the Bulbagarden Archives
# originals (docs/adr/0017): WebP with alpha, 80px tall, width by aspect ratio.
# Needs curl and vips. Run from anywhere; commit the resulting files, then
# upload them with `make host-series-images`.
#
# File names are the Series codes slugifySeries produces (pokemonasia/mapper.go).
set -euo pipefail

out="$(cd "$(dirname "$0")/.." && pwd)/assets/series"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out"

ua='cardstack-logo-build/1.0 (https://github.com/itsLeonB/cardstack)'
base=https://archives.bulbagarden.net/media/upload

fetch() { curl -fsSL -A "$ua" -o "$tmp/$1.png" "$base/$2"; }

fetch scarlet-violet 3/3b/SV_Era_Logo_I.png
fetch pedang-perisai 6/6c/Sword_Shield_Logo_Indonesian.png
fetch evolusi-mega d/d3/Mega_Evolution_Logo_Indonesian.png
fetch matahari-bulan 7/71/First_Impact_Logo_Indonesian.png

# Sun & Moon is the First Impact set logo: keep only the series wordmark on
# top (rows 0-182 of 1177x298; rows 182-188 are a fully transparent gap above
# the set name).
vips crop "$tmp/matahari-bulan.png" "$tmp/matahari-bulan-crop.png" 0 0 1177 182
mv "$tmp/matahari-bulan-crop.png" "$tmp/matahari-bulan.png"

for f in "$tmp"/*.png; do
  code="$(basename "$f" .png)"
  # The huge width means the 80px height is the binding constraint, so the
  # width follows the aspect ratio. The webp saver keeps alpha (no flatten).
  vips thumbnail "$f" "$out/$code.webp[Q=90]" 100000 --height 80
  echo "$code: $(vipsheader -f width "$out/$code.webp")x$(vipsheader -f height "$out/$code.webp"), $(vipsheader -f bands "$out/$code.webp") bands"
done
