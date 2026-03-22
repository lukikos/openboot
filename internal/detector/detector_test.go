package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan_DetectsGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/test\n\ngo 1.24\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "go", result.Detected[0].Package)
	assert.Equal(t, "1.24", result.Detected[0].Version)
	assert.Equal(t, "go.mod", result.Detected[0].Source)
	assert.Equal(t, ConfidenceRequired, result.Detected[0].Confidence)
}

func TestScan_DetectsPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "engines": {"node": ">=20"}}`)

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "node", result.Detected[0].Package)
	assert.Equal(t, "20", result.Detected[0].Version)
}

func TestScan_DetectsNvmrc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".nvmrc", "v18\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "node", result.Detected[0].Package)
	assert.Equal(t, "18", result.Detected[0].Version)
	assert.Equal(t, ".nvmrc", result.Detected[0].Source)
}

func TestScan_PrefersVersionFromNvmrc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, ".nvmrc", "20\n")

	result, err := Scan(dir)
	require.NoError(t, err)

	// Should detect node only once, with version from .nvmrc
	nodeDetections := filterByPackage(result.Detected, "node")
	assert.Len(t, nodeDetections, 1)
	assert.Equal(t, "20", nodeDetections[0].Version)
}

func TestScan_DetectsPythonVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".python-version", "3.11.5\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "python@3", result.Detected[0].Package)
	assert.Equal(t, "3.11.5", result.Detected[0].Version)
}

func TestScan_DetectsRequirementsTxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==2.0\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "python@3", result.Detected[0].Package)
}

func TestScan_DetectsCargoToml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Cargo.toml", "[package]\nname = \"test\"\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "rust", result.Detected[0].Package)
}

func TestScan_DetectsDockerfile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Dockerfile", "FROM node:20\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "docker", result.Detected[0].Package)
	assert.True(t, result.Detected[0].IsCask)
	assert.Equal(t, ConfidenceRecommended, result.Detected[0].Confidence)
}

func TestScan_DetectsDockerComposeServices(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yml", `
services:
  db:
    image: postgres:16
  cache:
    image: redis:7-alpine
  app:
    build: .
`)

	result, err := Scan(dir)
	require.NoError(t, err)

	// Should detect docker (from docker-compose.yml) + postgres + redis
	assert.GreaterOrEqual(t, len(result.Detected), 3)

	docker := findDetection(result.Detected, "docker")
	require.NotNil(t, docker)
	assert.Equal(t, ConfidenceRecommended, docker.Confidence)

	pg := findDetection(result.Detected, "postgresql@16")
	require.NotNil(t, pg)
	assert.Equal(t, ConfidenceOptional, pg.Confidence)
	assert.Contains(t, pg.Source, "service: db")

	redis := findDetection(result.Detected, "redis")
	require.NotNil(t, redis)
	assert.Equal(t, ConfidenceOptional, redis.Confidence)
}

func TestScan_MultiLanguageProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module test\n\ngo 1.24\n")
	writeFile(t, dir, "package.json", `{"name": "frontend"}`)
	writeFile(t, dir, "requirements.txt", "django\n")
	writeFile(t, dir, "Dockerfile", "FROM node:20\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 4)

	assert.NotNil(t, findDetection(result.Detected, "go"))
	assert.NotNil(t, findDetection(result.Detected, "node"))
	assert.NotNil(t, findDetection(result.Detected, "python@3"))
	assert.NotNil(t, findDetection(result.Detected, "docker"))
}

func TestScan_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Detected)
}

func TestScan_DetectsJava(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pom.xml", "<project></project>")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "openjdk", result.Detected[0].Package)
}

func TestScan_DetectsRuby(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Gemfile", "source 'https://rubygems.org'\n")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "ruby", result.Detected[0].Package)
}

func TestScan_DetectsTerraform(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".terraform.lock.hcl", "")

	result, err := Scan(dir)
	require.NoError(t, err)
	assert.Len(t, result.Detected, 1)
	assert.Equal(t, "terraform", result.Detected[0].Package)
}

func TestScan_SortsByConfidence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module test\n\ngo 1.24\n")
	writeFile(t, dir, "docker-compose.yml", `
services:
  db:
    image: postgres:16
