package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	goVersionRe   = regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)
	majorVersionRe = regexp.MustCompile(`(\d+)`)
)

// extractGoVersion reads the Go version from go.mod (e.g. "go 1.24").
func extractGoVersion(dir, file string) string {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return ""
	}

	matches := goVersionRe.FindSubmatch(data)
	if len(matches) >= 2 {
		return string(matches[1])
	}
	return ""
}

// extractNodeVersion reads the Node version from .nvmrc, .node-version,
// or package.json engines.node.
func extractNodeVersion(dir, file string) string {
	path := filepath.Join(dir, file)

	// .nvmrc or .node-version: file contains just the version
	if file == ".nvmrc" || file == ".node-version" {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		v := strings.TrimSpace(string(data))
		// Strip leading "v" if present
		v = strings.TrimPrefix(v, "v")
		if v != "" {
			return v
		}
		return ""
	}

	// package.json: read engines.node
	if file == "package.json" {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}

		var pkg struct {
			Engines struct {
				Node string `json:"node"`
			} `json:"engines"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			return ""
		}

		if pkg.Engines.Node != "" {
			// Extract major version from semver ranges like ">=20", "^20.0.0", "20.x"
			matches := majorVersionRe.FindString(pkg.Engines.Node)
			return matches
		}
	}

	return ""
}

// extractPythonVersion reads the Python version from .python-version.
func extractPythonVersion(dir, file string) string {
	if file != ".python-version" {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return ""
	}

	v := strings.TrimSpace(string(data))
	if v != "" {
		return v
	}
	return ""
}
