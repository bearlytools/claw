package clawiter

// walkOptions holds configuration for Walk.
type walkOptions struct {
	// Future options can be added here.
}

// WalkOption configures Walk behavior.
type WalkOption func(walkOptions) (walkOptions, error)
