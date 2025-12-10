package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Task represents a unit of work to be executed by the worker pool
// Each task is associated with a specific cluster and contains the execution logic
type Task struct {
	// ClusterName identifies which cluster this task targets
	ClusterName string

	// Execute is the function to run for this task
	// The client parameter is the cluster client (typically *cluster.Client)
	// Returns the result data and any error encountered
	Execute func(ctx context.Context, client interface{}) (interface{}, error)
}

// Result represents the outcome of executing a task
type Result struct {
	// ClusterName identifies which cluster this result is from
	ClusterName string

	// Data contains the successful result data (nil if error occurred)
	Data interface{}

	// Error contains any error that occurred during execution (nil if successful)
	Error error

	// Duration is how long the task took to execute
	Duration time.Duration
}

// Pool manages a pool of workers that execute tasks concurrently
// It provides bounded concurrency, graceful shutdown, and progress reporting
type Pool struct {
	// workers is the number of concurrent workers
	workers int

	// tasks is the queue of tasks to execute
	tasks []Task

	// mu protects the tasks slice and state flags
	mu sync.Mutex

	// logger for structured logging
	logger *slog.Logger

	// shutdown indicates if the pool is shutting down
	shutdown atomic.Bool

	// running indicates if the pool is currently executing
	running atomic.Bool
}

