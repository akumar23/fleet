package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

// BenchmarkPool_Submit benchmarks task submission performance
func BenchmarkPool_Submit(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	pool := NewPool(10, logger)

	task := Task{
		ClusterName: "benchmark-cluster",
		Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
			return "done", nil
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Submit(task)
	}
}

// BenchmarkPool_Execute benchmarks pool execution with different worker counts
func BenchmarkPool_Execute(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers_%d", workers), func(b *testing.B) {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				pool := NewPool(workers, logger)

				// Submit 100 tasks
				for j := 0; j < 100; j++ {
					pool.Submit(Task{
						ClusterName: fmt.Sprintf("cluster-%d", j),
						Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
							// Simulate minimal work
							time.Sleep(100 * time.Microsecond)
							return "done", nil
						},
					})
				}

				b.StartTimer()
				pool.Execute(context.Background())
			}
		})
	}
}

// BenchmarkPool_ExecuteParallel benchmarks concurrent pool usage
func BenchmarkPool_ExecuteParallel(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool := NewPool(4, logger)

			for i := 0; i < 10; i++ {
				pool.Submit(Task{
					ClusterName: fmt.Sprintf("cluster-%d", i),
					Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
						return "done", nil
					},
				})
			}

			pool.Execute(context.Background())
		}
	})
}

// BenchmarkResult_Filtering benchmarks result filtering operations
func BenchmarkResult_Filtering(b *testing.B) {
	// Create a large result set
	results := make([]Result, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = Result{
			ClusterName: fmt.Sprintf("cluster-%d", i),
			Data:        "test-data",
			Duration:    time.Duration(i) * time.Millisecond,
		}

		// Make half of them fail
		if i%2 == 0 {
			results[i].Error = fmt.Errorf("error %d", i)
		}
	}

	b.Run("FilterSuccessful", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			FilterSuccessful(results)
		}
	})

	b.Run("FilterFailed", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			FilterFailed(results)
		}
	})

	b.Run("CountSuccessful", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			CountSuccessful(results)
		}
	})

	b.Run("Summarize", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			Summarize(results)
		}
	})

	b.Run("GroupByCluster", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			GroupByCluster(results)
		}
	})
}

// BenchmarkPool_ProgressReporting benchmarks progress callback overhead
func BenchmarkPool_ProgressReporting(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	b.Run("WithProgress", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			pool := NewPool(4, logger)
			for j := 0; j < 50; j++ {
				pool.Submit(Task{
					ClusterName: fmt.Sprintf("cluster-%d", j),
					Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
						return "done", nil
					},
				})
			}

			b.StartTimer()
			pool.ExecuteWithProgress(context.Background(), func(completed, total int) {
				// Minimal callback
			})
		}
	})

	b.Run("WithoutProgress", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			pool := NewPool(4, logger)
			for j := 0; j < 50; j++ {
				pool.Submit(Task{
					ClusterName: fmt.Sprintf("cluster-%d", j),
					Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
						return "done", nil
					},
				})
			}

			b.StartTimer()
			pool.Execute(context.Background())
		}
	})
}

// BenchmarkDurationFunctions benchmarks duration calculation functions
func BenchmarkDurationFunctions(b *testing.B) {
	results := make([]Result, 1000)
	for i := 0; i < 1000; i++ {
		results[i] = Result{
			ClusterName: fmt.Sprintf("cluster-%d", i),
			Duration:    time.Duration(i) * time.Millisecond,
		}
	}

	b.Run("AverageDuration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			AverageDuration(results)
		}
	})

	b.Run("MaxDuration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			MaxDuration(results)
		}
	})

	b.Run("MinDuration", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			MinDuration(results)
		}
	})
}

// BenchmarkPool_Shutdown benchmarks graceful shutdown performance
func BenchmarkPool_Shutdown(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool := NewPool(4, logger)

		// Submit tasks
		for j := 0; j < 20; j++ {
			pool.Submit(Task{
				ClusterName: fmt.Sprintf("cluster-%d", j),
				Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
					time.Sleep(time.Millisecond)
					return "done", nil
				},
			})
		}

		// Start execution
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			pool.Execute(ctx)
			close(done)
		}()

		// Let some tasks start
		time.Sleep(2 * time.Millisecond)

		b.StartTimer()
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
		pool.Shutdown(shutdownCtx)
		shutdownCancel()
		<-done
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	b.Run("PoolCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			NewPool(10, logger)
		}
	})

	b.Run("TaskSubmission", func(b *testing.B) {
		pool := NewPool(10, logger)
		task := Task{
			ClusterName: "test",
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				return nil, nil
			},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pool.Submit(task)
		}
	})

	b.Run("ResultCollection", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			results := make([]Result, 100)
			for j := 0; j < 100; j++ {
				results[j] = Result{
					ClusterName: fmt.Sprintf("cluster-%d", j),
					Data:        "test",
					Duration:    time.Millisecond,
				}
			}
			_ = results
		}
	})
}

// BenchmarkContextPropagation benchmarks context cancellation propagation
func BenchmarkContextPropagation(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		pool := NewPool(8, logger)

		for j := 0; j < 100; j++ {
			pool.Submit(Task{
				ClusterName: fmt.Sprintf("cluster-%d", j),
				Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(10 * time.Millisecond):
						return "done", nil
					}
				},
			})
		}

		ctx, cancel := context.WithCancel(context.Background())

		b.StartTimer()
		// Cancel immediately to test propagation
		cancel()
		pool.Execute(ctx)
	}
}
