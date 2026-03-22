package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// imageToPackage maps well-known Docker image names to Homebrew formula names.
var imageToPackage = map[string]struct {
	pkg         string
	description string
}{
	"postgres":      {pkg: "postgresql@16", description: "PostgreSQL database"},
	"postgis":       {pkg: "postgresql@16", description: "PostgreSQL database"},
	"redis":         {pkg: "redis", description: "Redis key-value store"},
	"mysql":         {pkg: "mysql", description: "MySQL database"},
	"mariadb":       {pkg: "mariadb", description: "MariaDB database"},
	"mongo":         {pkg: "mongodb-community", description: "MongoDB database"},
	"mongodb":       {pkg: "mongodb-community", description: "MongoDB database"},
	"rabbitmq":      {pkg: "rabbitmq", description: "RabbitMQ message broker"},
	"memcached":     {pkg: "memcached", description: "Memcached caching system"},
	"elasticsearch": {pkg: "elasticsearch", description: "Elasticsearch search engine"},
	"minio":         {pkg: "minio", description: "MinIO object storage"},
}

// detectDockerComposeServices parses a docker-compose file and returns
// detections for well-known service images.
func detectDockerComposeServices(dir, file string) []Detection {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return nil
	}

	var compose struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var detections []Detection

	for serviceName, svc := range compose.Services {
		if svc.Image == "" {
			continue
		}

		// Extract base image name (strip tag, registry prefix)
		imageName := baseImageName(svc.Image)

		info, ok := imageToPackage[imageName]
		if !ok {
			continue
		}

		// Deduplicate by package name
		if seen[info.pkg] {
			continue
		}
		seen[info.pkg] = true

		detections = append(detections, Detection{
			Package:     info.pkg,
			Source:      fmt.Sprintf("%s (service: %s)", file, serviceName),
			Confidence:  ConfidenceOptional,
			Description: info.description,
		})
	}

	return detections
}

// baseImageName extracts the base name from a Docker image reference.
// "postgres:16" -> "postgres", "library/redis:7-alpine" -> "redis"
func baseImageName(image string) string {
	// Remove tag
	if i := strings.Index(image, ":"); i != -1 {
		image = image[:i]
	}
	// Remove registry/namespace prefix — take last segment
	parts := strings.Split(image, "/")
	return parts[len(parts)-1]
}
