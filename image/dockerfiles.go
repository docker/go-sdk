package image

import (
	"bufio"
	"io"
	"os"
	"strings"
)

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

		// interpolate build args
		for k, v := range buildArgs {
			if v != nil {
				parts[0] = strings.ReplaceAll(parts[0], "${"+k+"}", *v)
			}
		}
		images = append(images, parts[0])
	}

	return images, nil
}
