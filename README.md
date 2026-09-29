# plugin-distro

The `distro` build-vocabulary kind for OpenCharly — the per-distro bootstrap
command, package-format templates, and base-user declaration, relocated into a
plugin (`kind:distro`).

A `kind:distro` entity is the build vocabulary that drives image and VM builds:
the bootstrap command, per-format package templates (how `rpm:` / `pac:` / `deb:`
sections become build steps), cache mounts, and the `base_user:` declaration
(what uid-1000 account the upstream base image ships). The kind dispatches via
the `pb` `Invoke(OpLoad)` envelope: it decodes the authored `distro:` entity into
its core spec type and re-marshals it as canonical JSON. It serves itself in BOTH
placements (compiled-in or out-of-process) — no kit contract is needed because
kinds are `pb`-shape.

## What it provides

| Capability | Surface |
|---|---|
| `kind:distro` | the `distro:` build-vocabulary entity |

## The entity

The authored body is validated at runtime against the self-contained
`#DistroInput` (`schema/distro.cue`), spliced by the host. Its fields include:

- `bootstrap` — the install command + package list + cache mounts.
- `format` — per-package-format templates (`section_field`, `uninstall_template`,
  `phase`, `validate`, `local_pkg`).
- `base_user` — the pre-existing uid-1000 account of the upstream base image.
- `pacstrap` / `debootstrap` / `alpine_bootstrap` — bootstrap-from-scratch
  vocabulary.
- `bootloader` / `disk_layout` / `installer` — the on-disk / unattended-install
  shape for bootstrap VMs.
- `inherits` / `inherit_packages` / `version` / `workaround` / `dnf`.

## How to use it

Compose the plugin candy in a project's `candy:` list:

```yaml
- '@github.com/opencharly/plugin-distro/candy/plugin-distro:<tag>'
```

Then author a `distro:` entity:

```yaml
debian:
    distro:
        bootstrap:
            install_cmd: apt-get update && apt-get install -y
        base_user:
            name: ubuntu
            uid: 1000
            gid: 1000
            home: /home/ubuntu
```

## Layout

- `candy/plugin-distro/` — the plugin module: `plugin.go` (the kind provider +
  `NewProvider()` / `NewMeta()`), `resolve.go` (the `OpResolve` de-type),
  `schema/distro.cue` (the self-contained `#DistroInput` and its `#Ds*` defs),
  `params/cue_types_gen.go`, `resolve_test.go`, `resolve_parity_test.go`,
  `schema_parity_test.go`, `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-build:build` — the build vocabulary (`distro:` /
  `builder:` / `init:`), the `base_user:` declaration, and the per-distro
  bootstrap/format templates. This candy carries no `skill:` entity of its own;
  the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-image:layer` — the candy authoring surface and the package surface the
  `distro:` map drives.
- `/charly-internals:plugin` — the plugin/provider model, including the `kind`
  provider class.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
