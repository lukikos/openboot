package detector

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// composeFiles lists docker-compose file names that trigger service detection.
var composeFiles = map[string]bool{
	"docker-compose.yml":  true,
	"docker-compose.yaml": true,
	"compose.yml":         true,
	"compose.yaml":        true,
}

// Scan examines the given directory for known project files and returns
// detected dependencies. It does NOT check installed state — use Enrich for that.
func Scan(dir string) (ScanResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ScanResult{}, err
	}

	detections := make(map[string]Detection) // keyed by package name

	for _, r := range rules {
		for _, file := range r.files {
			path := filepath.Join(absDir, file)
			if _, err := os.Stat(path); err != nil {
				continue
			}

			// File exists — this rule matched
			d := Detection{
				Package:     r.pkg,
				IsCask:      r.isCask,
				Source:      file,
				Confidence:  r.confidence,
				Description: r.description,
			}

			// Try to extract version
			if r.versionFunc != nil {
				if v := r.versionFunc(absDir, file); v != "" {
					d.Version = v
				}
			}

			// Keep the detection with the most specific version info
			existing, exists := detections[r.pkg]
			if !exists || (existing.Version == "" && d.Version != "") {
				detections[r.pkg] = d
			}

			// Also detect docker-compose services
			if composeFiles[file] {
				for _, sd := range detectDockerComposeServices(absDir, file) {
					if _, exists := detections[sd.Package]; !exists {
						detections[sd.Package] = sd
					}
				}
			}

			// Don't break — check all files for this rule to find best version
		}
	}

	// Build ordered result
	result := ScanResult{Dir: absDir}
	for _, d := range detections {
		result.Detected = append(result.Detected, d)
	}

	// Sort: Required first, then Recommended, then Optional
	sortDetections(result.Detected)

	return result, nil
}

// Enrich takes a ScanResult and populates Installed, Missing, and Satisfied
// by checking against the provided installed package maps. Returns a new ScanResult.
func Enrich(result ScanResult, formulae, casks map[string]bool) ScanResult {
	enriched := ScanResult{
		Dir:      result.Dir,
		Detected: make([]Detection, len(result.Detected)),
	}
	copy(enriched.Detected, result.Detected)

	enriched.Satisfied = true
	for i := range enriched.Detected {
		d := &enriched.Detected[i]

		if d.IsCask {
			d.Installed = casks[d.Package]
		} else {
			d.Installed = formulae[d.Package] || formulae[stripVersion(d.Package)]
		}

		if !d.Installed && d.Confidence != ConfidenceOptional {
			enriched.Missing = append(enriched.Missing, d.Package)
			enriched.Satisfied = false
		}
	}

	return enriched
}

// stripVersion removes version suffix from package names like "python@3" -> "python".
// This handles cases where brew lists "python3" but the formula is "python@3".
func stripVersion(pkg string) string {
	if i := strings.Index(pkg, "@"); i != -1 {
		return pkg[:i]
	}
	return pkg
}

// sortDetections sorts by confidence (Required first) then alphabetically.
func sortDetections(detections []Detection) {
	sort.Slice(detections, func(i, j int) bool {
		if detections[i].Confidence != detections[j].Confidence {
			return detections[i].Confidence < detections[j].Confidence
		}
		return detections[i].Package < detections[j].Package
	})
}
