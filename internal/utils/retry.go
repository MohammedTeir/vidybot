package utils

import (
	"context"
	"fmt"
	"time"
)

// RetryOptions defines options for retry behavior
type RetryOptions struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
	Logger      *EnhancedLogger
}

// DefaultRetryOptions returns default retry options
func DefaultRetryOptions() *RetryOptions {
	return &RetryOptions{
		MaxRetries:  3,
		InitialWait: 1 * time.Second,
		MaxWait:     30 * time.Second,
		Multiplier:  2.0,
	}
}

// WithLogger adds a logger to retry options
func (o *RetryOptions) WithLogger(logger *EnhancedLogger) *RetryOptions {
	o.Logger = logger
	return o
}

// WithMaxRetries sets the maximum number of retries
func (o *RetryOptions) WithMaxRetries(maxRetries int) *RetryOptions {
	o.MaxRetries = maxRetries
	return o
}

// WithInitialWait sets the initial wait time
func (o *RetryOptions) WithInitialWait(initialWait time.Duration) *RetryOptions {
	o.InitialWait = initialWait
	return o
}

// WithMaxWait sets the maximum wait time
func (o *RetryOptions) WithMaxWait(maxWait time.Duration) *RetryOptions {
	o.MaxWait = maxWait
	return o
}

// WithMultiplier sets the backoff multiplier
func (o *RetryOptions) WithMultiplier(multiplier float64) *RetryOptions {
	o.Multiplier = multiplier
	return o
}

// RetryFunc is a function that can be retried
type RetryFunc func() error

// RetryWithContext retries a function with exponential backoff
func RetryWithContext(ctx context.Context, fn RetryFunc, options *RetryOptions) error {
	if options == nil {
		options = DefaultRetryOptions()
	}

	var err error
	wait := options.InitialWait

	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		// Execute the function
		err = fn()
		if err == nil {
			return nil // Success
		}

		// Check if we've reached max retries
		if attempt == options.MaxRetries {
			if options.Logger != nil {
				options.Logger.Error("Max retries reached (%d): %v", options.MaxRetries, err)
			}
			return fmt.Errorf("max retries reached (%d): %w", options.MaxRetries, err)
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			if options.Logger != nil {
				options.Logger.Error("Context cancelled during retry: %v", ctx.Err())
			}
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		default:
			// Continue with retry
		}

		// Log the retry
		if options.Logger != nil {
			options.Logger.Warn("Retry attempt %d/%d after error: %v (waiting %v before next attempt)",
				attempt+1, options.MaxRetries, err, wait)
		}

		// Wait before next attempt with exponential backoff
		select {
		case <-time.After(wait):
			// Calculate next wait time with exponential backoff
			wait = time.Duration(float64(wait) * options.Multiplier)
			if wait > options.MaxWait {
				wait = options.MaxWait
			}
		case <-ctx.Done():
			if options.Logger != nil {
				options.Logger.Error("Context cancelled during retry wait: %v", ctx.Err())
			}
			return fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		}
	}

	// This should never happen due to the return in the loop, but just in case
	return err
}

// RetryWithContextAndResult retries a function that returns a result and error
func RetryWithContextAndResult[T any](ctx context.Context, fn func() (T, error), options *RetryOptions) (T, error) {
	if options == nil {
		options = DefaultRetryOptions()
	}

	var result T
	var err error
	wait := options.InitialWait

	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		// Execute the function
		result, err = fn()
		if err == nil {
			return result, nil // Success
		}

		// Check if we've reached max retries
		if attempt == options.MaxRetries {
			if options.Logger != nil {
				options.Logger.Error("Max retries reached (%d): %v", options.MaxRetries, err)
			}
			return result, fmt.Errorf("max retries reached (%d): %w", options.MaxRetries, err)
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			if options.Logger != nil {
				options.Logger.Error("Context cancelled during retry: %v", ctx.Err())
			}
			return result, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		default:
			// Continue with retry
		}

		// Log the retry
		if options.Logger != nil {
			options.Logger.Warn("Retry attempt %d/%d after error: %v (waiting %v before next attempt)",
				attempt+1, options.MaxRetries, err, wait)
		}

		// Wait before next attempt with exponential backoff
		select {
		case <-time.After(wait):
			// Calculate next wait time with exponential backoff
			wait = time.Duration(float64(wait) * options.Multiplier)
			if wait > options.MaxWait {
				wait = options.MaxWait
			}
		case <-ctx.Done():
			if options.Logger != nil {
				options.Logger.Error("Context cancelled during retry wait: %v", ctx.Err())
			}
			return result, fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		}
	}

	// This should never happen due to the return in the loop, but just in case
	return result, err
}
