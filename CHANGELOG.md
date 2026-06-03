# Changelog

All notable user-facing and internal changes to go-ios are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

Releases are cut by dispatching the **Release-Go-iOS** workflow (Actions tab →
Run workflow, or `gh workflow run release.yml -f release_notes=... -f bump=patch`).
That workflow computes the next version, writes a new `## [x.y.z] - DATE` section
here from the notes you provide, and uses the same text as the GitHub release
body — so this file is updated automatically and never needs a manual PR.
Use `## [Unreleased]` to jot down notable changes between releases if you like.

## [Unreleased]

### 🐛 Fixes
- **REST API `KillApp`** — fixed kill-by-bundle-ID, which stopped matching after `CFBundleIdentifier`/`CFBundleExecutable` became method calls on the app model. (#729)

## [1.0.217] - 2026-06-03

First release published to npm since `1.0.213` — npm publishing broke when npm
revoked legacy tokens, and is now restored on secure OIDC trusted publishing.
This release also makes the npm package install correctly on Windows for the
first time.

### ✨ New features & improvements
- **`ostrace --follow`** — persistent log streaming that survives process restarts. Auto-reconnects when the target process exits/restarts: with `--process=<name>` it re-resolves the new PID, with an explicit `--pid` it exits cleanly when that PID ends. (#719)
- **`ostrace` plain-text output** — `--nojson` now prints structured, human-readable lines `[timestamp] PID:##### <Level> [subsystem:category] message` instead of raw JSON, with ANSI color coding (Info=cyan, Debug=gray, Error=red, Fault=bold red). Colors are emitted only when stdout is a TTY, so pipes and files stay clean. (#718)
- **More reliable Developer Disk Image (DDI) mounting on iOS 17+** — fixes `identity-not-found` failures on newer chips (e.g. A19 Pro) and adds a manifest fast-path that skips the nonce + Apple TSS round-trip on re-mounts when a valid personalization manifest already exists. (#720, #723)

### 🐛 Fixes
- **Windows npm install** — `npm i -g go-ios` previously installed a binary that wasn't on `PATH` (the postinstall placed it in a non-PATH `bin` subfolder), so `ios` was uncallable on Windows. The binary is now installed to the correct location. (#730)
- **macOS 26 binary launch** — bump the Go toolchain to 1.24.13 so produced binaries carry an `LC_UUID` load command; without it, macOS 26's dyld rejects them with `abort trap`. (#723)

### 🔧 Build, CI & internals
- **npm publishing migrated to OIDC trusted publishing** — no more long-lived `NODE_AUTH_TOKEN`; releases authenticate per-run via GitHub OIDC, with provenance attestations. (#727)
- **Cross-platform install verification** — release and canary pipelines now install the freshly published package on Windows, Linux and macOS and run the binary, asserting the version matches. (#728, #730)
- **Canary release pipeline** — manual-dispatch workflow publishing a throwaway `go-ios-canary` package to validate the full release flow without shipping. (#727)
- **Releases now gated behind the `make-release` label.** (#726)
- **Real-device e2e test suite** on the self-hosted macOS/Linux runners, gated to run only after unit tests pass, and runnable on fork PRs via a maintainer `/test-devices` comment. (#723, #725, #726)
