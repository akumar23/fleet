package get

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"
	"time"

	"github.com/aryankumar/fleet/internal/cluster"
	"github.com/aryankumar/fleet/internal/config"
	"github.com/aryankumar/fleet/internal/executor"
	"github.com/aryankumar/fleet/internal/output"
	"github.com/aryankumar/fleet/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NamespaceInfo represents namespace information for display
type NamespaceInfo struct {
	Cluster string
	Name    string
	Status  string
	Age     string
}

func newGetNamespacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespaces",
		Aliases: []string{"ns", "namespace"},
		Short:   "Get namespaces across clusters",
		Long: `Get namespaces from all connected Kubernetes clusters.

Displays namespace name, status, and age for each namespace
in the selected clusters.`,
		Example: `  # Get all namespaces across all clusters
  fleet get namespaces

  # Get namespaces from specific clusters
  fleet get namespaces --clusters prod-east,prod-west

  # Get namespaces in JSON format
  fleet get namespaces -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runGetNamespaces(ctx)
		},
	}

	return cmd
}

func runGetNamespaces(ctx context.Context) error {
	logger := slog.Default()

	logger.Debug("getting namespaces")

	// Load kubeconfig
	kubeconfigPath := viper.GetString("kubeconfig")
	loader := config.NewKubeconfigLoader(kubeconfigPath)

	// Create cluster manager
	mgr := cluster.NewManager(loader, logger)
	defer mgr.Close()

	// Determine which clusters to connect to
	targetClusters := viper.GetStringSlice("clusters")
	var err error

	if len(targetClusters) == 0 {
		err = mgr.ConnectAll(ctx)
	} else {
		err = mgr.Connect(ctx, targetClusters)
	}

	if err != nil {
		logger.Warn("some cluster connections failed", "error", err)
	}

	if mgr.Count() == 0 {
		return fmt.Errorf("no clusters connected")
	}

	logger.Info("connected to clusters", "count", mgr.Count())

	// Create executor pool
	parallelism := viper.GetInt("parallel")
	pool := executor.NewPool(parallelism, logger)

	// Submit tasks for each cluster
	clients := mgr.GetAllClients()
	for _, client := range clients {
		clusterName := client.Name
		clientset := client.Clientset

		task := executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				return getNamespaces(ctx, clientset, clusterName)
			},
		}

		if err := pool.Submit(task); err != nil {
			logger.Error("failed to submit task", "cluster", clusterName, "error", err)
			continue
		}
	}

	// Execute tasks with timeout
	timeout := viper.GetDuration("timeout")
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	results := pool.Execute(execCtx)

	// Format and display results
	return formatNamespaceResults(results)
}

func getNamespaces(ctx context.Context, clientset kubernetes.Interface, clusterName string) ([]NamespaceInfo, error) {
	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := make([]NamespaceInfo, 0, len(namespaceList.Items))
	now := time.Now()

	for _, ns := range namespaceList.Items {
		nsInfo := NamespaceInfo{
			Cluster: clusterName,
			Name:    ns.Name,
			Status:  string(ns.Status.Phase),
			Age:     calculateAge(ns.CreationTimestamp.Time, now),
		}
		namespaces = append(namespaces, nsInfo)
	}

	return namespaces, nil
}

func formatNamespaceResults(results []executor.Result) error {
	// Collect all namespaces from successful results
	var allNamespaces []NamespaceInfo
	var errors []string

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			continue
		}

		if namespaces, ok := result.Data.([]NamespaceInfo); ok {
			allNamespaces = append(allNamespaces, namespaces...)
		}
	}

	// Print errors if any
	if len(errors) > 0 {
		for _, errMsg := range errors {
			slog.Error("cluster query failed", "error", errMsg)
		}
	}

	// Format output
	outputFormat := viper.GetString("output")
	noColor := viper.GetBool("no-color")

	var format output.Format
	switch outputFormat {
	case "json":
		format = output.FormatJSON
	case "yaml":
		format = output.FormatYAML
	default:
		format = output.FormatTable
	}

	formatter := output.NewFormatter(format, output.WithNoColor(noColor))

	// For table format, create a custom table
	if format == output.FormatTable {
		return formatNamespacesTable(allNamespaces, noColor)
	}

	// For JSON/YAML, output the namespace data
	return formatter.Format(os.Stdout, allNamespaces)
}

func formatNamespacesTable(namespaces []NamespaceInfo, noColor bool) error {
	if len(namespaces) == 0 {
		fmt.Println("No namespaces found")
		return nil
	}

	colors := output.NewColorScheme(os.Stdout, noColor)

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print header
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		colors.Header("CLUSTER"),
		colors.Header("NAME"),
		colors.Header("STATUS"),
		colors.Header("AGE"))

	// Print rows
	for _, ns := range namespaces {
		statusColor := colors.Success
		if ns.Status != "Active" {
			statusColor = colors.Warning
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			colors.ClusterName(util.ShortClusterName(ns.Cluster)),
			ns.Name,
			statusColor(ns.Status),
			ns.Age)
	}

	w.Flush()
	fmt.Fprintf(os.Stdout, "\nTotal: %d namespaces\n", len(namespaces))

	return nil
}
