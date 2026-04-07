# Hosted Installer And Release Layout

## Goal

Make `atami` behave like a normal CLI app:

- install with a single hosted command
- update with the same command
- no local repo checkout required
- no Go toolchain required for end users
- clean upgrade messaging from inside the CLI

Target user experience:

```sh
curl -fsSL https://raw.githubusercontent.com/Kosmic/atami-ai/main/scripts/install-atami-release | sh
```

And when an update is available:

```text
Update available: atami 0.3.1 (current: 0.3.0)
Run: curl -fsSL https://raw.githubusercontent.com/Kosmic/atami-ai/main/scripts/install-atami-release | sh
```

## Principles

- Keep developer install and end-user install separate.
- Release builds should install prebuilt binaries, not build from source.
- The hosted installer must be stable and human-readable.
- The CLI should know how it was installed so it can recommend the right update command.
- `atami kb init` and `atami kb skills pull` should continue fetching runtime content from GitHub, independently of binary releases.

## Proposed Install Model

Use two install paths with distinct purposes:

1. Development install
   File: [`scripts/install-atami`](../scripts/install-atami)
   Purpose: local development from a checkout, uses `go install ./atami`

2. Hosted release install
   New file: `scripts/install-atami-release`
   Purpose: end-user install from published release binaries
   Invocation:

   ```sh
   curl -fsSL https://raw.githubusercontent.com/Kosmic/atami-ai/main/scripts/install-atami-release | sh
   ```

The hosted installer becomes the canonical public install/update path.

## Release Artifact Layout

Use GitHub Releases on `Kosmic/atami-ai`.

For each version tag like `v0.3.0`, publish these assets:

- `atami_darwin_amd64.tar.gz`
- `atami_darwin_arm64.tar.gz`
- `atami_linux_amd64.tar.gz`
- `atami_linux_arm64.tar.gz`
- `checksums.txt`

Each archive should contain:

- `atami`
- `LICENSE` if needed later
- optionally `README.md` if useful, but not required

Avoid `.zip` unless Windows support is added.

## Hosted Installer Responsibilities

`scripts/install-atami-release` should:

1. Detect OS and architecture.
2. Resolve the latest release version.
3. Download the matching release tarball.
4. Download `checksums.txt`.
5. Verify the archive checksum.
6. Extract `atami`.
7. Install it into a user-writable location.
8. Print any PATH instructions if required.
9. Write install metadata for the CLI.

Recommended install location order:

1. `$ATAMI_INSTALL_DIR` if explicitly set
2. `$HOME/.local/bin` on Linux
3. `$HOME/bin` on macOS if preferred by your team
4. fallback to `$HOME/.local/bin`

The script should not require `sudo`.

## Install Metadata

Write a small metadata file after install so the CLI knows how to recommend updates.

Recommended path:

- macOS/Linux: `~/.config/atami/install.json`

Suggested shape:

```json
{
  "channel": "script",
  "installed_version": "0.3.0",
  "update_command": "curl -fsSL https://raw.githubusercontent.com/Kosmic/atami-ai/main/scripts/install-atami-release | sh"
}
```

This removes guesswork from update messaging.

## Versioning

Stop relying on a hardcoded dev version for release builds.

Recommended model:

- keep default version in code as `dev`
- inject release version at build time with `-ldflags`

Example build flag:

```sh
go build -ldflags "-X github.com/atami-ai/atami-ai/cli/cmd.version=0.3.0"
```

To support this, `cmd.version` must become a mutable `var`, not a `const`.

## Update Check Design

The CLI should check for updates after command execution, not before.

Behavior:

1. Read local install metadata.
2. If channel is `script`, check the latest GitHub release version.
3. Cache the result for 24 hours.
4. If the latest version is newer than the running version, print a short notice to `stderr`.

Recommended cache file:

- `~/.config/atami/update-check.json`

Suggested shape:

```json
{
  "last_checked_at": "2026-04-07T12:00:00Z",
  "latest_version": "0.3.1"
}
```

The update check should be best-effort:

- never fail the command if update lookup fails
- keep messages short
- skip update checks for `dev` builds

## How To Resolve Latest Release

Preferred source:

- GitHub Releases API for `Kosmic/atami-ai`

The CLI and installer only need:

- latest version tag
- release asset download URLs

This keeps versioning separate from the branch tip used by `kb init` and `kb skills pull`.

## GitHub Actions Release Pipeline

Add a release workflow that runs on version tags like `v*`.

Pipeline steps:

1. Run `go test ./...`
2. Run `go vet ./...`
3. Build binaries for target OS/arch pairs
4. Package tarballs
5. Generate `checksums.txt`
6. Create or update the GitHub Release
7. Upload all assets

Optional later:

- artifact signing
- SBOM generation
- notarization if macOS distribution requires it

## Script Layout

Recommended file split:

- [`scripts/install-atami`](../scripts/install-atami)
  Local developer install from source

- `scripts/install-atami-release`
  Hosted public installer for end users

- optional later: `scripts/lib/install-common.sh`
  Shared shell helpers if both scripts start sharing logic

Keep the release installer independent from the repo checkout.

## CLI Changes Required

1. Change `cmd.version` from `const` to `var`.
2. Add a small internal package for install metadata:
   `internal/installmeta/`
3. Add a small internal package for update checks:
   `internal/updatecheck/`
4. Call the update checker from the command execution path after command completion.
5. Suppress update notices for:
   - `dev` builds
   - explicit local/source installs if you want different behavior later

## Rollout Plan

### Phase 1

- Keep current local source installer for development.
- Add release installer script.
- Add release workflow and binary publishing.

### Phase 2

- Add install metadata write/read.
- Add update check and update notice output.

### Phase 3

- Switch docs to prefer hosted installer everywhere.
- Keep local installer documented only for contributors.

## Security And Integrity

Minimum acceptable:

- checksum verification against `checksums.txt`

Stronger later:

- signed checksums
- release signature verification

For now, checksum verification is enough to avoid accidental corruption and incomplete downloads.

## Recommended Documentation Changes

Once implemented:

- top-level [`README.md`](../README.md) should prefer the hosted installer
- [`cli/README.md`](../cli/README.md) should separate contributor setup from end-user install
- local source install should move under a "Development" section

## Open Questions

1. Which install directory should be the default on macOS: `~/.local/bin` or `~/bin`?
2. Do you want update notices on every command after cache expiry, or only on mutating commands?
3. Should `atami --version` also print the install channel and latest-known version later?
4. Do you want release binaries for Linux only initially, or macOS and Linux together from day one?

## Recommendation

Build the hosted installer around GitHub Releases, not around `go install`.

That gives you:

- the cleanest install path
- the cleanest update path
- a stable version source
- normal CLI distribution semantics

The local repo installer should remain, but only as a contributor/development tool.
