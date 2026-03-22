package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openbootdotdev/openboot/internal/ui"
	"github.com/spf13/cobra"
)

// ruleContent is the instruction written to agent global rules.
const ruleContent = `# OpenBoot — Project Dependency Detection

When a command fails due to missing tools, runtimes, or services
(e.g. "command not found", "module not found", "connection refused"),
use openboot to detect and install all missing dependencies at once
instead of installing them one by one:

` + "```" + `bash
# See everything that's missing in one shot
openboot init --check --json

# Install all missing dependencies
openboot init --auto
` + "```" + `

openboot auto-detects dependencies from project files: package.json,
go.mod, Cargo.toml, pyproject.toml, docker-compose.yml, and 15+ other
file types. It installs via Homebrew.

## When to use
- A build, test, or run command fails due to missing dependencies
- The user asks you to set up or bootstrap the project environment
- You need multiple tools installed and want to avoid trial-and-error

## When NOT to use
- Read-only tasks (code review, explaining code, searching)
- The user only wants to look at the code, not run it
- Dependencies are already satisfied (openboot init --check exits 0)
`

// agentTarget defines where to write the rule for each AI coding tool.
type agentTarget struct {
	name     string
	detect   func(home string) bool
	rulePath func(home string) string
}

func agentTargets() []agentTarget {
	return []agentTarget{
		{
			name: "Claude Code",
			detect: func(home string) bool {
				return dirExists(filepath.Join(home, ".claude", "rules"))
			},
			rulePath: func(home string) string {
				return filepath.Join(home, ".claude", "rules", "openboot.md")
			},
		},
		{
			name: "OpenAI Codex",
			detect: func(home string) bool {
				return dirExists(filepath.Join(home, ".codex"))
			},
			rulePath: func(home string) string {
				return filepath.Join(home, ".codex", "instructions.md")
			},
		},
	}
}

var setupAgentCmd = &cobra.Command{
	Use:   "setup-agent",
	Short: "Enable AI agent integration (Claude Code, Codex)",
	Long: `Write global rules so AI coding agents know to use openboot when
they encounter missing dependencies.

Supports: Claude Code, OpenAI Codex.

One-time setup — applies to ALL projects automatically.`,
	Example: `  openboot setup-agent`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		ui.Header("OpenBoot Agent Setup")
		fmt.Println()

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}

		targets := agentTargets()
		detected := 0
		configured := 0
		skipped := 0

		for _, t := range targets {
			if !t.detect(home) {
				continue
			}
			detected++

			rulePath := t.rulePath(home)

			if fileExists(rulePath) {
				ui.Muted(fmt.Sprintf("  %s — already configured", t.name))
				skipped++
				continue
			}

			// Ensure parent directory exists
			ruleDir := filepath.Dir(rulePath)
			if !dirExists(ruleDir) {
				if err := os.MkdirAll(ruleDir, 0755); err != nil {
					ui.Warn(fmt.Sprintf("  %s — failed to create directory: %v", t.name, err))
					continue
				}
			}

			if err := os.WriteFile(rulePath, []byte(ruleContent), 0644); err != nil {
				ui.Warn(fmt.Sprintf("  %s — failed to write rule: %v", t.name, err))
				continue
			}

			ui.Success(fmt.Sprintf("%s — configured (%s)", t.name, rulePath))
			configured++
		}

		fmt.Println()

		if detected == 0 {
			ui.Warn("No supported AI coding tools detected.")
			ui.Muted("  Supported: Claude Code, OpenAI Codex")
			ui.Muted("  Install one of these tools first, then run 'openboot setup-agent' again.")
		} else if configured > 0 {
			ui.Success(fmt.Sprintf("Done! Configured %d agent(s). They will now use openboot when dependencies are missing.", configured))
		} else {
			ui.Info(fmt.Sprintf("All %d detected agent(s) already configured.", skipped))
		}

		fmt.Println()
		return nil
	},
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
