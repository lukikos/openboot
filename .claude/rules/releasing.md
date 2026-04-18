---
paths:
  - "CHANGELOG.md"
  - ".github/workflows/release.yml"
  - "internal/cli/root.go"
---

# Release Process

Tag-driven. CI handles everything. **Never edit `root.go` for version bumps** — version is `"dev"` in source, overridden via `-ldflags -X github.com/openbootdotdev/openboot/internal/cli.version=<tag>` at build time.

```bash
git tag v0.25.0
git push --tags
# .github/workflows/release.yml builds, uploads, creates GitHub release
```

- Dev builds (`version=dev`) skip auto-update.
- **Release only for user-facing changes** (features, bug fixes, package updates). Skip for docs, CI config, test-only.

## Writing Release Notes

CI creates a release with a generic install-only body. After CI completes, overwrite it via `gh release edit`.

### Step 1 — gather commits

```bash
PREV_TAG=$(git tag --sort=-v:refname | sed -n '2p')
git log ${PREV_TAG}..HEAD --oneline
```

### Step 2 — draft

```markdown
## What's New
- **Feature name** — One sentence, user-facing benefit only (`openboot <command>`)

## Improvements
- **Area** — What changed and why users care

## Bug Fixes
- **What was broken** — What's fixed now

## Installation

\`\`\`bash
brew install openbootdotdev/tap/openboot
\`\`\`

## Binaries

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (M1/M2/M3/M4) | `openboot-darwin-arm64` |
| macOS | Intel | `openboot-darwin-amd64` |
```

### Rules

- Omit empty sections (no "Bug Fixes" if there are none).
- Write for **users**, not developers. No internal refactors, no test-only changes.
- **Bold name**: 2–4 words, noun form — not a sentence.
- **Description**: ONE sentence, ~10–15 words. User benefit only, no implementation details.
- Include the CLI command at the end if it's a new/changed command.
- Installation and Binaries sections always last.

### Do / Don't

```
✓ - **Post-install script** — Run custom shell commands after your environment is set up (`openboot -u <user>`)
✗ - **Post-install script** — Run custom shell commands after your environment is set up. Add a post_install array to your config on openboot.dev and each command runs sequentially in your home directory after packages, shell, dotfiles, and macOS preferences are applied.

✓ - **Custom config install** — Shell, dotfiles, and macOS setup now run correctly when installing from a remote config
✗ - **Custom config installs now run shell, dotfiles, and macOS setup** — When installing via openboot -u <user>, shell configuration (Oh-My-Zsh), dotfiles cloning, and macOS preferences were silently skipped. All three steps now run as expected.
```

### Step 3 — publish

Use a `'EOF'` heredoc so the shell doesn't interpret backticks:

```bash
gh release edit v0.25.0 --repo openbootdotdev/openboot --notes "$(cat <<'EOF'
## What's New
- **Feature name** — One sentence description (`openboot <command>`)

## Installation

\`\`\`bash
brew install openbootdotdev/tap/openboot
\`\`\`

## Binaries

| Platform | Architecture | Download |
|----------|--------------|----------|
| macOS | Apple Silicon (M1/M2/M3/M4) | `openboot-darwin-arm64` |
| macOS | Intel | `openboot-darwin-amd64` |
EOF
)"
```

## CLI Breaking Changes

Breaking changes are allowed with a major version bump + migration entry in `CHANGELOG.md`. Same-name commands must not silently change behavior — preserve semantics or remove the command. Non-breaking changes (new flags, new commands) do not require version bumps.
