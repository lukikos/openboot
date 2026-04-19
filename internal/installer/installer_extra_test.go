package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openbootdotdev/openboot/internal/config"
)

// ---------------------------------------------------------------------------
// printPackageList
// ---------------------------------------------------------------------------

func TestPrintPackageList_EmptyList(t *testing.T) {
	// Empty list must be a silent no-op — no panic.
	assert.NotPanics(t, func() {
		printPackageList("CLI tools", config.PackageEntryList{})
	})
}

func TestPrintPackageList_NilList(t *testing.T) {
	assert.NotPanics(t, func() {
		printPackageList("CLI tools", nil)
	})
}

func TestPrintPackageList_SingleItemNoDesc(t *testing.T) {
	pkgs := config.PackageEntryList{{Name: "git"}}
	assert.NotPanics(t, func() {
		printPackageList("CLI tools", pkgs)
	})
}

func TestPrintPackageList_MultipleItemsNoDesc(t *testing.T) {
	pkgs := config.PackageEntryList{
		{Name: "git"},
		{Name: "curl"},
		{Name: "ripgrep"},
	}
	assert.NotPanics(t, func() {
		printPackageList("CLI tools", pkgs)
	})
}

func TestPrintPackageList_WithDescriptions(t *testing.T) {
	pkgs := config.PackageEntryList{
		{Name: "git", Desc: "Distributed version control"},
		{Name: "curl", Desc: "Transfer data with URLs"},
	}
	assert.NotPanics(t, func() {
		printPackageList("CLI tools", pkgs)
	})
}

func TestPrintPackageList_MixedDescAndNoDesc(t *testing.T) {
	// Some entries have Desc, some don't — triggers the hasDesc=true branch.
	pkgs := config.PackageEntryList{
		{Name: "git", Desc: "Version control"},
		{Name: "curl"},
	}
	assert.NotPanics(t, func() {
		printPackageList("CLI tools", pkgs)
	})
}

func TestPrintPackageList_EmptyLabelAndName(t *testing.T) {
	// Edge case: empty strings must not panic.
	pkgs := config.PackageEntryList{{Name: "", Desc: ""}}
	assert.NotPanics(t, func() {
		printPackageList("", pkgs)
	})
}

// ---------------------------------------------------------------------------
// showScreenRecordingReminderFromPlan
// ---------------------------------------------------------------------------

func TestShowScreenRecordingReminderFromPlan_DryRun_NoOp(t *testing.T) {
	plan := InstallPlan{DryRun: true, Silent: false}
	assert.NotPanics(t, func() {
		showScreenRecordingReminderFromPlan(plan)
	})
}

func TestShowScreenRecordingReminderFromPlan_Silent_NoOp(t *testing.T) {
	plan := InstallPlan{DryRun: false, Silent: true}
	assert.NotPanics(t, func() {
		showScreenRecordingReminderFromPlan(plan)
	})
}

func TestShowScreenRecordingReminderFromPlan_NoTriggerPackages(t *testing.T) {
	// Plan has no screen-recording trigger packages — should be a no-op.
	plan := InstallPlan{
		DryRun: false,
		Silent: true, // suppress any interactive prompts
		SelectedPkgs: map[string]bool{
			"git":  true,
			"curl": true,
		},
	}
	assert.NotPanics(t, func() {
		showScreenRecordingReminderFromPlan(plan)
	})
}

func TestShowScreenRecordingReminderFromPlan_DryRunAndSilentBothTrue(t *testing.T) {
	plan := InstallPlan{DryRun: true, Silent: true}
	assert.NotPanics(t, func() {
		showScreenRecordingReminderFromPlan(plan)
	})
}

// ---------------------------------------------------------------------------
// checkDependencies
// ---------------------------------------------------------------------------

func TestCheckDependencies_DryRun_AlwaysNil(t *testing.T) {
	opts := &config.InstallOptions{DryRun: true}
	st := &config.InstallState{}
	err := checkDependencies(opts, st)
	assert.NoError(t, err)
}

