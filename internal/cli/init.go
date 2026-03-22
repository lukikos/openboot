package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openbootdotdev/openboot/internal/config"
	"github.com/openbootdotdev/openboot/internal/initializer"
	"github.com/openbootdotdev/openboot/internal/updater"
	"github.com/spf13/cobra"
)

var (
	initCheck bool
	initAuto  bool
	initJSON  bool
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Detect and install project dependencies",
	Long: `Scan project files to detect required dependencies and install missing ones.

Works without any config file — auto-detects from package.json, go.mod,
Cargo.toml, docker-compose.yml, and 15+ other project file types.

If .openboot.yml exists, uses that instead (explicit config takes priority).

Designed for AI coding agents: run "openboot init --auto" before starting
work on any project to ensure all dependencies are satisfied.`,
	Example: `  # Auto-detect and interactively select (human mode)
  openboot init

  # Auto-install all missing dependencies (agent/CI mode)
  openboot init --auto

  # Check what's missing without installing (returns exit code 1 if missing)
  openboot init --check

  # JSON output for AI agents
  openboot init --check --json

  # Use explicit .openboot.yml config
  openboot init /path/to/project

  # Preview changes without installing
  openboot init --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		updater.AutoUpgrade(version)

		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("resolve directory: %w", err)
		}

		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", absDir)
		}

		// If --check, --auto, or --json is used, go straight to auto-detect
		if initCheck || initAuto || initJSON {
			return runAutoDetect(absDir)
		}

		// Try .openboot.yml first
		projectCfg, err := config.LoadProjectConfig(absDir)
		if err != nil {
			// .openboot.yml not found — fall back to auto-detection
			if errors.Is(err, config.ErrConfigNotFound) {
				return runAutoDetect(absDir)
			}
			return err
		}

		initCfg := &initializer.Config{
			ProjectDir:    absDir,
			ProjectConfig: projectCfg,
			DryRun:        cfg.DryRun,
			Silent:        cfg.Silent,
			Update:        cfg.Update,
			Version:       version,
		}

		return initializer.Run(initCfg)
	},
}

func init() {
	initCmd.Flags().SortFlags = false
	initCmd.Flags().BoolVar(&initCheck, "check", false, "check dependencies without installing (exit 1 if missing)")
	initCmd.Flags().BoolVar(&initAuto, "auto", false, "auto-install all detected missing dependencies")
	initCmd.Flags().BoolVar(&initJSON, "json", false, "output results as JSON (for AI agents and scripts)")
	initCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "preview changes without installing")
	initCmd.Flags().BoolVarP(&cfg.Silent, "silent", "s", false, "non-interactive mode (for CI/CD)")
	initCmd.Flags().BoolVar(&cfg.Update, "update", false, "update Homebrew before installing")
}
