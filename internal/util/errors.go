package util

import (
	"errors"
	"fmt"
	"strings"
)

// Common error types for the Fleet CLI
var (
	// ErrInvalidConfig indicates a configuration error
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrClusterNotFound indicates a cluster was not found
	ErrClusterNotFound = errors.New("cluster not found")

	// ErrConnectionFailed indicates a connection failure
	ErrConnectionFailed = errors.New("connection failed")

	// ErrTimeout indicates an operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrCancelled indicates an operation was cancelled
	ErrCancelled = errors.New("operation cancelled")

	// ErrResourceNotFound indicates a Kubernetes resource was not found
	ErrResourceNotFound = errors.New("resource not found")

	// ErrInvalidResource indicates an invalid resource specification
	ErrInvalidResource = errors.New("invalid resource")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = errors.New("permission denied")

	// ErrAlreadyExists indicates a resource already exists
	ErrAlreadyExists = errors.New("already exists")

	// ErrShutdown indicates the system is shutting down
	ErrShutdown = errors.New("system shutting down")
)

// ClusterError wraps an error with cluster context
type ClusterError struct {
	ClusterName string
	Err         error
}

// Error implements the error interface
func (e *ClusterError) Error() string {
	return fmt.Sprintf("cluster %q: %v", e.ClusterName, e.Err)
}

// Unwrap returns the wrapped error for errors.Is/As compatibility
func (e *ClusterError) Unwrap() error {
	return e.Err
}

// WrapClusterError wraps an error with cluster context
func WrapClusterError(clusterName string, err error) error {
	if err == nil {
		return nil
	}
	return &ClusterError{
		ClusterName: clusterName,
		Err:         err,
	}
}

// MultiError aggregates multiple errors
type MultiError struct {
	Errors []error
}

// Error implements the error interface
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d errors occurred:", len(m.Errors)))
	for i, err := range m.Errors {
		if i < 10 { // Limit to first 10 errors in the message
			sb.WriteString(fmt.Sprintf("\n  %d. %v", i+1, err))
		} else if i == 10 {
			sb.WriteString(fmt.Sprintf("\n  ... and %d more errors", len(m.Errors)-10))
			break
		}
	}
	return sb.String()
}

// Unwrap returns the errors for errors.Is/As compatibility
func (m *MultiError) Unwrap() []error {
	return m.Errors
}

// Add adds an error to the multi-error
func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// ErrorOrNil returns nil if no errors were added, otherwise returns the MultiError
func (m *MultiError) ErrorOrNil() error {
	if len(m.Errors) == 0 {
		return nil
	}
	return m
}

// NewMultiError creates a new MultiError from a slice of errors
// It filters out nil errors
func NewMultiError(errors []error) *MultiError {
	m := &MultiError{
		Errors: make([]error, 0, len(errors)),
	}
	for _, err := range errors {
		if err != nil {
			m.Errors = append(m.Errors, err)
		}
	}
	return m
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (v *ValidationError) Error() string {
	if v.Value != nil {
		return fmt.Sprintf("validation failed for field %q (value: %v): %s", v.Field, v.Value, v.Message)
	}
	return fmt.Sprintf("validation failed for field %q: %s", v.Field, v.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// IsTimeout checks if an error is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsCancelled checks if an error is a cancellation error
func IsCancelled(err error) bool {
	return errors.Is(err, ErrCancelled)
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrResourceNotFound) || errors.Is(err, ErrClusterNotFound)
}

// IsConnectionError checks if an error is a connection error
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrConnectionFailed)
}

// IsPermissionError checks if an error is a permission error
func IsPermissionError(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// RetryableError wraps an error to indicate it should be retried
type RetryableError struct {
	Err        error
	RetryAfter int // seconds
}

// Error implements the error interface
func (r *RetryableError) Error() string {
	if r.RetryAfter > 0 {
		return fmt.Sprintf("retryable error (retry after %ds): %v", r.RetryAfter, r.Err)
	}
	return fmt.Sprintf("retryable error: %v", r.Err)
}

// Unwrap returns the wrapped error
func (r *RetryableError) Unwrap() error {
	return r.Err
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	var retryErr *RetryableError
	return errors.As(err, &retryErr)
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, retryAfter int) *RetryableError {
	return &RetryableError{
		Err:        err,
		RetryAfter: retryAfter,
	}
}

// FriendlyError converts technical errors to user-friendly messages
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}

	// Check for known error types
	switch {
	case IsTimeout(err):
		return "Operation timed out. Please try again or increase the timeout value with --timeout flag."
	case IsCancelled(err):
		return "Operation was cancelled."
	case IsNotFound(err):
		return "Resource not found. Please check the cluster name or resource identifier."
	case IsConnectionError(err):
		return "Failed to connect to cluster. Please check your kubeconfig and network connectivity."
	case IsPermissionError(err):
		return "Permission denied. Please check your cluster credentials and RBAC permissions."
	case errors.Is(err, ErrInvalidConfig):
		return "Invalid configuration. Please check your config file and command-line flags."
	case errors.Is(err, ErrInvalidResource):
		return "Invalid resource specification. Please check your resource definition."
	case errors.Is(err, ErrAlreadyExists):
		return "Resource already exists. Use a different name or delete the existing resource first."
	default:
		// Return the original error message for unknown errors
		return err.Error()
	}
}

// CombineErrors combines multiple errors into a single error
// Returns nil if all errors are nil
func CombineErrors(errors ...error) error {
	m := NewMultiError(errors)
	return m.ErrorOrNil()
}

// WrapErrorf wraps an error with a formatted message
func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// ErrorWithContext adds context to an error message
type ErrorWithContext struct {
	Err     error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *ErrorWithContext) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Err.Error())
	if len(e.Context) > 0 {
		sb.WriteString(" (")
		first := true
		for k, v := range e.Context {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s: %v", k, v))
			first = false
		}
		sb.WriteString(")")
	}
	return sb.String()
}

// Unwrap returns the wrapped error
func (e *ErrorWithContext) Unwrap() error {
	return e.Err
}

// AddContext adds context information to an error
func AddContext(err error, key string, value interface{}) error {
	if err == nil {
		return nil
	}

	// If already an ErrorWithContext, add to existing context
	var ctxErr *ErrorWithContext
	if errors.As(err, &ctxErr) {
		ctxErr.Context[key] = value
		return ctxErr
	}

	// Create new ErrorWithContext
	return &ErrorWithContext{
		Err:     err,
		Context: map[string]interface{}{key: value},
	}
}
