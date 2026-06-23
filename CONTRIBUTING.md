# Contributing to Continuum

## Project Structure

Continuum is a single Go binary (`ctx`) with zero runtime dependencies beyond `git` and optional `gpg`.

```
cmd/              — main entry point
internal/         — all logic (no public API guarantees)
docs/             — ADRs, encryption, release process
templates/        — prompt templates used by ctx
```

## Development Workflow

Continuum uses a two-repository model (see `docs/RELEASING.md`):

- **`continuum-dev`** (private) — active development on `develop`
- **`continuum`** (public) — squashed releases published to `main`

Submit changes against the `develop` branch of `continuum-dev`.

### Quick start

```bash
git clone https://github.com/simoneaveotti/continuum-dev.git
cd continuum
go build -o ctx ./cmd/
go test ./...
```

## Running Tests

```bash
go test ./... -count=1
```

All 16 packages must pass before a change is ready.

## Code Conventions

- **No external public API** — packages in `internal/` are not importable outside the module.
- **Small packages** — a package should do one thing. Split when it grows beyond ~400 lines.
- **No init()** — explicit constructors only.
- **No global state** — dependencies are passed explicitly or resolved through package-level constructors.
- **Error handling** — use `die(err)` in `cmd/` for fatal errors; return errors everywhere else.
- **No unused code** — delete wrappers that only duplicate the underlying call; keep them only if they add clarity.
- **Test patterns** — use `t.Run` subtables where possible; prefer `-count=1` to avoid caching issues.
- **Confirmations** — use `internal/prompt.Confirm` for y/n prompts; never open `/dev/tty` directly.
- **Markdown parsing** — use `internal/parse` instead of ad-hoc section extraction.

## What Makes a Good PR

1. Addresses a single concern.
2. Includes or updates tests.
3. All existing tests pass.
4. Solves a real problem — ask first if unsure.

## Release Cadence

Releases are cut from `main` by squashing `develop`. See `docs/RELEASING.md` for the full process.

## License

MIT — see `LICENSE`.
