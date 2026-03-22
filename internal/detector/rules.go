package detector

// rule defines a single detection rule mapping project files to dependencies.
type rule struct {
	files       []string
	pkg         string
	isCask      bool
	confidence  Confidence
	description string
	versionFunc func(dir, file string) string
}

// rules is the declarative detection table. Adding a new detection is a single entry.
// Default confidence is ConfidenceRequired (file exists = dependency needed).
var rules = []rule{
	// Languages & runtimes — high confidence
	{
		files:       []string{"package.json", ".nvmrc", ".node-version"},
		pkg:         "node",
		description: "Node.js runtime",
		versionFunc: extractNodeVersion,
	},
	{
		files:       []string{"go.mod"},
		pkg:         "go",
		description: "Go programming language",
		versionFunc: extractGoVersion,
	},
	{
		files:       []string{"Cargo.toml", "rust-toolchain.toml"},
		pkg:         "rust",
		description: "Rust toolchain",
	},
	{
		files:       []string{"pyproject.toml", "requirements.txt", "Pipfile", ".python-version", "setup.py"},
		pkg:         "python@3",
		description: "Python 3",
		versionFunc: extractPythonVersion,
	},
	{
		files:       []string{"Gemfile", ".ruby-version"},
		pkg:         "ruby",
		description: "Ruby",
	},
	{
		files:       []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		pkg:         "openjdk",
		description: "Java Development Kit",
	},
	{
		files:       []string{"composer.json"},
		pkg:         "php",
		description: "PHP",
	},
	{
		files:       []string{"mix.exs"},
		pkg:         "elixir",
		description: "Elixir",
	},
	{
		files:       []string{"pubspec.yaml"},
		pkg:         "flutter",
		isCask:      true,
		description: "Flutter SDK",
	},
	{
		files:       []string{"Package.swift"},
		pkg:         "swift",
		description: "Swift (Xcode CLT)",
	},

	// Build tools
	{
		files:       []string{"CMakeLists.txt"},
		pkg:         "cmake",
		description: "CMake build system",
	},

	// Container tools — recommended confidence
	{
		files:       []string{"Dockerfile"},
		pkg:         "docker",
		isCask:      true,
		confidence:  ConfidenceRecommended,
		description: "Docker Desktop",
	},
	{
		files:       []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"},
		pkg:         "docker",
		isCask:      true,
		confidence:  ConfidenceRecommended,
		description: "Docker Desktop",
	},

	// Infrastructure
	{
		files:       []string{".terraform.lock.hcl"},
		pkg:         "terraform",
		confidence:  ConfidenceRecommended,
		description: "Terraform",
	},
}
