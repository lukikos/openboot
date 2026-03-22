package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/openbootdotdev/openboot/internal/brew"
	"github.com/openbootdotdev/openboot/internal/detector"
	"github.com/openbootdotdev/openboot/internal/ui"
)

// jsonOutput is the JSON response format for --json mode.
type jsonOutput struct {
	Satisfied      bool                 `json:"satisfied"`
	Dir            string               `json:"dir"`
	Detected       []detector.Detection `json:"detected"`
	Missing        []string             `json:"missing"`
	InstalledNow   []string             `json:"installed_now,omitempty"`
	InstallCommand string               `json:"install_command,omitempty"`
	Error          string               `json:"error,omitempty"`
}

func runAutoDetect(dir string) error {
	// 1. Scan project files
	if !initJSON {
		ui.Header("OpenBoot Auto-Detect")
		fmt.Println()
	}

	result, err := detector.Scan(dir)
	if err != nil {
		return handleError(err, dir)
	}

	if len(result.Detected) == 0 {
		if initJSON {
			return printJSON(jsonOutput{Satisfied: true, Dir: dir, Detected: []detector.Detection{}, Missing: []string{}})
		}
		ui.Muted("No project files detected in " + dir)
		return nil
	}

	// 2. Check installed state
	if !initJSON {
		ui.Info("Checking installed packages...")
	}
	formulae, casks, brewErr := brew.GetInstalledPackages()
	if brewErr != nil {
		if !initJSON {
			ui.Warn(fmt.Sprintf("Failed to query Homebrew: %v", brewErr))
		}
		formulae = make(map[string]bool)
		casks = make(map[string]bool)
	}
	result = detector.Enrich(result, formulae, casks)

	// === Route to mode ===

	// --check: report only, exit 1 if missing
	if initCheck {
		return handleCheck(result)
	}

	// --auto or --silent: install without asking
	if initAuto || cfg.Silent {
		return handleAuto(result)
	}

	// --json without --check or --auto: just output current state
	if initJSON {
		return printJSON(resultToJSON(result))
	}

	// Default: interactive TUI
	return handleInteractive(result)
}

func handleCheck(result detector.ScanResult) error {
	if initJSON {
		out := resultToJSON(result)
		if !result.Satisfied {
			out.InstallCommand = "openboot init --auto"
		}
		if err := printJSON(out); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
		}
		if !result.Satisfied {
			os.Exit(1)
		}
		return nil
	}

	printCheckSummary(result)
	if !result.Satisfied {
		os.Exit(1)
	}
	return nil
}

func handleAuto(result detector.ScanResult) error {
	if result.Satisfied {
		if initJSON {
			return printJSON(resultToJSON(result))
		}
		ui.Success("All dependencies satisfied")
		return nil
	}

	toInstall := result.NonOptionalMissing()
	if len(toInstall) == 0 {
		if initJSON {
			return printJSON(resultToJSON(result))
		}
		ui.Success("All required dependencies satisfied (optional dependencies skipped)")
		return nil
	}

	if !initJSON {
		fmt.Println()
		ui.Info(fmt.Sprintf("Installing %d missing dependencies...", len(toInstall)))
		fmt.Println()
	}

	var formulae, casks []string
	for _, d := range toInstall {
		if d.IsCask {
			casks = append(casks, d.Package)
		} else {
			formulae = append(formulae, d.Package)
		}
	}

	start := time.Now()

	if cfg.DryRun {
		if initJSON {
			out := resultToJSON(result)
			out.InstallCommand = "openboot init --auto (dry-run)"
			return printJSON(out)
		}
		if _, _, err := brew.InstallWithProgress(formulae, casks, true); err != nil {
			return fmt.Errorf("dry-run: %w", err)
		}
		return nil
	}

	installedFormulae, installedCasks, err := brew.InstallWithProgress(formulae, casks, false)
	if err != nil {
		return handleError(err, result.Dir)
	}

	elapsed := time.Since(start).Round(time.Second)
	var installedNow []string
	installedNow = append(installedNow, installedFormulae...)
	installedNow = append(installedNow, installedCasks...)

	if initJSON {
		// Re-scan to get updated state
		updated, scanErr := detector.Scan(result.Dir)
		if scanErr != nil {
			return handleError(scanErr, result.Dir)
		}
		newFormulae, newCasks, _ := brew.GetInstalledPackages()
		if newFormulae == nil {
			newFormulae = make(map[string]bool)
		}
		if newCasks == nil {
			newCasks = make(map[string]bool)
		}
		updated = detector.Enrich(updated, newFormulae, newCasks)
		out := resultToJSON(updated)
		out.InstalledNow = installedNow
		return printJSON(out)
	}

	fmt.Println()
	ui.Success(fmt.Sprintf("All dependencies satisfied (%d installed in %s)", len(installedNow), elapsed))
	fmt.Println()
	return nil
}