func TestCheckDependencies_DryRun_PackagesOnly(t *testing.T) {
	opts := &config.InstallOptions{DryRun: true, PackagesOnly: true}
	st := &config.InstallState{}
	err := checkDependencies(opts, st)
	assert.NoError(t, err)
}

func TestCheckDependencies_DryRun_SilentAndPackagesOnly(t *testing.T) {
	opts := &config.InstallOptions{
		DryRun:       true,
		Silent:       true,
		PackagesOnly: true,
	}
	st := &config.InstallState{}
	err := checkDependencies(opts, st)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// hasDotfiles
// ---------------------------------------------------------------------------

func TestHasDotfiles_Skip_ReturnsFalse(t *testing.T) {
	t.Setenv("OPENBOOT_DOTFILES", "")
	opts := &config.InstallOptions{Dotfiles: "skip"}
	st := &config.InstallState{}
	assert.False(t, hasDotfiles(opts, st))
}

func TestHasDotfiles_OptsURLSet_ReturnsTrue(t *testing.T) {
	t.Setenv("OPENBOOT_DOTFILES", "")
	opts := &config.InstallOptions{DotfilesURL: "https://github.com/user/dotfiles"}
	st := &config.InstallState{}
	assert.True(t, hasDotfiles(opts, st))
}

func TestHasDotfiles_EnvVarSet_ReturnsTrue(t *testing.T) {
	t.Setenv("OPENBOOT_DOTFILES", "https://github.com/envuser/dotfiles")
	opts := &config.InstallOptions{}
	st := &config.InstallState{}
	assert.True(t, hasDotfiles(opts, st))
}

func TestHasDotfiles_NoURLAnywhere_ReturnsFalse(t *testing.T) {
	t.Setenv("OPENBOOT_DOTFILES", "")
	opts := &config.InstallOptions{DotfilesURL: ""}
	st := &config.InstallState{}
	assert.False(t, hasDotfiles(opts, st))
}

func TestHasDotfiles_SkipOverridesEnvVar(t *testing.T) {
	t.Setenv("OPENBOOT_DOTFILES", "https://github.com/envuser/dotfiles")
	opts := &config.InstallOptions{Dotfiles: "skip"}
	st := &config.InstallState{}
	// "skip" always wins, even when env var is set.
	assert.False(t, hasDotfiles(opts, st))
}

// ---------------------------------------------------------------------------
// showCompletionFromPlan (via Apply integration path)
// ---------------------------------------------------------------------------

func TestShowCompletionFromPlan_NoErrors(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := InstallPlan{
		DryRun:       true,
		Silent:       true,
		PackagesOnly: true,
		Formulae:     []string{"git", "curl"},
		Casks:        []string{"firefox"},
		Npm:          []string{"typescript"},
	}
	assert.NotPanics(t, func() {
		showCompletionFromPlan(plan, NopReporter{}, 0)
	})
}

func TestShowCompletionFromPlan_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := InstallPlan{
		DryRun:       true,
		Silent:       true,
		PackagesOnly: true,
		Formulae:     []string{"git"},
		Casks:        []string{},
		Npm:          []string{},
	}
	assert.NotPanics(t, func() {
		showCompletionFromPlan(plan, NopReporter{}, 2)
	})
}

func TestShowCompletionFromPlan_WithNpm(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := InstallPlan{
		DryRun:       true,
		Silent:       true,
		PackagesOnly: false,
		Formulae:     []string{"git"},
		Casks:        []string{},
		Npm:          []string{"typescript", "eslint"},
	}
	assert.NotPanics(t, func() {
		showCompletionFromPlan(plan, NopReporter{}, 0)
	})
}

// ---------------------------------------------------------------------------
// Apply (happy path with NopReporter through DryRun)
// ---------------------------------------------------------------------------

func TestApply_DryRun_PackagesOnly_EmptyPlan(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := InstallPlan{
		DryRun:       true,
		Silent:       true,
		PackagesOnly: true,
	}
	err := Apply(plan, NopReporter{})
	require.NoError(t, err)
}

