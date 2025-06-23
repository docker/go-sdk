package image_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/go-sdk/image"
)

func ExampleSave() {
	img := "redis:alpine"

	err := image.Pull(context.Background(), img)
	if err != nil {
		log.Println("error pulling image", err)
		return
	}

	tmpDir := os.TempDir()

	output := filepath.Join(tmpDir, "images.tar")
	err = image.Save(context.Background(), output, img)
	if err != nil {
		log.Println("error saving image", err)
		return
	}
	defer func() {
		err := os.Remove(output)
		if err != nil {
			log.Println("error removing image", err)
		}
	}()

	info, err := os.Stat(output)
	if err != nil {
		log.Println("error getting image info", err)
		return
	}

	fmt.Println(info.Size() > 0)

	// Output:
	// true
}
