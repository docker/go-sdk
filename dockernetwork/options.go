package dockernetwork

import (
	"errors"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
)

type options struct {
	client     *dockerclient.Client
	ipam       *network.IPAM
	labels     map[string]string
	driver     string
	name       string
	attachable bool
	enableIPv6 bool
	internal   bool
}

// Option is a function that modifies the options to create a network.
type Option func(*options) error

// WithClient sets the docker client.
func WithClient(client *dockerclient.Client) Option {
	return func(o *options) error {
		o.client = client
		return nil
	}
}

// WithName sets the name of the network.
func WithName(name string) Option {
	return func(o *options) error {
		if name == "" {
			return errors.New("name is required")
		}

		o.name = name
		return nil
	}
}

// WithDriver sets the driver of the network.
func WithDriver(driver string) Option {
	return func(o *options) error {
		o.driver = driver
		return nil
	}
}

// WithInternal makes the network internal.
func WithInternal() Option {
	return func(o *options) error {
		o.internal = true
		return nil
	}
}

// WithEnableIPv6 enables IPv6 on the network.
func WithEnableIPv6() Option {
	return func(o *options) error {
		o.enableIPv6 = true
		return nil
	}
}

// WithAttachable makes the network attachable.
func WithAttachable() Option {
	return func(o *options) error {
		o.attachable = true
		return nil
	}
}

// WithLabels sets the labels of the network.
func WithLabels(labels map[string]string) Option {
	return func(o *options) error {
		o.labels = labels
		return nil
	}
}

// WithIPAM sets the IPAM of the network.
func WithIPAM(ipam *network.IPAM) Option {
	return func(o *options) error {
		o.ipam = ipam
		return nil
	}
}
