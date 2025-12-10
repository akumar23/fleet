package util

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestClusterError(t *testing.T) {
	baseErr := errors.New("connection failed")
	clusterErr := WrapClusterError("test-cluster", baseErr)

	if clusterErr == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := `cluster "test-cluster": connection failed`
	if clusterErr.Error() != expectedMsg {
		t.Errorf("expected %q, got %q", expectedMsg, clusterErr.Error())
	}

	// Test unwrapping
	if !errors.Is(clusterErr, baseErr) {
		t.Error("expected cluster error to wrap base error")
	}

	// Test nil wrapping
	nilErr := WrapClusterError("test", nil)
	if nilErr != nil {
		t.Errorf("expected nil, got %v", nilErr)
	}
}

func TestMultiError(t *testing.T) {
	t.Run("empty multi-error", func(t *testing.T) {
		m := &MultiError{}
		if m.ErrorOrNil() != nil {
			t.Error("expected nil for empty multi-error")
		}
	})

	t.Run("single error", func(t *testing.T) {
		err := errors.New("test error")
		m := NewMultiError([]error{err})

		if m.Error() != "test error" {
			t.Errorf("expected %q, got %q", "test error", m.Error())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errors := []error{
			errors.New("error 1"),
			errors.New("error 2"),
			errors.New("error 3"),
		}
		m := NewMultiError(errors)

		msg := m.Error()
		if !strings.Contains(msg, "3 errors occurred") {
			t.Errorf("expected message to contain '3 errors occurred', got %q", msg)
		}
		if !strings.Contains(msg, "error 1") {
			t.Errorf("expected message to contain 'error 1', got %q", msg)
		}
	})

	t.Run("filtering nil errors", func(t *testing.T) {
		errors := []error{
			errors.New("error 1"),
			nil,
			errors.New("error 2"),
			nil,
		}
		m := NewMultiError(errors)

		if len(m.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(m.Errors))
		}
	})

	t.Run("add errors", func(t *testing.T) {
		m := &MultiError{}
		m.Add(errors.New("error 1"))
		m.Add(nil) // Should not be added
		m.Add(errors.New("error 2"))

		if len(m.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(m.Errors))
		}
	})

	t.Run("many errors truncation", func(t *testing.T) {
		m := &MultiError{}
		for i := 0; i < 20; i++ {
			m.Add(fmt.Errorf("error %d", i+1))
		}

		msg := m.Error()
		if !strings.Contains(msg, "and 10 more errors") {
			t.Errorf("expected truncation message, got %q", msg)
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		err := NewValidationError("port", 9999999, "port out of range")
		expectedMsg := `validation failed for field "port" (value: 9999999): port out of range`
		if err.Error() != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("without value", func(t *testing.T) {
		err := NewValidationError("name", nil, "name is required")
		expectedMsg := `validation failed for field "name": name is required`
		if err.Error() != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestRetryableError(t *testing.T) {
	baseErr := errors.New("temporary failure")

	t.Run("with retry after", func(t *testing.T) {
		err := NewRetryableError(baseErr, 30)
		if !strings.Contains(err.Error(), "retry after 30s") {
			t.Errorf("expected retry after message, got %q", err.Error())
		}

		if !IsRetryable(err) {
			t.Error("expected error to be retryable")
		}

		if !errors.Is(err, baseErr) {
			t.Error("expected error to wrap base error")
		}
	})

	t.Run("without retry after", func(t *testing.T) {
		err := NewRetryableError(baseErr, 0)
		if !strings.Contains(err.Error(), "retryable error") {
			t.Errorf("expected retryable error message, got %q", err.Error())
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		err := errors.New("permanent failure")
		if IsRetryable(err) {
			t.Error("expected error to not be retryable")
		}
	})
}

func TestErrorCheckers(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checker  func(error) bool
		expected bool
	}{
		{
			name:     "timeout error",
			err:      ErrTimeout,
			checker:  IsTimeout,
			expected: true,
		},
		{
			name:     "wrapped timeout error",
			err:      fmt.Errorf("operation failed: %w", ErrTimeout),
			checker:  IsTimeout,
			expected: true,
		},
		{
			name:     "cancelled error",
			err:      ErrCancelled,
			checker:  IsCancelled,
			expected: true,
		},
		{
			name:     "resource not found",
			err:      ErrResourceNotFound,
			checker:  IsNotFound,
			expected: true,
		},
		{
			name:     "cluster not found",
			err:      ErrClusterNotFound,
			checker:  IsNotFound,
			expected: true,
		},
		{
			name:     "connection error",
			err:      ErrConnectionFailed,
			checker:  IsConnectionError,
			expected: true,
		},
		{
			name:     "permission error",
			err:      ErrPermissionDenied,
			checker:  IsPermissionError,
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("something else"),
			checker:  IsTimeout,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checker(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFriendlyError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "nil error",
			err:      nil,
			contains: "",
		},
		{
			name:     "timeout error",
			err:      ErrTimeout,
			contains: "timed out",
		},
		{
			name:     "cancelled error",
			err:      ErrCancelled,
			contains: "cancelled",
		},
		{
			name:     "not found error",
			err:      ErrResourceNotFound,
			contains: "not found",
		},
		{
			name:     "connection error",
			err:      ErrConnectionFailed,
			contains: "connect to cluster",
		},
		{
			name:     "permission error",
			err:      ErrPermissionDenied,
			contains: "Permission denied",
		},
		{
			name:     "invalid config",
			err:      ErrInvalidConfig,
			contains: "Invalid configuration",
		},
		{
			name:     "invalid resource",
			err:      ErrInvalidResource,
			contains: "Invalid resource",
		},
		{
			name:     "already exists",
			err:      ErrAlreadyExists,
			contains: "already exists",
		},
		{
			name:     "unknown error",
			err:      errors.New("custom error message"),
			contains: "custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := FriendlyError(tt.err)
			if tt.contains == "" {
				if msg != "" {
					t.Errorf("expected empty string, got %q", msg)
				}
				return
			}

			if !strings.Contains(msg, tt.contains) {
				t.Errorf("expected message to contain %q, got %q", tt.contains, msg)
			}
		})
	}
}

func TestCombineErrors(t *testing.T) {
	t.Run("all nil errors", func(t *testing.T) {
		err := CombineErrors(nil, nil, nil)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("mixed nil and non-nil errors", func(t *testing.T) {
		err := CombineErrors(
			errors.New("error 1"),
			nil,
			errors.New("error 2"),
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		msg := err.Error()
		if !strings.Contains(msg, "error 1") || !strings.Contains(msg, "error 2") {
			t.Errorf("expected combined error message, got %q", msg)
		}
	})
}

func TestWrapErrorf(t *testing.T) {
	baseErr := errors.New("base error")

	t.Run("wrap error", func(t *testing.T) {
		wrapped := WrapErrorf(baseErr, "failed to process file %q", "test.txt")
		expectedMsg := `failed to process file "test.txt": base error`
		if wrapped.Error() != expectedMsg {
			t.Errorf("expected %q, got %q", expectedMsg, wrapped.Error())
		}

		if !errors.Is(wrapped, baseErr) {
			t.Error("expected wrapped error to contain base error")
		}
	})

	t.Run("wrap nil error", func(t *testing.T) {
		wrapped := WrapErrorf(nil, "this should be nil")
		if wrapped != nil {
			t.Errorf("expected nil, got %v", wrapped)
		}
	})
}

func TestErrorWithContext(t *testing.T) {
	baseErr := errors.New("base error")

	t.Run("add single context", func(t *testing.T) {
		err := AddContext(baseErr, "cluster", "test-cluster")
		msg := err.Error()

		if !strings.Contains(msg, "base error") {
			t.Errorf("expected base error in message, got %q", msg)
		}
		if !strings.Contains(msg, "cluster: test-cluster") {
			t.Errorf("expected context in message, got %q", msg)
		}

		if !errors.Is(err, baseErr) {
			t.Error("expected error to wrap base error")
		}
	})

	t.Run("add multiple contexts", func(t *testing.T) {
		err := AddContext(baseErr, "cluster", "test-cluster")
		err = AddContext(err, "namespace", "default")

		msg := err.Error()
		if !strings.Contains(msg, "cluster: test-cluster") {
			t.Errorf("expected cluster context, got %q", msg)
		}
		if !strings.Contains(msg, "namespace: default") {
			t.Errorf("expected namespace context, got %q", msg)
		}
	})

	t.Run("add context to nil error", func(t *testing.T) {
		err := AddContext(nil, "cluster", "test")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestClusterErrorUnwrap(t *testing.T) {
	baseErr := errors.New("connection timeout")
	clusterErr := &ClusterError{
		ClusterName: "prod-cluster",
		Err:         baseErr,
	}

	// Test errors.Is
	if !errors.Is(clusterErr, baseErr) {
		t.Error("errors.Is should find wrapped error")
	}

	// Test errors.As
	var ce *ClusterError
	if !errors.As(clusterErr, &ce) {
		t.Error("errors.As should find ClusterError")
	}
	if ce.ClusterName != "prod-cluster" {
		t.Errorf("expected cluster name %q, got %q", "prod-cluster", ce.ClusterName)
	}
}

func TestMultiErrorUnwrap(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	m := NewMultiError([]error{err1, err2, err3})

	// Check if MultiError implements Unwrap() []error
	unwrapped := m.Unwrap()
	if len(unwrapped) != 3 {
		t.Errorf("expected 3 unwrapped errors, got %d", len(unwrapped))
	}

	// Verify errors.Is works with MultiError
	if !errors.Is(m, err1) {
		t.Error("errors.Is should find err1 in MultiError")
	}
	if !errors.Is(m, err2) {
		t.Error("errors.Is should find err2 in MultiError")
	}
	if !errors.Is(m, err3) {
		t.Error("errors.Is should find err3 in MultiError")
	}
}
