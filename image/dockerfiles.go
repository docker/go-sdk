package image

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

// Single regex to handle both ${VAR_NAME:-default_value} and ${VAR_NAME} patterns
var buildArgPattern = regexp.MustCompile(`\$\{([^}:-]+)(?::-([^}]*))?\}`)

// ImagesFromDockerfile extracts images from the Dockerfile sourced from dockerfile.
func ImagesFromDockerfile(dockerfile string, buildArgs map[string]*string) ([]string, error) {
	file, err := os.Open(dockerfile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ImagesFromReader(file, buildArgs)
}

// ImagesFromReader extracts images from the Dockerfile sourced from r.
func ImagesFromReader(r io.Reader, buildArgs map[string]*string) ([]string, error) {
	var images []string
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	// extract images from dockerfile
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), "FROM") {
			continue
		}

		// remove FROM
		line = strings.TrimPrefix(line, "FROM")
		parts := strings.Split(strings.TrimSpace(line), " ")
		if len(parts) == 0 {
			continue
		}

		parts[0] = handleBuildArgs(parts[0], buildArgs)
		images = append(images, parts[0])
	}

	return images, nil
}

// handleBuildArgs handles the build args in the Dockerfile.
// It replaces the ${VAR_NAME:-default_value} and ${VAR_NAME} patterns with the actual values.
// If the build arg is not provided, the default value is used.
// If the build arg is provided, the actual value is used.
// If the build arg is provided and the default value is not provided, the original syntax is kept.
func handleBuildArgs(part string, buildArgs map[string]*string) string {
	s := buildArgPattern.ReplaceAllStringFunc(part, func(match string) string {
		matches := buildArgPattern.FindStringSubmatch(match)

		varName := matches[1]
		hasDefault := len(matches) == 3 && matches[2] != ""
		defaultValue := ""
		if hasDefault {
			defaultValue = matches[2]
		}

		// Check if build arg is provided and not nil
		if buildArg, exists := buildArgs[varName]; exists && buildArg != nil {
			return *buildArg
		}

		// Use default value if available
		if hasDefault {
			return defaultValue
		}

		// For ${VAR} without default and no build arg, keep original syntax
		return match
	})

	return s
}
