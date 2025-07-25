package context

// contextOptions is the options for creating a context.
type contextOptions struct {
	host             string
	description      string
	additionalFields map[string]any
	skipTLSVerify    bool
	current          bool
}

// CreateContextOption is a function that can be used to create a context.
type CreateContextOption func(*contextOptions) error

// WithHost sets the host for the context.
func WithHost(host string) CreateContextOption {
	return func(c *contextOptions) error {
		c.host = host
		return nil
	}
}

// WithDescription sets the description for the context.
func WithDescription(description string) CreateContextOption {
	return func(c *contextOptions) error {
		c.description = description
		return nil
	}
}

// WithAdditionalFields sets the additional fields for the context.
func WithAdditionalFields(fields map[string]any) CreateContextOption {
	return func(c *contextOptions) error {
		c.additionalFields = fields
		return nil
	}
}

// WithSkipTLSVerify sets the skipTLSVerify flag to true.
func WithSkipTLSVerify() CreateContextOption {
	return func(c *contextOptions) error {
		c.skipTLSVerify = true
		return nil
	}
}

// AsCurrent sets the context as the current context.
func AsCurrent() CreateContextOption {
	return func(c *contextOptions) error {
		c.current = true
		return nil
	}
}
