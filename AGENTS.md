# AGENTS.md

Guidance for AI agents working **on the go-ios codebase**. Keep changes minimal
and idiomatic; match the style of surrounding code.

(Looking for how to *use* the `ios` CLI to automate a device? See the README and
`ios help` — this file is about developing go-ios itself.)

## Build & test

- Build: `go build ./...` (CI uses `make build`, which also wires up the
  `go.work` workspace). Use the Go toolchain pinned in `go.mod`.
- Unit tests: `go test ./...` (device-free; this is what CI runs).
- Device/integration testing lives only in the `test/e2e/` suite, gated by the
  `e2e` build tag and `GO_IOS_E2E_DEVICES`, and runs on the self-hosted CI
  runners. Put device-dependent tests there, not in the package unit tests.

## Real-device CI

- PRs from branches **in this repo** run the real-device e2e suite
  automatically, after the unit tests pass.
- PRs **from forks** do not (untrusted code can't get the self-hosted runners
  and device secrets). A maintainer runs them by commenting **`/test-devices`**
  on the PR; only OWNER/MEMBER/COLLABORATOR comments are honored.

## Releasing

Releases are **dispatch-only** — merging a PR never ships anything.

- **Cut a release** from the Actions tab ("Release-Go-iOS" → Run workflow) or:
  ```
  gh workflow run release.yml -f bump=patch -f release_notes="<markdown notes>"
  ```
  `bump` is `patch` | `minor` | `major`.
- **Write the notes yourself:** summarize everything merged since the last
  release (review `git log <latest-tag>..main` and the merged PRs) into markdown
  and pass it as `release_notes`. Group user-facing changes and fixes; reference
  PR numbers.
- The workflow then computes the next version from the latest tag, writes a
  `## [version] - date` section into `CHANGELOG.md`, pushes it to `main`, tags
  `vX.Y.Z`, creates the GitHub release with your notes, npm-publishes via OIDC,
  and verifies the published package installs and runs on Windows/Linux/macOS.
  All mutating/outbound steps are in the final job, so a failed build ships
  nothing.
- The **`RELEASE_PAT`** secret (admin fine-grained PAT, Contents: read/write) is
  required — it's how the workflow pushes the changelog commit to protected
  `main`. The default `GITHUB_TOKEN` cannot.
- **Test any change to the release/publish/packaging path on the canary first:**
  ```
  gh workflow run release-canary.yml --ref <your-branch>
  ```
  It publishes a throwaway `go-ios-canary` package and runs the same cross-OS
  install verification. Never debug the release pipeline with real releases.

## Rules

- Publishing uses npm OIDC trusted publishing (registered per-package on
  npmjs.com). Do not add an npm auth token — even an empty `NODE_AUTH_TOKEN`
  env var or an `.npmrc` token line breaks OIDC.
- The release workflow owns version numbers and the `CHANGELOG.md` history;
  don't hand-write version sections.
- In `npm_publish/postinstall.js`, the Windows install path must place the
  binary directly in the npm prefix (on Windows the prefix itself is the bin
  dir, not a `bin/` subdir), or `ios` won't be on `PATH`.
- `main` is protected (1 approving review). Branch for changes and open a PR; the
  release workflow is the only thing that writes to `main` directly.
