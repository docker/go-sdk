package image

import (
	"archive/tar"
	"bufio"
	"errors"
	"fmt"
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
// Use this function if you want to extract images from a Dockerfile that is not in a tar reader.
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

// ImagesFromTarReader extracts images from the Dockerfile sourced from a tar reader.
// The name of the Dockerfile in the tar reader must be the same as the dockerfile parameter.
// Use this function if you want to extract images from a Dockerfile that is in a tar reader.
func ImagesFromTarReader(r io.ReadSeeker, dockerfile string, buildArgs map[string]*string) ([]string, error) {
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("dockerfile %q not found in context archive", dockerfile)
			}

			return nil, fmt.Errorf("reading tar archive: %w", err)
		}

		if hdr.Name != dockerfile {
			continue
		}

		images, err := ImagesFromReader(tr, buildArgs)
		if err != nil {
			return nil, fmt.Errorf("extract images from Dockerfile: %w", err)
		}

		// Reset the archive to the beginning.
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek context archive to start: %w", err)
		}

		return images, nil
	}
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
