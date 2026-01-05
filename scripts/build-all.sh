#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN_DIR="$ROOT/bin"
mkdir -p "$BIN_DIR"

# Platforms to build for
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

# Collect commands found under cmd/
CMD_DIRS=()
for d in "$ROOT"/cmd/*; do
  [ -d "$d" ] || continue
  CMD_DIRS+=("$(basename "$d")")
done

if [ ${#CMD_DIRS[@]} -eq 0 ]; then
  echo "No commands found under cmd/ to build. Exiting."
  exit 1
fi

echo "Building ${#CMD_DIRS[@]} command(s): ${CMD_DIRS[*]}"

echo "Platforms: ${PLATFORMS[*]}"

for plat in "${PLATFORMS[@]}"; do
  GOOS=${plat%/*}
  GOARCH=${plat#*/}

  for cmd in "${CMD_DIRS[@]}"; do
    # Place binary directly in $BIN_DIR with filename: <cmd>-<os>-<arch>
    outfile="$BIN_DIR/${cmd}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then outfile="${outfile}.exe"; fi

    echo "-> Building $cmd for $GOOS/$GOARCH -> $outfile"
    # Disable CGO for maximum portability, strip debug info
    env CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags "-s -w" -o "$outfile" "./cmd/$cmd"
  done
done

echo "Builds complete. Binaries available under: $BIN_DIR"
