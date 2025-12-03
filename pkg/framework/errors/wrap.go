package errors

import "fmt"

// WrapKubernetes wraps an error with Kubernetes context
func WrapKubernetes(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrKubernetes, context, err)
}

// WrapStorage wraps an error with storage context
func WrapStorage(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrStorage, context, err)
}

// WrapInvalid wraps an error with invalid input context
func WrapInvalid(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrInvalid, context, err)
}

// WrapNotFound wraps an error with not found context
func WrapNotFound(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrNotFound, context, err)
}

// WrapInvalidYAML wraps an error with invalid YAML context
func WrapInvalidYAML(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrInvalidYAML, context, err)
}

