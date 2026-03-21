//go:build integration

package integration

import (
	"testing"

	"github.com/openbootdotdev/openboot/internal/brew"
	"github.com/openbootdotdev/openboot/internal/cleaner"
	"github.com/openbootdotdev/openboot/internal/snapshot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Cleaner_DiffFromLists_AllDesiredInstalled(t *testing.T) {
	// Given: brew is installed and we know what's installed
	require.True(t, brew.IsInstalled(), "brew must be installed")
	leaves, err := brew.GetInstalledLeaves()
	require.NoError(t, err)
	_, casks, err := brew.GetInstalledPackages()
	require.NoError(t, err)

	desiredFormulae := make([]string, 0, len(leaves))
	for name := range leaves {
		desiredFormulae = append(desiredFormulae, name)
	}
	desiredCasks := make([]string, 0, len(casks))
	for name := range casks {
		desiredCasks = append(desiredCasks, name)
	}

	// When: desired == installed leaves
	result, err := cleaner.DiffFromLists(desiredFormulae, desiredCasks, nil, nil)

	// Then: no extras detected
	require.NoError(t, err)
	assert.Empty(t, result.ExtraFormulae, "no extra formulae when desired matches installed leaves")
	assert.Empty(t, result.ExtraCasks, "no extra casks when desired matches installed")
}

func TestIntegration_Cleaner_DiffFromLists_EmptyDesired(t *testing.T) {
	// Given: brew is installed with at least one package
	require.True(t, brew.IsInstalled(), "brew must be installed")
	leaves, err := brew.GetInstalledLeaves()
	require.NoError(t, err)
	_, casks, err := brew.GetInstalledPackages()
	require.NoError(t, err)

	// When: desired lists are empty (want nothing installed)
	result, err := cleaner.DiffFromLists(nil, nil, nil, nil)

	// Then: all top-level formulae (leaves) show up as extra
	require.NoError(t, err)
	assert.Equal(t, len(leaves), len(result.ExtraFormulae),
		"all installed leaves should appear as extra when desired is empty")
	assert.Equal(t, len(casks), len(result.ExtraCasks),
		"all installed casks should appear as extra when desired is empty")
	t.Logf("Extra formulae (leaves): %d, extra casks: %d", len(result.ExtraFormulae), len(result.ExtraCasks))
}

func TestIntegration_Cleaner_DiffFromLists_SubsetDesired(t *testing.T) {
	// Given: brew is installed with multiple top-level packages
	require.True(t, brew.IsInstalled(), "brew must be installed")
	leaves, err := brew.GetInstalledLeaves()
	require.NoError(t, err)
	if len(leaves) < 2 {
		t.Skip("need at least 2 installed leaves for subset test")
	}

	// When: desired is exactly one installed leaf formula
	var oneFormula string
	for name := range leaves {
		oneFormula = name
		break
	}
	result, err := cleaner.DiffFromLists([]string{oneFormula}, nil, nil, nil)

	// Then: all other installed leaves appear as extras
	require.NoError(t, err)
	assert.NotContains(t, result.ExtraFormulae, oneFormula, "desired package should not appear as extra")
	assert.Equal(t, len(leaves)-1, len(result.ExtraFormulae),
		"all leaves except the desired one should be extra")
}

func TestIntegration_Cleaner_DiffFromSnapshot_CurrentState(t *testing.T) {
	// Given: brew is installed; we build a snapshot from current installed state (leaves)
	require.True(t, brew.IsInstalled(), "brew must be installed")
	leaves, err := brew.GetInstalledLeaves()
	require.NoError(t, err)
	_, casks, err := brew.GetInstalledPackages()
	require.NoError(t, err)

	installedLeaves := make([]string, 0, len(leaves))
	for name := range leaves {
		installedLeaves = append(installedLeaves, name)
	}
	installedCasks := make([]string, 0, len(casks))
	for name := range casks {
		installedCasks = append(installedCasks, name)
	}

	snap := &snapshot.Snapshot{
		Packages: snapshot.PackageSnapshot{
			Formulae: installedLeaves,
			Casks:    installedCasks,
		},
	}

	// When: we diff against a snapshot that matches current leaves
	result, err := cleaner.DiffFromSnapshot(snap)

	// Then: no extras
	require.NoError(t, err)
	assert.Empty(t, result.ExtraFormulae, "no extra formulae when snapshot matches installed leaves")
	assert.Empty(t, result.ExtraCasks, "no extra casks when snapshot matches installed")
}

func TestIntegration_Cleaner_Execute_DryRun_NoChanges(t *testing.T) {
	// Given: brew is installed; build a result with formulae we know exist but won't actually remove
	require.True(t, brew.IsInstalled(), "brew must be installed")

	result := &cleaner.CleanResult{
		ExtraFormulae: []string{"wget"},
		ExtraCasks:    []string{},
	}

	// When: Execute in dry-run mode
	err := cleaner.Execute(result, true)

	// Then: no error, nothing removed from the system
	assert.NoError(t, err)
	t.Logf("Dry-run removed: %d formulae, %d casks", len(result.RemovedFormulae), len(result.RemovedCasks))
}

func TestIntegration_Cleaner_CleanResult_TotalMethods(t *testing.T) {
	// Given: a real diff result from the current system
	require.True(t, brew.IsInstalled(), "brew must be installed")

	result, err := cleaner.DiffFromLists(nil, nil, nil, nil)
	require.NoError(t, err)

	// When: we query totals
	total := result.TotalExtra()

	// Then: total matches component counts
	assert.Equal(t, len(result.ExtraFormulae)+len(result.ExtraCasks)+len(result.ExtraNpm)+len(result.ExtraTaps), total)
	assert.Equal(t, 0, result.TotalRemoved(), "nothing removed yet")
	assert.Equal(t, 0, result.TotalFailed(), "nothing failed yet")
}
