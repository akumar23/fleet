package executor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	tests := []struct {
		name            string
		workers         int
		expectedWorkers int
	}{
		{
			name:            "positive workers",
			workers:         5,
			expectedWorkers: 5,
		},
		{
			name:            "zero workers defaults to 1",
			workers:         0,
			expectedWorkers: 1,
		},
		{
			name:            "negative workers defaults to 1",
			workers:         -5,
			expectedWorkers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.workers, nil)
			if pool == nil {
				t.Fatal("NewPool returned nil")
			}

			if pool.WorkerCount() != tt.expectedWorkers {
				t.Errorf("expected %d workers, got %d", tt.expectedWorkers, pool.WorkerCount())
			}

			if pool.TaskCount() != 0 {
				t.Errorf("expected 0 tasks initially, got %d", pool.TaskCount())
			}

			if pool.IsShutdown() {
				t.Error("new pool should not be shut down")
			}

			if pool.IsRunning() {
				t.Error("new pool should not be running")
			}
		})
	}
}

func TestPool_Submit(t *testing.T) {
	tests := []struct {
		name        string
		task        Task
		wantErr     bool
		errContains string
	}{
		{
			name: "valid task",
			task: Task{
				ClusterName: "test-cluster",
				Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
					return "success", nil
				},
			},
			wantErr: false,
		},
		{
			name: "missing cluster name",
			task: Task{
				ClusterName: "",
				Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
					return nil, nil
				},
			},
			wantErr:     true,
			errContains: "cluster name",
		},
		{
			name: "missing execute function",
			task: Task{
				ClusterName: "test-cluster",
				Execute:     nil,
			},
			wantErr:     true,
			errContains: "execute function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(1, slog.Default())
			err := pool.Submit(tt.task)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if pool.TaskCount() != 1 {
					t.Errorf("expected 1 task, got %d", pool.TaskCount())
				}
			}
		})
	}
}

func TestPool_Submit_WhileRunning(t *testing.T) {
	pool := NewPool(1, slog.Default())

	// Submit a long-running task
	err := pool.Submit(Task{
		ClusterName: "cluster1",
		Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return "done", nil
		},
	})
	if err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	// Start execution in background
	ctx := context.Background()
	go pool.Execute(ctx)

	// Wait a bit for execution to start
	time.Sleep(10 * time.Millisecond)

	// Try to submit another task while running
	err = pool.Submit(Task{
		ClusterName: "cluster2",
		Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
			return "done", nil
		},
	})

	if err == nil {
		t.Error("expected error when submitting while running")
	}
	if !contains(err.Error(), "running") {
		t.Errorf("expected error about running, got: %v", err)
	}
}

func TestPool_Submit_AfterShutdown(t *testing.T) {
	pool := NewPool(1, slog.Default())

	ctx := context.Background()
	err := pool.Shutdown(ctx)
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	err = pool.Submit(Task{
		ClusterName: "cluster1",
		Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
			return "done", nil
		},
	})

	if err == nil {
		t.Error("expected error when submitting after shutdown")
	}
	if !contains(err.Error(), "shutting down") {
		t.Errorf("expected error about shutdown, got: %v", err)
	}
}

func TestPool_Execute(t *testing.T) {
	tests := []struct {
		name          string
		workers       int
		tasks         []Task
		expectedCount int
		checkResults  func(t *testing.T, results []Result)
	}{
		{
			name:    "single task",
			workers: 1,
			tasks: []Task{
				{
					ClusterName: "cluster1",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result1", nil
					},
				},
			},
			expectedCount: 1,
			checkResults: func(t *testing.T, results []Result) {
				if results[0].Error != nil {
					t.Errorf("expected no error, got %v", results[0].Error)
				}
				if results[0].ClusterName != "cluster1" {
					t.Errorf("expected cluster1, got %s", results[0].ClusterName)
				}
				if results[0].Data != "result1" {
					t.Errorf("expected result1, got %v", results[0].Data)
				}
			},
		},
		{
			name:    "multiple tasks fewer workers",
			workers: 2,
			tasks: []Task{
				{
					ClusterName: "cluster1",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result1", nil
					},
				},
				{
					ClusterName: "cluster2",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result2", nil
					},
				},
				{
					ClusterName: "cluster3",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result3", nil
					},
				},
				{
					ClusterName: "cluster4",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result4", nil
					},
				},
			},
			expectedCount: 4,
			checkResults: func(t *testing.T, results []Result) {
				successful := CountSuccessful(results)
				if successful != 4 {
					t.Errorf("expected 4 successful results, got %d", successful)
				}
			},
		},
		{
			name:    "more workers than tasks",
			workers: 10,
			tasks: []Task{
				{
					ClusterName: "cluster1",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result1", nil
					},
				},
				{
					ClusterName: "cluster2",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "result2", nil
					},
				},
			},
			expectedCount: 2,
			checkResults: func(t *testing.T, results []Result) {
				if len(results) != 2 {
					t.Errorf("expected 2 results, got %d", len(results))
				}
			},
		},
		{
			name:    "mixed success and failure",
			workers: 2,
			tasks: []Task{
				{
					ClusterName: "cluster1",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "success", nil
					},
				},
				{
					ClusterName: "cluster2",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return nil, errors.New("task failed")
					},
				},
				{
					ClusterName: "cluster3",
					Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
						return "success", nil
					},
				},
			},
			expectedCount: 3,
			checkResults: func(t *testing.T, results []Result) {
				successful := CountSuccessful(results)
				failed := CountFailed(results)
				if successful != 2 {
					t.Errorf("expected 2 successful, got %d", successful)
				}
				if failed != 1 {
					t.Errorf("expected 1 failed, got %d", failed)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.workers, slog.Default())

			// Submit all tasks
			for _, task := range tt.tasks {
				if err := pool.Submit(task); err != nil {
					t.Fatalf("failed to submit task: %v", err)
				}
			}

			// Execute
			ctx := context.Background()
			results := pool.Execute(ctx)

			// Check result count
			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}

			// Run custom checks
			if tt.checkResults != nil {
				tt.checkResults(t, results)
			}

			// Verify all results have durations
			for i, r := range results {
				if r.Duration == 0 {
					t.Errorf("result %d has zero duration", i)
				}
			}
		})
	}
}

