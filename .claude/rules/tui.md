---
paths:
  - "internal/ui/**/*.go"
---

# TUI Conventions

- **Pattern**: bubbletea `Model` (`Init`/`Update`/`View`) + lipgloss styling + huh for prompts.
- **All styled output must flow through `ui` package** — never `fmt.Println` directly in user-facing paths. Use `ui.Header`, `ui.Success`, `ui.Error`, `ui.Info`, `ui.Warn`, `ui.Muted`.
- **Color palette** (from `internal/ui`):
  - Primary: `#22c55e`
  - Secondary: `#60a5fa`
  - Warning: `#eab308`
  - Danger: `#ef4444`
  - Subtle: `#666666`
- **Progress output**: use `StickyProgress` (see `internal/brew`) for long-running multi-worker operations.
- Adding a TUI component: new file under `internal/ui/`, export a `Model` or a helper function returning styled `string`.
