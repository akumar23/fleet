package util

import (
	"context"
	"syscall"
	"testing"
	"time"
)

func TestSetupSignalHandler(t *testing.T) {
	ctx := SetupSignalHandler()

	// Verify context is not cancelled initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Expected behavior
	}

	// Send SIGTERM to trigger context cancellation
	// Note: This test sends a signal to the current process
	// which will trigger the signal handler
	go func() {
		time.Sleep(10 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()

	// Wait for context cancellation with timeout
	select {
	case <-ctx.Done():
		// Expected: context should be cancelled after signal
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", ctx.Err())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Context was not cancelled after SIGTERM")
	}
}