`)

	result, err := Scan(dir)
	require.NoError(t, err)

	// Required should come before Recommended, which comes before Optional
	for i := 1; i < len(result.Detected); i++ {
		assert.GreaterOrEqual(t, int(result.Detected[i].Confidence), int(result.Detected[i-1].Confidence),
			"detections should be sorted by confidence")
	}
}

func TestBaseImageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"postgres:16", "postgres"},
		{"redis:7-alpine", "redis"},
		{"library/redis:7", "redis"},
		{"ghcr.io/user/custom:latest", "custom"},
		{"postgres", "postgres"},
		{"docker.io/library/mysql:8", "mysql"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, baseImageName(tt.input))
		})
	}
}

func TestExtractGoVersion(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{"standard", "module test\n\ngo 1.24\n", "1.24"},
		{"with patch", "module test\n\ngo 1.24.1\n", "1.24"},
		{"no version", "module test\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFile(t, dir, "go.mod", tt.content)
			assert.Equal(t, tt.expected, extractGoVersion(dir, "go.mod"))
		})
	}
}

func TestExtractNodeVersion(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		file     string
		content  string
		expected string
	}{
		{"nvmrc plain", ".nvmrc", "20\n", "20"},
		{"nvmrc with v", ".nvmrc", "v18.17.0\n", "18.17.0"},
		{"node-version", ".node-version", "20.10.0\n", "20.10.0"},
		{"package.json engines", "package.json", `{"engines":{"node":">=20"}}`, "20"},
		{"package.json no engines", "package.json", `{"name":"test"}`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeFile(t, dir, tt.file, tt.content)
			assert.Equal(t, tt.expected, extractNodeVersion(dir, tt.file))
		})
	}
}

func TestExtractPythonVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".python-version", "3.11.5\n")
	assert.Equal(t, "3.11.5", extractPythonVersion(dir, ".python-version"))
}

func TestMissingDetections(t *testing.T) {
	result := ScanResult{
		Detected: []Detection{
			{Package: "go", Installed: true},
			{Package: "node", Installed: false},
			{Package: "redis", Installed: false},
		},
	}

	missing := result.MissingDetections()
	assert.Len(t, missing, 2)
	assert.Equal(t, "node", missing[0].Package)
	assert.Equal(t, "redis", missing[1].Package)
}

func TestNonOptionalMissing(t *testing.T) {
	result := ScanResult{
		Detected: []Detection{
			{Package: "go", Installed: false, Confidence: ConfidenceRequired},
			{Package: "redis", Installed: false, Confidence: ConfidenceOptional},
			{Package: "docker", Installed: false, Confidence: ConfidenceRecommended},
		},
	}

	nonOpt := result.NonOptionalMissing()
	assert.Len(t, nonOpt, 2)

	packages := make([]string, len(nonOpt))
	for i, d := range nonOpt {
		packages[i] = d.Package
	}
	assert.Contains(t, packages, "go")
	assert.Contains(t, packages, "docker")
}

func TestEnrich_AllInstalled(t *testing.T) {
	result := ScanResult{
		Dir: "/test",
		Detected: []Detection{
			{Package: "go", Source: "go.mod"},
			{Package: "node", Source: "package.json"},
		},
	}
	formulae := map[string]bool{"go": true, "node": true}
	casks := map[string]bool{}

	enriched := Enrich(result, formulae, casks)
	assert.True(t, enriched.Satisfied)
	assert.Empty(t, enriched.Missing)
	assert.True(t, enriched.Detected[0].Installed)
	assert.True(t, enriched.Detected[1].Installed)
}

func TestEnrich_SomeMissing(t *testing.T) {
	result := ScanResult{
		Dir: "/test",
		Detected: []Detection{
			{Package: "go", Source: "go.mod"},
			{Package: "node", Source: "package.json"},
		},
	}
	formulae := map[string]bool{"go": true}
	casks := map[string]bool{}

	enriched := Enrich(result, formulae, casks)
	assert.False(t, enriched.Satisfied)
	assert.Equal(t, []string{"node"}, enriched.Missing)
}

func TestEnrich_OnlyOptionalMissing_StillSatisfied(t *testing.T) {
	result := ScanResult{
		Dir: "/test",
		Detected: []Detection{
			{Package: "go", Source: "go.mod", Confidence: ConfidenceRequired},
			{Package: "redis", Source: "compose", Confidence: ConfidenceOptional},
		},
	}
	formulae := map[string]bool{"go": true}
	casks := map[string]bool{}

	enriched := Enrich(result, formulae, casks)
	assert.True(t, enriched.Satisfied, "should be satisfied when only optional deps are missing")
	assert.Empty(t, enriched.Missing)
}

func TestEnrich_CaskDetection(t *testing.T) {
	result := ScanResult{
		Dir: "/test",
		Detected: []Detection{
			{Package: "docker", IsCask: true, Source: "Dockerfile"},
		},
	}
	formulae := map[string]bool{}
	casks := map[string]bool{"docker": true}

	enriched := Enrich(result, formulae, casks)
	assert.True(t, enriched.Detected[0].Installed)
	assert.True(t, enriched.Satisfied)
}

func TestEnrich_VersionedPackage(t *testing.T) {
	result := ScanResult{
		Dir: "/test",
		Detected: []Detection{
			{Package: "python@3", Source: "requirements.txt"},
		},
	}
	// brew lists "python" not "python@3"
	formulae := map[string]bool{"python": true}
	casks := map[string]bool{}

	enriched := Enrich(result, formulae, casks)
	assert.True(t, enriched.Detected[0].Installed, "should match python@3 to python")
}

// --- helpers ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}

func findDetection(detections []Detection, pkg string) *Detection {
	for i := range detections {
		if detections[i].Package == pkg {
			return &detections[i]
		}
	}
	return nil
}

func filterByPackage(detections []Detection, pkg string) []Detection {
	var result []Detection
	for _, d := range detections {
		if d.Package == pkg {
			result = append(result, d)
		}
	}
	return result
}
