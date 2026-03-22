package detector

// Confidence indicates how certain we are that a dependency is needed.
type Confidence int

const (
	// ConfidenceRequired means the file exists and the dependency is certainly needed.
	// Pre-selected in TUI, auto-installed in --auto mode.
	ConfidenceRequired Confidence = iota

	// ConfidenceRecommended means the dependency was inferred from file contents.
	// Pre-selected in TUI, auto-installed in --auto mode.
	ConfidenceRecommended

	// ConfidenceOptional means the dependency may or may not be needed locally
	// (e.g. docker-compose services). NOT pre-selected, NOT auto-installed.
	ConfidenceOptional
)

// Detection represents a single detected dependency.
type Detection struct {
	Package     string     `json:"package"`
	IsCask      bool       `json:"cask,omitempty"`
	Source      string     `json:"source"`
	Confidence  Confidence `json:"-"`
	Version     string     `json:"version,omitempty"`
	Description string     `json:"description,omitempty"`
	Installed   bool       `json:"installed"`
}

// ScanResult is the immutable output of scanning a project directory.
type ScanResult struct {
	Dir        string      `json:"dir"`
	Detected   []Detection `json:"detected"`
	Satisfied  bool        `json:"satisfied"`
	Missing    []string    `json:"missing"`
	InstalledNow []string  `json:"installed_now,omitempty"`
}

// MissingDetections returns only detections that are not installed.
func (r ScanResult) MissingDetections() []Detection {
	var missing []Detection
	for _, d := range r.Detected {
		if !d.Installed {
			missing = append(missing, d)
		}
	}
	return missing
}

// NonOptionalMissing returns missing detections that are Required or Recommended.
func (r ScanResult) NonOptionalMissing() []Detection {
	var result []Detection
	for _, d := range r.Detected {
		if !d.Installed && d.Confidence != ConfidenceOptional {
			result = append(result, d)
		}
	}
	return result
}
