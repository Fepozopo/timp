# timp — Terminal Image Manipulation Program

## Overview

`timp` is a lightweight image-processing CLI and a pure-Go image engine. The project provides a user-facing CLI in `cmd/timp` and an in-repo stdlib engine in `pkg/stdimg`. CLI helpers and UX utilities live in `pkg/cli`.

## Repository layout

- `cmd/timp` — CLI entrypoint and `main` package.
- `pkg/cli` — CLI helpers and UX code used by the command.
- `pkg/stdimg` — pure-Go image engine primitives and command registry.
- `scripts/build-all.sh` — cross-build script that produces platform binaries into `bin/`.
- `bin/` — output directory (created by the build script).
- `LICENSE` — project license.

## Quick start

- Build locally:

  `go build ./cmd/timp`

- Cross-build (creates per-platform binaries under `bin/`):

  `./scripts/build-all.sh`

## Install

- From source (module-aware):

  `go install github.com/Fepozopo/timp/cmd/timp@latest`

## Usage

timp is an interactive CLI image editor. Run the `timp` command to start the program and follow the on-screen prompts.

Typical interactive flow:

1. Start the tool:

   `timp`

2. Open an image when prompted by entering a filesystem path (absolute or relative).
3. Choose an operation from the command list (filters, resizing, annotate, composite, etc.).
4. Provide any command arguments when asked (width/height, intensity, text, colors).
5. Preview (when available) and save/export the edited image to a new file.

Quick example session (illustrative):

- Run `timp` and open `images/photo.jpg`.
- Select the `resize` command and enter a new width (e.g., `800`) and height (or leave blank to keep aspect ratio).
- Select `save` or `export` and write `images/photo-resized.jpg`.

Notes and tips:

- The CLI prompts are context-aware: commands and required arguments are shown while you work.

### Metadata support

timp currently handles image metadata (EXIF/XMP/other tags) only for JPEG/JPG files. That means:

- When you open and save a JPEG, the tool preserves the image's metadata.
- Opening or saving PNG, TIFF, or other formats may strip or ignore metadata because those formats (PNG text chunks, XMP, TIFF tags) are not implemented in the engine.
- Converting an image from one format to another may result in lost metadata.

## Development

- Run tests:

  `go test ./...`

- Format code:

  `gofmt -w .`

- Vet code:

  `go vet ./...`

- Run the full build script:

  `./scripts/build-all.sh`

## Contributing

- Fork, branch, implement, run `go test ./...`, and open a PR.
- Keep public API changes minimal; prefer small, reviewable commits.

- Binaries produced by `scripts/build-all.sh` are saved under `bin/` as `bin/<cmd>-<os>-<arch>` (Windows binaries have `.exe`).

## License

See `LICENSE` for license terms.