// NewPool creates a new worker pool with the specified number of workers
// workers must be > 0, otherwise it defaults to 1
func NewPool(workers int, logger *slog.Logger) *Pool {
	if workers <= 0 {
		workers = 1
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Pool{
		workers: workers,
		tasks:   make([]Task, 0),
		logger:  logger,
	}
}

// Submit adds a task to the pool's queue
// Returns an error if the pool is shutting down or already running
func (p *Pool) Submit(task Task) error {
	if p.shutdown.Load() {
		return fmt.Errorf("pool is shutting down, cannot submit new tasks")
	}

	if p.running.Load() {
		return fmt.Errorf("pool is running, cannot submit new tasks")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if task.ClusterName == "" {
		return fmt.Errorf("task must have a cluster name")
	}

	if task.Execute == nil {
		return fmt.Errorf("task must have an execute function")
	}

	p.tasks = append(p.tasks, task)
	p.logger.Debug("task submitted", "cluster", task.ClusterName, "total_tasks", len(p.tasks))

	return nil
}

// Execute runs all submitted tasks using the worker pool pattern
// It creates a bounded number of worker goroutines that process tasks concurrently
// Returns a slice of results, one for each task (order may not match submission order)
func (p *Pool) Execute(ctx context.Context) []Result {
	return p.ExecuteWithProgress(ctx, nil)
}

// ExecuteWithProgress runs all tasks with progress reporting
// The progressFn callback is called after each task completes with (completed, total) counts
// Progress updates are safe to use for UI updates or logging
func (p *Pool) ExecuteWithProgress(ctx context.Context, progressFn func(completed, total int)) []Result {
	if !p.running.CompareAndSwap(false, true) {
		p.logger.Error("pool is already running")
		return []Result{}
	}
	defer p.running.Store(false)

	p.mu.Lock()
	taskCount := len(p.tasks)
	if taskCount == 0 {
		p.mu.Unlock()
		p.logger.Debug("no tasks to execute")
		return []Result{}
	}

	// Create a copy of tasks to avoid holding the lock during execution
	tasksCopy := make([]Task, len(p.tasks))
	copy(tasksCopy, p.tasks)
	p.mu.Unlock()

	p.logger.Info("starting task execution",
		"workers", p.workers,
		"tasks", taskCount)

	startTime := time.Now()

	// Create channels for task distribution and result collection
	// Buffer size = task count to avoid blocking
	taskChan := make(chan taskWithIndex, taskCount)
	resultChan := make(chan resultWithIndex, taskCount)

	// Completed counter for progress reporting
	var completed atomic.Int32

	// Start worker goroutines
	var wg sync.WaitGroup
	workerCount := p.workers
	if workerCount > taskCount {
		// Don't create more workers than tasks
		workerCount = taskCount
	}

	p.logger.Debug("starting workers", "count", workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go p.worker(ctx, i, taskChan, resultChan, &wg, &completed, taskCount, progressFn)
	}

	// Send all tasks to the task channel
	for i, task := range tasksCopy {
		select {
		case taskChan <- taskWithIndex{task: task, index: i}:
		case <-ctx.Done():
			p.logger.Warn("context cancelled while queuing tasks")
			close(taskChan)
			goto waitForWorkers
		}
	}
	close(taskChan)

waitForWorkers:
	// Wait for all workers to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	results := make([]Result, taskCount)
	resultsReceived := 0

	for res := range resultChan {
		if res.index >= 0 && res.index < taskCount {
			results[res.index] = res.result
			resultsReceived++
		}
	}

	// For any tasks that didn't complete (e.g., context cancelled before execution)
	// create error results
	for i := range results {
		if results[i].ClusterName == "" {
			results[i] = Result{
				ClusterName: tasksCopy[i].ClusterName,
				Error:       fmt.Errorf("task not executed: %w", ctx.Err()),
				Duration:    0,
			}
		}
	}

	totalDuration := time.Since(startTime)
	successCount := CountSuccessful(results)
	failureCount := taskCount - successCount

	p.logger.Info("task execution completed",
		"total", taskCount,
		"successful", successCount,
		"failed", failureCount,
		"duration", totalDuration)

	return results
}

// worker is the worker goroutine that processes tasks from the task channel
func (p *Pool) worker(
	ctx context.Context,
	workerID int,
	taskChan <-chan taskWithIndex,
	resultChan chan<- resultWithIndex,
	wg *sync.WaitGroup,
	completed *atomic.Int32,
	total int,
	progressFn func(completed, total int),
) {
	defer wg.Done()

	p.logger.Debug("worker started", "worker_id", workerID)

	for {
		select {
		case <-ctx.Done():
			p.logger.Debug("worker stopping due to context cancellation", "worker_id", workerID)
			return

		case taskItem, ok := <-taskChan:
			if !ok {
				// Channel closed, no more tasks
				p.logger.Debug("worker finished (no more tasks)", "worker_id", workerID)
				return
			}

			// Execute the task
			result := p.executeTask(ctx, taskItem.task)

			// Send result
			select {
			case resultChan <- resultWithIndex{result: result, index: taskItem.index}:
			case <-ctx.Done():
				p.logger.Warn("context cancelled while sending result",
					"worker_id", workerID,
					"cluster", taskItem.task.ClusterName)
				return
			}

			// Update progress
			completedCount := completed.Add(1)
			p.logger.Debug("task completed",
				"worker_id", workerID,
				"cluster", taskItem.task.ClusterName,
				"success", result.Error == nil,
				"duration", result.Duration,
				"progress", fmt.Sprintf("%d/%d", completedCount, total))

			// Call progress callback if provided
			if progressFn != nil {
				progressFn(int(completedCount), total)
			}
		}
	}
}

// executeTask executes a single task and returns the result
func (p *Pool) executeTask(ctx context.Context, task Task) Result {
	startTime := time.Now()

	p.logger.Debug("executing task", "cluster", task.ClusterName)

	// Check context before execution
	select {
	case <-ctx.Done():
		return Result{
			ClusterName: task.ClusterName,
			Error:       fmt.Errorf("task cancelled before execution: %w", ctx.Err()),
			Duration:    time.Since(startTime),
		}
	default:
	}

	// Execute the task
	// Note: We pass nil as the client here. In real usage, the Execute function
	// should have the client bound via closure or the pool should maintain a client map
	data, err := task.Execute(ctx, nil)

	duration := time.Since(startTime)

	result := Result{
		ClusterName: task.ClusterName,
		Data:        data,
		Error:       err,
		Duration:    duration,
	}

	if err != nil {
		p.logger.Warn("task failed",
			"cluster", task.ClusterName,
			"error", err,
			"duration", duration)
	} else {
		p.logger.Debug("task succeeded",
			"cluster", task.ClusterName,
			"duration", duration)
	}

	return result
}

// Shutdown gracefully shuts down the pool
// It stops accepting new tasks and waits for in-progress tasks to complete
// The context timeout controls how long to wait for tasks to finish
func (p *Pool) Shutdown(ctx context.Context) error {
	if !p.shutdown.CompareAndSwap(false, true) {
		return fmt.Errorf("pool already shut down")
	}

	p.logger.Info("shutting down worker pool")

	// If the pool is currently running, wait for it to finish
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		p.logger.Debug("waiting for pool to finish", "deadline", deadline)
	}

	// Poll until the pool is no longer running or context times out
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for p.running.Load() {
		select {
		case <-ctx.Done():
			return fmt.Errorf("shutdown timeout: %w", ctx.Err())
		case <-ticker.C:
			// Continue polling
		}
	}

	p.logger.Info("worker pool shut down successfully")
	return nil
}

// IsShutdown returns true if the pool has been shut down
func (p *Pool) IsShutdown() bool {
	return p.shutdown.Load()
}

// IsRunning returns true if the pool is currently executing tasks
func (p *Pool) IsRunning() bool {
	return p.running.Load()
}

// TaskCount returns the number of tasks currently queued
func (p *Pool) TaskCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.tasks)
}

// WorkerCount returns the number of workers in the pool
func (p *Pool) WorkerCount() int {
	return p.workers
}

// taskWithIndex pairs a task with its original index for result ordering
type taskWithIndex struct {
	task  Task
	index int
}

// resultWithIndex pairs a result with its original task index
type resultWithIndex struct {
	result Result
	index  int
}
