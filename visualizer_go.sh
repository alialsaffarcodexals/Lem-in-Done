#!/usr/bin/env bash
# Build-and-run wrapper for the Go PNG visualizer.
# Usage examples:
#   ./lem-in example00.txt | ./visualizer_go.sh frames
#   go run . example00.txt | ./visualizer_go.sh frames
#
# This will produce PNG frames in the specified output directory (default: frames).

set -euo pipefail

OUTDIR="${1:-frames}"

# Build to a temp binary (avoids interfering with your lem-in module)
BIN="$(mktemp -t lemvis-XXXXXX)"
trap 'rm -f "$BIN"' EXIT

# Compile and run
go build -o "$BIN" visualizer.go
cat - | "$BIN" -out "$OUTDIR"

echo "PNG frames written to: $OUTDIR"
echo "Example to preview the first frame:"
echo "  open $OUTDIR/turn_0000.png   # macOS"
echo "  xdg-open $OUTDIR/turn_0000.png # Linux"
