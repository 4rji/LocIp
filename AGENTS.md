# Repository Guidelines

## Project Structure & Module Organization

This is a small Go CLI module for IP geolocation lookups.

- `locip.go` contains the CLI entry point, argument parsing, colorized output, ipinfo.io requests, and GeoLite2 database flow.
- `locip_test.go` contains unit tests for argument parsing, URL generation, HTTP handling, help output, file input, and color behavior.
- `README.md` documents user-facing usage; keep it aligned with `printUsage()` in `locip.go`.
- `img/` and `menu.webp` hold visual assets used by the README.
- `build.sh` cross-compiles release binaries into `builds/` and generates SHA-256 checksums.

## Build, Test, and Development Commands

- `go test ./...` — run all Go tests.
- `go run . -h` — show the CLI help/menu locally.
- `go run . 8.8.8.8` — run an online ipinfo.io lookup.
- `go run . -d 1.1.1.1` — run a local GeoLite2 lookup using the default database path.
- `go build -o locip .` — build a local binary.
- `./build.sh` — create multi-platform release artifacts in `builds/`.

Agent note: do not build after changes unless the user explicitly asks. Prefer static review and tests when validation is needed.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt -w locip.go locip_test.go` after edits. Keep package-level helpers small and focused. Use lowerCamelCase for unexported functions and types, and reserve exported names only for APIs that must be public. Keep CLI flags and help text consistent with the actual parser behavior.

## Testing Guidelines

Tests use Go's standard `testing` package. Name tests as `Test<Behavior>` and prefer behavior-focused cases such as `TestParseArgsDefaultsToOnlineTarget`. When changing CLI behavior, update or add tests in `locip_test.go`, especially for parsing, output, HTTP failure paths, and `NO_COLOR` handling.

## Commit & Pull Request Guidelines

Recent history contains terse non-conventional messages, so do not copy it. New commits must use conventional commits, for example `fix: handle ipinfo http errors` or `docs: align README usage menu`. Never add `Co-Authored-By` or AI attribution.

Pull requests should include a concise summary, commands run, user-visible CLI changes, linked issues when applicable, and README/screenshot updates if menu output changes.
