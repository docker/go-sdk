package context

import (
	"errors"
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/docker/go-sdk/config"
)

// New creates a new context.
//
// If the context already exists, it returns an error.
//
// If the [AsCurrent] option is passed, it updates the Docker config
// file, setting the current context to the new context.
func New(name string, opts ...CreateContextOption) (*Context, error) {
	switch name {
	case "":
		return nil, errors.New("name is required")
	case "default":
		return nil, errors.New("name cannot be 'default'")
	}

	_, err := Inspect(name)
	if err == nil {
		return nil, fmt.Errorf("context %s already exists", name)
	}

	defaultOptions := &contextOptions{}
	for _, opt := range opts {
		if err := opt(defaultOptions); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	ctx := &Context{
		Name:        name,
		encodedName: digest.FromString(name).Encoded(),
		Metadata: &Metadata{
			Description:      defaultOptions.description,
			additionalFields: defaultOptions.additionalFields,
		},
		Endpoints: map[string]*endpoint{
			"docker": {
				Host:          defaultOptions.host,
				SkipTLSVerify: defaultOptions.skipTLSVerify,
			},
		},
	}

	metaRoot, err := metaRoot()
	if err != nil {
		return nil, fmt.Errorf("meta root: %w", err)
	}

	s := &store{root: metaRoot}

	if err := s.add(ctx); err != nil {
		return nil, fmt.Errorf("add context: %w", err)
	}

	// set the context as the current context if the option is set
	if defaultOptions.current {
		cfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}

		cfg.CurrentContext = ctx.Name

		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}

		ctx.isCurrent = true
	}

	return ctx, nil
}
