package amigo

import "log/slog"

// APIOption configures an API when it is created.
type APIOption func(*API)

// WithLogger sets the logger used for server-side request failures. The
// default is [slog.Default].
func WithLogger(logger *slog.Logger) APIOption {
	if logger == nil {
		panic("amigo: logger cannot be nil")
	}
	return func(api *API) {
		api.logger = logger
	}
}