func TestPool_Execute_Empty(t *testing.T) {
	pool := NewPool(5, slog.Default())

	ctx := context.Background()
	results := pool.Execute(ctx)

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty pool, got %d", len(results))
	}
}

func TestPool_Execute_ContextCancellation(t *testing.T) {
	pool := NewPool(2, slog.Default())

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Submit tasks that check for cancellation
	for i := 0; i < 5; i++ {
		clusterName := fmt.Sprintf("cluster%d", i+1)
		err := pool.Submit(Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				// Simulate work and check for cancellation
				select {
				case <-time.After(100 * time.Millisecond):
					return "completed", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	// Cancel context shortly after starting
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	results := pool.Execute(ctx)

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// At least some tasks should have been cancelled
	cancelled := 0
	for _, r := range results {
		if r.Error != nil && errors.Is(r.Error, context.Canceled) {
			cancelled++
		}
	}

	if cancelled == 0 {
		t.Error("expected at least some tasks to be cancelled")
	}
}

func TestPool_ExecuteWithProgress(t *testing.T) {
	pool := NewPool(2, slog.Default())

	// Submit tasks
	taskCount := 5
	for i := 0; i < taskCount; i++ {
		clusterName := fmt.Sprintf("cluster%d", i+1)
		err := pool.Submit(Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return "done", nil
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	// Track progress
	var progressCalls atomic.Int32
	var maxCompleted atomic.Int32
	var progressMu sync.Mutex
	progressUpdates := make([]struct{ completed, total int }, 0)

	progressFn := func(completed, total int) {
		progressCalls.Add(1)

		// Track max completed
		for {
			current := maxCompleted.Load()
			if int32(completed) <= current {
				break
			}
			if maxCompleted.CompareAndSwap(current, int32(completed)) {
				break
			}
		}

		progressMu.Lock()
		progressUpdates = append(progressUpdates, struct{ completed, total int }{completed, total})
		progressMu.Unlock()
	}

	ctx := context.Background()
	results := pool.ExecuteWithProgress(ctx, progressFn)

	// Verify results
	if len(results) != taskCount {
		t.Errorf("expected %d results, got %d", taskCount, len(results))
	}

	// Verify progress was called
	calls := progressCalls.Load()
	if calls != int32(taskCount) {
		t.Errorf("expected %d progress calls, got %d", taskCount, calls)
	}

	// Verify we reached 100% completion
	if maxCompleted.Load() != int32(taskCount) {
		t.Errorf("expected max completed to be %d, got %d", taskCount, maxCompleted.Load())
	}

	// Verify total is always correct
	progressMu.Lock()
	for i, update := range progressUpdates {
		if update.total != taskCount {
			t.Errorf("progress update %d: expected total %d, got %d", i, taskCount, update.total)
		}
		if update.completed < 1 || update.completed > taskCount {
			t.Errorf("progress update %d: completed %d out of range [1, %d]", i, update.completed, taskCount)
		}
	}
	progressMu.Unlock()
}

func TestPool_PartialFailures(t *testing.T) {
	pool := NewPool(3, slog.Default())

	// Submit mix of successful and failing tasks
	tasks := []struct {
		cluster    string
		shouldFail bool
	}{
		{"cluster1", false},
		{"cluster2", true},
		{"cluster3", false},
		{"cluster4", true},
		{"cluster5", false},
		{"cluster6", false},
	}

	for _, tc := range tasks {
		shouldFail := tc.shouldFail
		err := pool.Submit(Task{
			ClusterName: tc.cluster,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				if shouldFail {
					return nil, errors.New("simulated failure")
				}
				return "success", nil
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	ctx := context.Background()
	results := pool.Execute(ctx)

	// Verify all tasks completed
	if len(results) != len(tasks) {
		t.Errorf("expected %d results, got %d", len(tasks), len(results))
	}

	// Count successes and failures
	successful := CountSuccessful(results)
	failed := CountFailed(results)

	expectedSuccess := 4
	expectedFailed := 2

	if successful != expectedSuccess {
		t.Errorf("expected %d successful, got %d", expectedSuccess, successful)
	}

	if failed != expectedFailed {
		t.Errorf("expected %d failed, got %d", expectedFailed, failed)
	}

	// Verify we can filter results
	successResults := FilterSuccessful(results)
	if len(successResults) != expectedSuccess {
		t.Errorf("FilterSuccessful: expected %d, got %d", expectedSuccess, len(successResults))
	}

	failResults := FilterFailed(results)
	if len(failResults) != expectedFailed {
		t.Errorf("FilterFailed: expected %d, got %d", expectedFailed, len(failResults))
	}
}

func TestPool_GracefulShutdown(t *testing.T) {
	pool := NewPool(2, slog.Default())

	// Submit long-running tasks
	for i := 0; i < 3; i++ {
		clusterName := fmt.Sprintf("cluster%d", i+1)
		err := pool.Submit(Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				time.Sleep(100 * time.Millisecond)
				return "completed", nil
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	// Start execution in background
	ctx := context.Background()
	var results []Result
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		results = pool.Execute(ctx)
	}()

	// Wait a bit for execution to start
	time.Sleep(20 * time.Millisecond)

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := pool.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	// Wait for execution to complete
	wg.Wait()

	// Verify all tasks completed
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Verify pool is shut down
	if !pool.IsShutdown() {
		t.Error("pool should be shut down")
	}

	// Verify we can't submit after shutdown
	err = pool.Submit(Task{
		ClusterName: "cluster4",
		Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Error("should not be able to submit after shutdown")
	}
}

func TestPool_GracefulShutdown_Timeout(t *testing.T) {
	pool := NewPool(1, slog.Default())

	// Submit a very long task
	err := pool.Submit(Task{
		ClusterName: "cluster1",
		Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
			time.Sleep(1 * time.Second)
			return "done", nil
		},
	})
	if err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	// Start execution
	ctx := context.Background()
	go pool.Execute(ctx)

	// Wait for task to start
	time.Sleep(20 * time.Millisecond)

	// Shutdown with short timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = pool.Shutdown(shutdownCtx)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected deadline exceeded error, got: %v", err)
	}
}

func TestPool_DoubleShutdown(t *testing.T) {
	pool := NewPool(1, slog.Default())

	ctx := context.Background()
	err := pool.Shutdown(ctx)
	if err != nil {
		t.Errorf("first shutdown failed: %v", err)
	}

	// Second shutdown should return error
	err = pool.Shutdown(ctx)
	if err == nil {
		t.Error("second shutdown should return error")
	}
	if !contains(err.Error(), "already") {
		t.Errorf("expected 'already' in error, got: %v", err)
	}
}

func TestPool_ConcurrentExecution(t *testing.T) {
	// This test verifies that tasks are actually executed concurrently
	pool := NewPool(5, slog.Default())

	// Submit tasks that track their execution time
	var startTimes sync.Map
	var endTimes sync.Map

	taskCount := 10
	for i := 0; i < taskCount; i++ {
		clusterName := fmt.Sprintf("cluster%d", i+1)
		err := pool.Submit(Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, client interface{}) (interface{}, error) {
				cluster := clusterName
				startTimes.Store(cluster, time.Now())
				time.Sleep(50 * time.Millisecond)
				endTimes.Store(cluster, time.Now())
				return "done", nil
			},
		})
		if err != nil {
			t.Fatalf("failed to submit task: %v", err)
		}
	}

	ctx := context.Background()
	start := time.Now()
	results := pool.Execute(ctx)
	totalDuration := time.Since(start)

	// Verify all tasks completed
	if len(results) != taskCount {
		t.Errorf("expected %d results, got %d", taskCount, len(results))
	}

	// With 5 workers and 10 tasks of 50ms each, total time should be around 100ms
	// (two batches of 5), not 500ms (sequential)
	// Allow some overhead
	maxExpected := 200 * time.Millisecond
	if totalDuration > maxExpected {
		t.Errorf("execution took too long (%v), expected around 100ms (concurrent), not 500ms (sequential)",
			totalDuration)
	}

	// Verify at least some tasks overlapped (ran concurrently)
	overlaps := 0
	startTimes.Range(func(k1, v1 interface{}) bool {
		start1 := v1.(time.Time)
		end1, _ := endTimes.Load(k1)

		startTimes.Range(func(k2, v2 interface{}) bool {
			if k1 == k2 {
				return true
			}
			start2 := v2.(time.Time)
			end2, _ := endTimes.Load(k2)

			// Check if tasks overlapped
			if start2.Before(end1.(time.Time)) && start1.Before(end2.(time.Time)) {
				overlaps++
			}
			return true
		})
		return true
	})

	if overlaps == 0 {
		t.Error("no tasks overlapped, suggesting they didn't run concurrently")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && contains(s[1:], substr)) ||
		(len(s) >= len(substr) && s[:len(substr)] == substr))
}
