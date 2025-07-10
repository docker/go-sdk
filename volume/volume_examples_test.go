package volume_test

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-sdk/volume"
)

func ExampleNew() {
	v, err := volume.New(context.Background(), volume.WithName("my-volume"))
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err := v.Terminate(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	fmt.Println(v.Name)
	fmt.Println(v.ID())

	// Output:
	// my-volume
	// my-volume
}

func ExampleFindByID() {
	v, err := volume.New(context.Background(), volume.WithName("my-volume-id"))
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err := v.Terminate(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	vol, err := volume.FindByID(context.Background(), "my-volume-id")
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(vol.ID())
	fmt.Println(vol.Name)

	// Output:
	// my-volume-id
	// my-volume-id
}

func ExampleList() {
	v, err := volume.New(context.Background(), volume.WithName("my-volume-list"), volume.WithLabels(map[string]string{"volume.type": "example-test"}))
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err := v.Terminate(context.Background()); err != nil {
			log.Println(err)
		}
	}()

	vols, err := volume.List(context.Background(), volume.WithFilters(filters.NewArgs(filters.Arg("label", "volume.type=example-test"))))
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(len(vols))
	for _, v := range vols {
		fmt.Println(v.ID())
		fmt.Println(v.Name)
	}

	// Output:
	// 1
	// my-volume-list
	// my-volume-list
}
