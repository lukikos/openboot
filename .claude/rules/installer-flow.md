---
paths:
  - "internal/installer/**"
  - "internal/cli/install.go"
  - "internal/sync/**"
  - "internal/cleaner/**"
---

# Installer Flow

`openboot install` is orchestrated in `internal/installer/installer.go` as a 7-step wizard:

1. Check deps
2. Homebrew setup
3. git config
4. Preset selection
5. Package install (brew + npm, parallel — 4 workers max for brew)
6. Shell setup (Oh-My-Zsh + `.zshrc`)
7. macOS prefs + dotfiles

## Source resolution

`internal/cli/install.go` → `resolvePositionalArg` smart-detects file / user-slug / preset / alias.

- Positional arg → local file OR remote user config (`openboot install <slug>`)
- No args + authed → `runSyncInstall` (diff remote config vs system, confirm, execute)
- Preset → embedded `internal/config/data/presets.yaml` (`minimal` / `developer` / `full`)

## Sync flow

`internal/sync/diff.go` computes `Diff`; `runSyncInstall` confirms then executes via the same brew/npm/shell/macos/dotfiles modules.

## Clean / uninstall

`internal/cleaner/cleaner.go` diffs current vs desired, calls `brew.Uninstall` / `npm.Uninstall`.

## Snapshot restore

`installer.go` → `stepRestoreGit`, `stepRestoreShell` + packages/shell/macos steps. Snapshot data captured in `internal/snapshot/capture.go` via `CaptureWithProgress`.

## Dry-run

**Every destructive op must check `cfg.DryRun` first.** Env var: `OPENBOOT_DRY_RUN=1`.
