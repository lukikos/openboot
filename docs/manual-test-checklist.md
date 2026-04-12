# Manual Test Checklist

Tests that require a real system, real auth, interactive TUI, or a browser.
Unit/integration tests cover error paths and logic; this covers the full UX.

Binary: `/Users/fullstackjam/workspace/openbotdotdev/openboot/openboot`  
Rebuild: `make build` from the repo root.

---

## Prerequisites

- [ ] Logged in: `./openboot login`
- [ ] At least one config on openboot.dev
- [ ] Sync source configured (run `./openboot install <config>` or `./openboot push` once)

---

## openboot push

### Happy path — with sync source (silent update)
- [ ] Run `./openboot push`
- [ ] No prompts appear (name/desc/visibility not asked)
- [ ] Output shows `✓ Config uploaded successfully!` with the config URL
- [ ] Run `./openboot log` — new revision appears at the top

### Happy path — with message
- [ ] Run `./openboot push -m "manual test"`
- [ ] Run `./openboot log` — newest revision shows `manual test` as message

### Interactive picker — no sync source
- [ ] Delete `~/.openboot/sync_source.json`
- [ ] Run `./openboot push`
- [ ] Picker appears listing existing configs + "Create a new config"
- [ ] Select an existing config → uploads silently, no name/desc prompts
- [ ] Repeat, choose "Create a new config" → name / description / visibility prompts appear

### --slug flag skips picker
- [ ] Run `./openboot push --slug <slug>`
- [ ] No picker, no prompts — silent update

---

## openboot pull

### Happy path
- [ ] Run `./openboot pull`
- [ ] Diff is shown, changes applied (or "already up to date")

### Dry run
- [ ] Run `./openboot pull --dry-run`
- [ ] Shows diff, nothing installed/removed

---

## openboot list

### Shows configs
- [ ] Run `./openboot list`
- [ ] All configs appear with slug and name
- [ ] Currently linked config is marked with `→`
- [ ] Public configs show `[public]`, private show `[private]`, unlisted show nothing
- [ ] Footer shows install / edit / delete hints

### Linked config warning
- [ ] Edit `~/.openboot/sync_source.json`, change slug to `nonexistent-slug`
- [ ] Run `./openboot list`
- [ ] Warning appears: "linked to 'nonexistent-slug' but that config no longer exists"
- [ ] Restore the original slug in sync_source.json

---

## openboot edit

### Interactive picker
- [ ] Run `./openboot edit`
- [ ] Picker lists all your configs
- [ ] Select one → browser opens `https://openboot.dev/dashboard/edit/<slug>`
- [ ] Edit page loads with the correct config

### --slug skips picker
- [ ] Run `./openboot edit --slug <slug>`
- [ ] No picker — browser opens directly

---

## openboot log

### Shows revision history
- [ ] Run `./openboot push -m "rev A"`, then `./openboot push -m "rev B"`
- [ ] Run `./openboot log`
- [ ] Both revisions appear, newest first
- [ ] IDs, timestamps, package counts, and messages all shown
- [ ] Footer shows `openboot restore <revision-id>` hint

### Empty history
- [ ] Create a brand new config with `./openboot push` (choose "Create a new config")
- [ ] Run `./openboot log --slug <new-slug>`
- [ ] Output: "No revisions yet. Push a config update to create one."

### --slug flag
- [ ] Run `./openboot log --slug <slug>`
- [ ] Shows history for that specific config, not the linked one

---

## openboot restore

### Dry run — preview only
- [ ] Run `./openboot log` to get a revision ID
- [ ] Run `./openboot restore <rev-id> --dry-run`
- [ ] Diff is shown (packages that would change)
- [ ] Nothing is installed or removed
- [ ] Restore endpoint is NOT called (run `./openboot log` — no new "before restore" revision)

### Actual restore
- [ ] Note current package count from `./openboot log`
- [ ] Push a config with a different package list (e.g. add a formula)
- [ ] Run `./openboot log` — confirm new revision at top
- [ ] Run `./openboot restore <previous-rev-id> --yes`
- [ ] Diff shown, changes applied
- [ ] Run `./openboot log` — new "before restore to …" revision saved at top
- [ ] Confirm local system matches the restored revision

### Already up to date
- [ ] Run `./openboot restore <rev-id>` where the revision matches your current system
- [ ] Output: "Your system already matches this revision — nothing to do."

### Revision not found
- [ ] Run `./openboot restore rev_doesnotexist --slug <slug>`
- [ ] Error: "revision not found" (or similar)

### --slug flag
- [ ] Run `./openboot restore <rev-id> --slug <other-slug>`
- [ ] Restores the correct config, not the linked one

---

## openboot sync

### Shows diff and applies selectively
- [ ] Make a change to the remote config on openboot.dev (add a package)
- [ ] Run `./openboot sync`
- [ ] Diff shown, interactive checkboxes to select which changes to apply
- [ ] Apply selection — package installed

### --yes applies all
- [ ] Run `./openboot sync --yes`
- [ ] No prompts, all changes applied

---

## openboot diff

- [ ] Run `./openboot diff`
- [ ] Shows packages missing locally vs extra locally compared to remote config
- [ ] No changes applied

---

## openboot delete

### With confirmation
- [ ] Create a throwaway config: `./openboot push --slug test-delete-me`
- [ ] Run `./openboot delete test-delete-me`
- [ ] Confirmation prompt appears — type `n` to cancel
- [ ] Run `./openboot list` — config still exists
- [ ] Run `./openboot delete test-delete-me` again, confirm `y`
- [ ] Config removed from `./openboot list`

### --force skips prompt
- [ ] Create another throwaway config
- [ ] Run `./openboot delete <slug> --force`
- [ ] No prompt — deleted immediately

---

## Revision cap (10 max)

- [ ] Run `./openboot push` 11 times in a row
- [ ] Run `./openboot log`
- [ ] Only 10 revisions shown (oldest pruned automatically)
