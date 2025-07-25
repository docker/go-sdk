package context

import (
	"errors"
	"fmt"

	"github.com/docker/go-sdk/config"
)

// Delete deletes a context. The context must exist: it must have been created with [New]
// or inspected with [Inspect].
// If the context is the default context, the current context will be reset to the default context.
func (ctx *Context) Delete() error {
	if ctx.encodedName == "" {
		return errors.New("context has no encoded name")
	}

	metaRoot, err := metaRoot()
	if err != nil {
		return fmt.Errorf("meta root: %w", err)
	}

	s := &store{root: metaRoot}

	err = s.delete(ctx.encodedName)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	if ctx.isCurrent {
		// reset the current context to the default context
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		cfg.CurrentContext = DefaultContextName

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
	}

	return nil
}
