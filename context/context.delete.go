package context

import (
	"errors"
	"fmt"
)

// Delete deletes a context. The context must exist: it must have been created with [New]
// or inspected with [Inspect].
func (ctx *Context) Delete() error {
	if ctx.encodedName == "" {
		return errors.New("context has no encoded name")
	}

	metaRoot, err := metaRoot()
	if err != nil {
		return fmt.Errorf("meta root: %w", err)
	}

	s := &store{root: metaRoot}

	return s.delete(ctx.encodedName)
}