func handleInteractive(result detector.ScanResult) error {
	if result.Satisfied {
		showInstalledSummary(result)
		return nil
	}

	if cfg.DryRun {
		printCheckSummary(result)
		return nil
	}

	// TUI selection
	selected, err := ui.RunDetectionSelector(result.Detected)
	if err != nil {
		return fmt.Errorf("selection: %w", err)
	}
	if len(selected) == 0 {
		return nil
	}

	// Install selected
	var formulae, casks []string
	for _, d := range selected {
		if d.IsCask {
			casks = append(casks, d.Package)
		} else {
			formulae = append(formulae, d.Package)
		}
	}

	fmt.Println()
	start := time.Now()
	_, _, err = brew.InstallWithProgress(formulae, casks, false)
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}

	elapsed := time.Since(start).Round(time.Second)
	fmt.Println()
	ui.Success(fmt.Sprintf("Done in %s", elapsed))

	// Offer to save as .openboot.yml
	fmt.Println()
	save, confirmErr := ui.Confirm("Save as .openboot.yml for your team?", false)
	if confirmErr != nil {
		ui.Warn(fmt.Sprintf("Failed to prompt: %v", confirmErr))
	} else if save {
		if err := detector.SaveProjectConfig(result.Dir, selected); err != nil {
			ui.Warn(fmt.Sprintf("Failed to save: %v", err))
		} else {
			ui.Success("Saved .openboot.yml")
		}
	}

	fmt.Println()
	ui.Success("Environment ready!")
	fmt.Println()
	return nil
}

// --- Output helpers ---

func printCheckSummary(result detector.ScanResult) {
	fmt.Println()
	for _, d := range result.Detected {
		if d.Installed {
			version := ""
			if d.Version != "" {
				version = "@" + d.Version
			}
			ui.Success(fmt.Sprintf("%-18s %s", d.Package+version, d.Source))
		}
	}
	for _, d := range result.Detected {
		if !d.Installed {
			version := ""
			if d.Version != "" {
				version = "@" + d.Version
			}
			label := "missing"
			if d.Confidence == detector.ConfidenceOptional {
				label = "optional"
			}
			ui.Warn(fmt.Sprintf("%-18s %s (%s)", d.Package+version, d.Source, label))
		}
	}

	fmt.Println()
	if result.Satisfied {
		ui.Success("All dependencies satisfied")
	} else {
		ui.Info(fmt.Sprintf("%d missing — run 'openboot init --auto' to install", len(result.Missing)))
	}
	fmt.Println()
}

func showInstalledSummary(result detector.ScanResult) {
	fmt.Println()
	for _, d := range result.Detected {
		version := ""
		if d.Version != "" {
			version = "@" + d.Version
		}
		ui.Success(fmt.Sprintf("%-18s %s", d.Package+version, d.Source))
	}
	fmt.Println()
	ui.Success("All dependencies satisfied")
	fmt.Println()
}

func resultToJSON(result detector.ScanResult) jsonOutput {
	detected := result.Detected
	if detected == nil {
		detected = []detector.Detection{}
	}
	missing := result.Missing
	if missing == nil {
		missing = []string{}
	}
	return jsonOutput{
		Satisfied: result.Satisfied,
		Dir:       result.Dir,
		Detected:  detected,
		Missing:   missing,
	}
}

func printJSON(out jsonOutput) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func handleError(err error, dir string) error {
	if initJSON {
		if jsonErr := printJSON(jsonOutput{
			Satisfied: false,
			Dir:       dir,
			Error:     err.Error(),
		}); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", jsonErr)
		}
		os.Exit(2)
	}
	return err
}
