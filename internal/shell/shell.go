package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/openbootdotdev/openboot/internal/system"
)

var shellIdentifierRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

func validateShellIdentifier(value, label string) error {
	if value == "" {
		return nil
	}
	if !shellIdentifierRe.MatchString(value) {
		return fmt.Errorf("invalid %s: %q (only alphanumerics, hyphens, underscores, dots allowed)", label, value)
	}
	return nil
}

func IsOhMyZshInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".oh-my-zsh"))
	return err == nil
}

func InstallOhMyZsh(dryRun bool) error {
	if IsOhMyZshInstalled() {
		return nil
	}

	if dryRun {
		fmt.Println("[DRY-RUN] Would install Oh-My-Zsh")
		return nil
	}

	script := `sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended`
	cmd := exec.Command("bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	home, err := system.HomeDir()
	if err != nil {
		return err
	}
	zshrcPath := filepath.Join(home, ".zshrc")
	if _, err := os.Stat(zshrcPath); err == nil {
		if err := os.Rename(zshrcPath, zshrcPath+".openboot.bak"); err != nil {
			return fmt.Errorf("backup %s: %w", zshrcPath, err)
		}
	}

	return nil
}

func SetDefaultShell(dryRun bool) error {
	zshPath := "/bin/zsh"
	if _, err := os.Stat(zshPath); os.IsNotExist(err) {
		zshPath = "/usr/bin/zsh"
	}

	if dryRun {
		fmt.Printf("[DRY-RUN] Would set default shell to %s\n", zshPath)
		return nil
	}

	cmd := exec.Command("chsh", "-s", zshPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

const restoreBlockStart = "# >>> OpenBoot-Restore"
const restoreBlockEnd = "# <<< OpenBoot-Restore"

var restoreBlockRe = regexp.MustCompile(`(?s)# >>> OpenBoot-Restore\n.*?# <<< OpenBoot-Restore\n?`)

func buildRestoreBlock(theme string, plugins []string) (string, error) {
	if err := validateShellIdentifier(theme, "ZSH_THEME"); err != nil {
		return "", err
	}
	for _, p := range plugins {
		if err := validateShellIdentifier(p, "plugin"); err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	sb.WriteString(restoreBlockStart + "\n")
	if theme != "" {
		sb.WriteString(fmt.Sprintf("ZSH_THEME=\"%s\"\n", theme))
	}
	if len(plugins) > 0 {
		sb.WriteString(fmt.Sprintf("plugins=(%s)\n", strings.Join(plugins, " ")))
	}
	sb.WriteString(restoreBlockEnd + "\n")
	return sb.String(), nil
}

var (
	looseThemeRe   = regexp.MustCompile(`(?m)^ZSH_THEME="[^"]*"\n?`)
	loosePluginsRe = regexp.MustCompile(`(?m)^plugins=\((?s:.*?)\)\n?`)
)

func patchZshrcBlock(zshrcPath, theme string, plugins []string) error {
	raw, err := os.ReadFile(zshrcPath)
	if err != nil {
		return fmt.Errorf("read .zshrc: %w", err)
	}

	block, err := buildRestoreBlock(theme, plugins)
	if err != nil {
		return fmt.Errorf("build restore block: %w", err)
	}
	content := string(raw)

	if restoreBlockRe.MatchString(content) {
		content = restoreBlockRe.ReplaceAllString(content, block)
	} else {
		if theme != "" {
			content = looseThemeRe.ReplaceAllString(content, "")
		}
		if len(plugins) > 0 {
			content = loosePluginsRe.ReplaceAllString(content, "")
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			content = content + "\n"
		}
		content = content + block
	}

	tmpPath := zshrcPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write .zshrc: %w", err)
	}
	if err := os.Rename(tmpPath, zshrcPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename .zshrc: %w", err)
	}
	return nil
}

func RestoreFromSnapshot(ohMyZsh bool, theme string, plugins []string, dryRun bool) error {
	if !ohMyZsh {
		return nil
	}

	if !IsOhMyZshInstalled() {
		if dryRun {
			fmt.Println("[DRY-RUN] Would install Oh-My-Zsh")
		} else {
			if err := InstallOhMyZsh(dryRun); err != nil {
				return fmt.Errorf("install oh-my-zsh: %w", err)
			}
		}
	}

	home, err := system.HomeDir()
	if err != nil {
		return err
	}
	zshrcPath := filepath.Join(home, ".zshrc")

	if _, err := os.Stat(zshrcPath); os.IsNotExist(err) {
		if dryRun {
			fmt.Printf("[DRY-RUN] Would create %s\n", zshrcPath)
			return nil
		}
		if err := validateShellIdentifier(theme, "ZSH_THEME"); err != nil {
			return err
		}
		for _, p := range plugins {
			if err := validateShellIdentifier(p, "plugin"); err != nil {
				return err
			}
		}
		template := fmt.Sprintf(`export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="%s"
plugins=(%s)
source $ZSH/oh-my-zsh.sh
`, theme, strings.Join(plugins, " "))
		if err := os.WriteFile(zshrcPath, []byte(template), 0644); err != nil {
			return fmt.Errorf("create .zshrc: %w", err)
		}
		return nil
	}

	if dryRun {
		if theme != "" {
			fmt.Printf("[DRY-RUN] Would set ZSH_THEME=\"%s\"\n", theme)
		}
		if len(plugins) > 0 {
			fmt.Printf("[DRY-RUN] Would set plugins=(%s)\n", strings.Join(plugins, " "))
		}
		return nil
	}

	if theme == "" && len(plugins) == 0 {
		return nil
	}

	return patchZshrcBlock(zshrcPath, theme, plugins)
}
