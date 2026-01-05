timp — Terminal Image Manipulation Program

Overview

`timp` is a lightweight image-processing CLI and a pure-Go image engine. The project provides a user-facing CLI in `cmd/timp` and an in-repo stdlib engine in `pkg/stdimg`. CLI helpers and UX utilities live in `pkg/cli`.

Repository layout

- `cmd/timp` — CLI entrypoint and `main` package.
- `pkg/cli` — CLI helpers and UX code used by the command.
- `pkg/stdimg` — pure-Go image engine primitives and command registry.
- `scripts/build-all.sh` — cross-build script that produces platform binaries into `bin/`.
- `bin/` — output directory (created by the build script).
- `LICENSE.txt` — project license.

Quick start

- Build locally:

  `go build ./cmd/timp`

- Cross-build (creates per-platform binaries under `bin/`):

  `./scripts/build-all.sh`

Install

- From source (module-aware):

  `go install github.com/Fepozopo/timp/cmd/timp@latest`

Usage

- General pattern:

  `timp`

They're running the command you'll be prompted with commands. You'll need to open an image and then select a command.

Development

- Run tests:

  `go test ./...`

- Format code:

  `gofmt -w .`

- Vet code:

  `go vet ./...`

- Run the full build script:

  `./scripts/build-all.sh`

Contributing

- Fork, branch, implement, run `go test ./...`, and open a PR.
- Keep public API changes minimal; prefer small, reviewable commits.

- Binaries produced by `scripts/build-all.sh` are saved under `bin/` as `bin/<cmd>-<os>-<arch>` (Windows binaries have `.exe`).

License

See `LICENSE.txt` for license terms.
