# RNS 510 Bootlogo Toolkit — Go edition

This is the native Go rewrite under active development. The original Python implementation remains in the repository and is not modified.

## Targets

- Linux x86-64
- Fyne desktop GUI and standalone CLI
- AppImage as the primary Linux distribution format
- No Python or pip runtime dependency
- ISO generation uses xorriso; the AppImage build bundles xorriso and its runtime dependencies where possible.

## Current Go commands

```sh
go run ./cmd/rns510 encode input.png -o LOGO.FLI
go run ./cmd/rns510 verify LOGO.FLI
go run ./cmd/rns510 build-iso input.png -o bootlogo.iso --template template_swl
go run ./cmd/rns510-gui
```

Supported input formats currently follow Go's standard image decoders (PNG and JPEG). The initial encoder uses nearest-neighbour scaling; this will be improved and cross-checked against reference outputs before the rewrite is considered feature-complete.

## Build

```sh
go test ./...
go build -o dist/rns510 ./cmd/rns510
go build -o dist/rns510-gui ./cmd/rns510-gui
```

## AppImage

The GitHub Actions workflow in `.github/workflows/appimage.yml` builds an x86-64 AppImage on Ubuntu. Run it from a tagged release or manually through Actions. AppImage compatibility is bounded by the glibc baseline of the build image and the host's graphics/runtime stack; test on supported distributions before release.

## Important format note

The Go implementation follows the byte layout used by the original source serializer/parser, not the simplified header table in the README. In particular, the original code stores CRC1 at bytes 18–19, CRC2 at bytes 50–51, dimensions at bytes 40–47, and the RLE payload length at bytes 60–63. These offsets are covered by Go tests and should be validated against known-good OEM and generated files before flashing.
