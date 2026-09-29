# AGENTS.md — plugin-distro

Standalone plugin repo serving the `distro` build-vocabulary kind
(`kind:distro`). The plugin is a Go module at `candy/plugin-distro/` (module path
`github.com/opencharly/plugin-distro/candy/plugin-distro`); the root `charly.yml`
only declares `discover: candy` so the repo is a project and its candy is
scanned.

Canonical files:

- `candy/plugin-distro/charly.yml` — the `plugin-distro:` candy entity
  (`plugin:` block, `plan:` check).
- `candy/plugin-distro/plugin.go` — the kind provider (`Invoke(OpLoad)` decode →
  canonical JSON; `Invoke(OpResolve)` de-type) and `NewProvider()`/`NewMeta()`.
- `candy/plugin-distro/resolve.go` — the `OpResolve` projection into a
  `ResolvedDistro` the build engine consumes.
- `candy/plugin-distro/schema/distro.cue` — the self-contained `#DistroInput`
  and its `#Ds*` defs (single source for `params/cue_types_gen.go`).
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-build:build` — the build vocabulary (`distro:` / `builder:` / `init:`),
  the `base_user:` declaration, and the per-distro bootstrap/format templates.
  Load before changing the kind's decoded fields. This candy carries no `skill:`
  entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the `kind` provider class, the per-plugin CUE-schema contract. Load
  before touching the provider or schema.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-distro/` — compile the plugin module.
- `go test ./...` in `candy/plugin-distro/` — the plugin's Go tests (the resolve
  parity + schema parity seams).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- The changed path is exercised by any box/deploy composing a `distro:` node.

## Modify this repo

- Edit the `plugin-distro:` candy entity, the Go source, and
  `schema/distro.cue` **together** — the schema is the single source for the
  kind's `params/` struct and a faithful reproduction of the core `#Distro` wire
  keys. A wire-key change must keep the host's canonicalisation through
  `spec.Distro` valid.
- The entity name is the node KEY, never a body field.
- `schema_parity_test.go` guards the served `#DistroInput` against drift from
  spec's authored surface — keep it passing when spec changes.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