func TestApply_DryRun_WithFormulaeAndCasks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := InstallPlan{
		DryRun:       true,
		Silent:       true,
		PackagesOnly: true,
		Formulae:     []string{"git", "curl"},
		Casks:        []string{"firefox"},
	}
	err := Apply(plan, NopReporter{})
	require.NoError(t, err)
}

func TestApply_DryRun_SkipGit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := InstallPlan{
		DryRun:   true,
		Silent:   true,
		SkipGit:  true,
		Formulae: []string{"git"},
	}
	err := Apply(plan, NopReporter{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Plan (top-level, with RemoteConfig)
// ---------------------------------------------------------------------------

func TestPlan_RemoteConfig_Taps(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
		RemoteConfig: &config.RemoteConfig{
			Username: "testuser",
			Slug:     "default",
			Packages: config.PackageEntryList{{Name: "git"}},
			Taps:     []string{"homebrew/cask", "homebrew/core"},
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	assert.Equal(t, []string{"homebrew/cask", "homebrew/core"}, plan.Taps)
}

func TestPlan_RemoteConfig_NpmPackages(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
		RemoteConfig: &config.RemoteConfig{
			Username: "testuser",
			Slug:     "default",
			Npm:      config.PackageEntryList{{Name: "typescript"}, {Name: "eslint"}},
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	assert.Contains(t, plan.Npm, "typescript")
	assert.Contains(t, plan.Npm, "eslint")
}

func TestPlan_RemoteConfig_ShellOhMyZsh(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
		RemoteConfig: &config.RemoteConfig{
			Username: "testuser",
			Slug:     "default",
			Shell:    &config.RemoteShellConfig{OhMyZsh: true},
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	assert.True(t, plan.InstallOhMyZsh)
}

func TestPlan_RemoteConfig_MacOSPrefs(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
		RemoteConfig: &config.RemoteConfig{
			Username: "testuser",
			Slug:     "default",
			MacOSPrefs: []config.RemoteMacOSPref{
				{Domain: "com.apple.dock", Key: "autohide", Type: "bool", Value: "true", Desc: "Auto-hide dock"},
			},
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	require.Len(t, plan.MacOSPrefs, 1)
	assert.Equal(t, "com.apple.dock", plan.MacOSPrefs[0].Domain)
	assert.Equal(t, "bool", plan.MacOSPrefs[0].Type)
}

func TestPlan_RemoteConfig_MacOSPrefs_InferredType(t *testing.T) {
	// When Type is empty, planFromRemoteConfig should infer it.
	cfg := &config.Config{
		DryRun: true,
		RemoteConfig: &config.RemoteConfig{
			Username: "testuser",
			Slug:     "default",
			MacOSPrefs: []config.RemoteMacOSPref{
				{Domain: "com.apple.dock", Key: "autohide", Type: "", Value: "true"},
			},
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	require.Len(t, plan.MacOSPrefs, 1)
	// Type must be inferred (non-empty).
	assert.NotEmpty(t, plan.MacOSPrefs[0].Type)
}

func TestPlan_RemoteConfig_PostInstall(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
		RemoteConfig: &config.RemoteConfig{
			Username:    "testuser",
			Slug:        "default",
			PostInstall: []string{"mise install", "npm install -g pnpm"},
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	assert.Equal(t, []string{"mise install", "npm install -g pnpm"}, plan.PostInstall)
}

func TestPlan_RemoteConfig_DotfilesFromOpts(t *testing.T) {
	// When RemoteConfig has no DotfilesRepo, fall back to opts.DotfilesURL.
	cfg := &config.Config{
		DryRun:      true,
		DotfilesURL: "https://github.com/opts/dotfiles",
		RemoteConfig: &config.RemoteConfig{
			Username:     "testuser",
			Slug:         "default",
			DotfilesRepo: "",
		},
	}
	opts := cfg.ToInstallOptions()
	st := cfg.ToInstallState()
	plan, err := Plan(opts, st)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/opts/dotfiles", plan.DotfilesURL)
}
