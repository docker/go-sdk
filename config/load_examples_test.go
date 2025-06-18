package config_test

import (
	"fmt"
	"log"
	"strings"

	"github.com/docker/go-sdk/config"
)

func ExampleDir() {
	dir, err := config.Dir()
	fmt.Println(err)
	fmt.Println(strings.HasSuffix(dir, ".docker"))

	// Output:
	// <nil>
	// true
}

func ExampleFilepath() {
	filepath, err := config.Filepath()
	if err != nil {
		log.Printf("error getting config filepath: %s", err)
		return
	}

	fmt.Println(strings.HasSuffix(filepath, "config.json"))

	// Output:
	// true
}

func ExampleLoad() {
	cfg, err := config.Load()
	fmt.Println(err)
	fmt.Println(len(cfg.AuthConfigs) > 0)

	// Output:
	// <nil>
	// true
}
