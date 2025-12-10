package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/aryankumar/fleet/internal/config"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// BenchmarkManager_Connect benchmarks connecting to multiple clusters
func BenchmarkManager_Connect(b *testing.B) {
	clusterCounts := []int{5, 10, 25, 50}

	for _, count := range clusterCounts {
		b.Run(fmt.Sprintf("clusters_%d", count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				// Create test kubeconfig
				clusters := make([]string, count)
				for j := 0; j < count; j++ {
					clusters[j] = fmt.Sprintf("cluster-%d", j)
				}
				kubeconfigPath := benchCreateKubeconfig(b, clusters)

				logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
				loader := config.NewKubeconfigLoader(kubeconfigPath)
				manager := NewManager(loader, logger)

				b.StartTimer()
				ctx := context.Background()
				manager.Connect(ctx, clusters)
				manager.Close()
			}
		})
	}
}

// BenchmarkManager_HealthCheck benchmarks health checking multiple clusters
func BenchmarkManager_HealthCheck(b *testing.B) {
	clusterCounts := []int{5, 10, 25}

	for _, count := range clusterCounts {
		b.Run(fmt.Sprintf("clusters_%d", count), func(b *testing.B) {
			// Setup
			clusters := make([]string, count)
			for j := 0; j < count; j++ {
				clusters[j] = fmt.Sprintf("cluster-%d", j)
			}
			kubeconfigPath := benchCreateKubeconfig(b, clusters)

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			loader := config.NewKubeconfigLoader(kubeconfigPath)
			manager := NewManager(loader, logger)

			ctx := context.Background()
			manager.Connect(ctx, clusters)
			defer manager.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				manager.HealthCheck(ctx)
			}
		})
	}
}

// BenchmarkManager_GetClient benchmarks retrieving individual clients
func BenchmarkManager_GetClient(b *testing.B) {
	clusters := make([]string, 50)
	for i := 0; i < 50; i++ {
		clusters[i] = fmt.Sprintf("cluster-%d", i)
	}
	kubeconfigPath := benchCreateKubeconfig(b, clusters)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := NewManager(loader, logger)

	ctx := context.Background()
	manager.Connect(ctx, clusters)
	defer manager.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clusterIdx := i % 50
		_, _ = manager.GetClient(fmt.Sprintf("cluster-%d", clusterIdx))
	}
}

// BenchmarkManager_GetAllClients benchmarks retrieving all clients
func BenchmarkManager_GetAllClients(b *testing.B) {
	clusters := make([]string, 50)
	for i := 0; i < 50; i++ {
		clusters[i] = fmt.Sprintf("cluster-%d", i)
	}
	kubeconfigPath := benchCreateKubeconfig(b, clusters)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := NewManager(loader, logger)

	ctx := context.Background()
	manager.Connect(ctx, clusters)
	defer manager.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetAllClients()
	}
}

// BenchmarkManager_ConcurrentAccess benchmarks concurrent access to manager
func BenchmarkManager_ConcurrentAccess(b *testing.B) {
	clusters := make([]string, 20)
	for i := 0; i < 20; i++ {
		clusters[i] = fmt.Sprintf("cluster-%d", i)
	}
	kubeconfigPath := benchCreateKubeconfig(b, clusters)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := NewManager(loader, logger)

	ctx := context.Background()
	manager.Connect(ctx, clusters)
	defer manager.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			// Mix of different operations
			switch idx % 4 {
			case 0:
				manager.GetAllClients()
			case 1:
				manager.GetClientNames()
			case 2:
				clusterIdx := idx % 20
				manager.GetClient(fmt.Sprintf("cluster-%d", clusterIdx))
			case 3:
				manager.Count()
			}
			idx++
		}
	})
}

// BenchmarkManager_HealthCheckWithStatus benchmarks detailed health checks
func BenchmarkManager_HealthCheckWithStatus(b *testing.B) {
	clusters := make([]string, 10)
	for i := 0; i < 10; i++ {
		clusters[i] = fmt.Sprintf("cluster-%d", i)
	}
	kubeconfigPath := benchCreateKubeconfig(b, clusters)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	loader := config.NewKubeconfigLoader(kubeconfigPath)
	manager := NewManager(loader, logger)

	ctx := context.Background()
	manager.Connect(ctx, clusters)
	defer manager.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.HealthCheckWithStatus(ctx)
	}
}

// BenchmarkHealthStatus benchmarks the HealthStatus type
func BenchmarkHealthStatus(b *testing.B) {
	b.Run("Creation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = HealthStatus{
				ClusterName:   "test-cluster",
				Healthy:       true,
				ServerVersion: "v1.28.0",
				Error:         nil,
			}
		}
	})
}

// BenchmarkManager_MemoryAllocation benchmarks memory allocation patterns
func BenchmarkManager_MemoryAllocation(b *testing.B) {
	b.Run("ManagerCreation", func(b *testing.B) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		loader := config.NewKubeconfigLoader("")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewManager(loader, logger)
		}
	})

	b.Run("GetClientNames", func(b *testing.B) {
		clusters := make([]string, 50)
		for i := 0; i < 50; i++ {
			clusters[i] = fmt.Sprintf("cluster-%d", i)
		}
		kubeconfigPath := benchCreateKubeconfig(b, clusters)

		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		loader := config.NewKubeconfigLoader(kubeconfigPath)
		manager := NewManager(loader, logger)

		ctx := context.Background()
		manager.Connect(ctx, clusters)
		defer manager.Close()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = manager.GetClientNames()
		}
	})
}

// benchCreateKubeconfig creates a temporary kubeconfig for benchmarking
func benchCreateKubeconfig(b *testing.B, clusters []string) string {
	b.Helper()

	tmpDir := b.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")

	config := api.Config{
		Clusters:       make(map[string]*api.Cluster),
		AuthInfos:      make(map[string]*api.AuthInfo),
		Contexts:       make(map[string]*api.Context),
		CurrentContext: "",
	}

	for i, clusterName := range clusters {
		config.Clusters[clusterName] = &api.Cluster{
			Server:                fmt.Sprintf("https://cluster%d.example.com:6443", i+1),
			InsecureSkipTLSVerify: true,
		}

		config.AuthInfos[clusterName] = &api.AuthInfo{
			Token: fmt.Sprintf("token-%s", clusterName),
		}

		config.Contexts[clusterName] = &api.Context{
			Cluster:   clusterName,
			AuthInfo:  clusterName,
			Namespace: "default",
		}

		if i == 0 {
			config.CurrentContext = clusterName
		}
	}

	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		b.Fatalf("failed to write kubeconfig: %v", err)
	}

	return kubeconfigPath
}
