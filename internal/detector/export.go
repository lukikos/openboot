package detector

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openbootdotdev/openboot/internal/config"
	"gopkg.in/yaml.v3"
)

// ToProjectConfig converts a list of detections into a ProjectConfig
// suitable for saving as .openboot.yml.
func ToProjectConfig(detections []Detection) *config.ProjectConfig {
	pc := &config.ProjectConfig{
		Version: "1.0",
		Brew:    &config.BrewConfig{},
	}

	for _, d := range detections {
		if d.IsCask {
			pc.Brew.Casks = append(pc.Brew.Casks, d.Package)
		} else {
			pc.Brew.Packages = append(pc.Brew.Packages, d.Package)
		}
	}

	return pc
}

// SaveProjectConfig writes detections as .openboot.yml in the given directory.
func SaveProjectConfig(dir string, detections []Detection) error {
	pc := ToProjectConfig(detections)

	data, err := yaml.Marshal(pc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := filepath.Join(dir, config.ProjectConfigFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", config.ProjectConfigFileName, err)
	}

	return nil
}
