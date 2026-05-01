# Releasing Continuum

## Distribution

Continuum is a single statically-linked Go binary (`ctx`) with no runtime dependency beyond `git` and optional `gpg` for encrypted exports.

Primary release targets:

| Platform | Arch | Notes |
|---|---|---|
| macOS | arm64 | Apple Silicon primary target |
| macOS | amd64 | Intel Mac |
| Linux | amd64 | x86_64 servers and desktops |
| Linux | arm64 | Raspberry Pi and ARM servers |
| Windows | amd64 | Secondary target |

Release automation is handled by GitHub Actions and GoReleaser.

- push a `v*` tag to trigger `.github/workflows/release.yml`
- GoReleaser builds archives, checksums, GitHub draft releases, and a Homebrew formula update
- the direct install fallback is `install.sh`

Typical release flow:

```bash
go test ./...
git tag v0.6.0 -m "v0.6.0"
git push origin v0.6.0
```

Homebrew target:

```bash
brew tap simoneaveotti/continuum
brew install continuum
```

Direct install fallback:

```bash
curl -fsSL https://raw.githubusercontent.com/simoneaveotti/continuum/main/install.sh | bash
```

Continuum uses two repositories and two different branch roles:

- `continuum-dev`: private development repository
- `continuum`: public release repository

Within the local clone:

- `develop` is the real development branch with full private history
- `main` is a squashed public publication branch

This means public releases are intentionally published from a condensed `main`,
while the detailed development history remains on `develop` / `continuum-dev`.

## Remotes

Recommended remote layout:

```bash
origin  https://github.com/simoneaveotti/continuum.git
dev     https://github.com/simoneaveotti/continuum-dev.git
```

Recommended branch tracking:

```bash
main    -> origin/main
develop -> dev/develop
```

## Development Workflow

Normal work happens on `develop` and is pushed to `dev/develop`.

When the release automation needs validation before a public release:

1. commit the change on `develop`
2. push `develop` to `dev`
3. create an RC tag on `develop`
4. push the RC tag to `dev`

Example:

```bash
git checkout develop
git push dev develop
git tag -a v0.6.1-rc1 -m "v0.6.1 rc1"
git push dev v0.6.1-rc1
```

The `continuum-dev` release workflow validates:

- `go test ./...`
- multi-platform GoReleaser builds
- archive and checksum generation
- GitHub draft release creation

Homebrew publishing is optional in dev and should not block validation when the
tap token is not configured there.

## Public Release Workflow

Public releases are cut from `main`, but `main` is not developed directly.
Instead, each public release is a fresh squash of the current `develop`.

### 1. Prepare Public `main`

From a clean worktree:

```bash
git checkout main
git merge --squash develop
git commit -m "build: prepare public release vX.Y.Z"
```

This keeps the public branch concise while preserving the detailed private
history on `develop`.

### 2. Tag the Public Release

```bash
git tag -a vX.Y.Z -m "vX.Y.Z"
```

### 3. Publish

```bash
git push origin main
git push origin vX.Y.Z
```

The public `release.yml` workflow runs on `v*` tags and should publish:

- GitHub release artifacts
- checksums
- Homebrew formula updates

## Homebrew

The Homebrew tap repository is:

```text
simoneaveotti/homebrew-continuum
```

The public repo `continuum` should expose this GitHub Actions secret:

```text
HOMEBREW_TAP_TOKEN
```

That token must have write access to the tap repository itself, not just to the
main `continuum` repository.

Minimum required access:

- repository: `homebrew-continuum`
- permission: `Contents: Read and write`

## Retry Rule

If the release workflow fails due to credentials or publishing configuration,
do not mutate the existing tag. Fix the configuration, then create a new tag:

```bash
git checkout main
git tag -a vX.Y.Z+1 -m "vX.Y.Z+1"
git push origin vX.Y.Z+1
```

For dev validation:

```bash
git checkout develop
git tag -a vX.Y.Z-rcN -m "vX.Y.Z rcN"
git push dev vX.Y.Z-rcN
```

## Release Checklist

- Run `go test ./...` locally
- Push a version tag like `v0.6.0`
- Verify the GitHub Actions release workflow succeeds
- Review and publish the GitHub draft release
- Verify the Homebrew tap update if enabled
- Verify direct install and `ctx --version` on a clean machine
