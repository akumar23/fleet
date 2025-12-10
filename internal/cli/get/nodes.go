package get

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aryankumar/fleet/internal/cluster"
	"github.com/aryankumar/fleet/internal/config"
	"github.com/aryankumar/fleet/internal/executor"
	"github.com/aryankumar/fleet/internal/output"
	"github.com/aryankumar/fleet/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NodeInfo represents node information for display
type NodeInfo struct {
	Cluster string
	Name    string
	Status  string
	Roles   string
	Age     string
	Version string
}

func newGetNodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes",
		Short: "Get nodes across clusters",
		Long: `Get nodes from all connected Kubernetes clusters.

Displays node name, status, roles, age, and Kubernetes version for each node
in the selected clusters.`,
		Example: `  # Get all nodes across all clusters
  fleet get nodes

  # Get nodes from specific clusters
  fleet get nodes --clusters prod-east,prod-west

  # Get nodes in JSON format
  fleet get nodes -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runGetNodes(ctx)
		},
	}

	return cmd
}

func runGetNodes(ctx context.Context) error {
	logger := slog.Default()

	logger.Debug("getting nodes")

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
				return getNodes(ctx, clientset, clusterName)
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
	return formatNodeResults(results)
}

func getNodes(ctx context.Context, clientset kubernetes.Interface, clusterName string) ([]NodeInfo, error) {
	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodes := make([]NodeInfo, 0, len(nodeList.Items))
	now := time.Now()

	for _, node := range nodeList.Items {
		nodeInfo := NodeInfo{
			Cluster: clusterName,
			Name:    node.Name,
			Status:  getNodeStatus(&node),
			Roles:   getNodeRoles(&node),
			Age:     calculateAge(node.CreationTimestamp.Time, now),
			Version: node.Status.NodeInfo.KubeletVersion,
		}
		nodes = append(nodes, nodeInfo)
	}

	return nodes, nil
}

func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func getNodeRoles(node *corev1.Node) string {
	roles := []string{}

	// Check labels for roles
	for key := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}

	if len(roles) == 0 {
		return "<none>"
	}

	return strings.Join(roles, ",")
}

func formatNodeResults(results []executor.Result) error {
	// Collect all nodes from successful results
	var allNodes []NodeInfo
	var errors []string

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			continue
		}

		if nodes, ok := result.Data.([]NodeInfo); ok {
			allNodes = append(allNodes, nodes...)
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
		return formatNodesTable(allNodes, noColor)
	}

	// For JSON/YAML, output the node data
	return formatter.Format(os.Stdout, allNodes)
}

func formatNodesTable(nodes []NodeInfo, noColor bool) error {
	if len(nodes) == 0 {
		fmt.Println("No nodes found")
		return nil
	}

	colors := output.NewColorScheme(os.Stdout, noColor)

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print header
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		colors.Header("CLUSTER"),
		colors.Header("NAME"),
		colors.Header("STATUS"),
		colors.Header("ROLES"),
		colors.Header("AGE"),
		colors.Header("VERSION"))

	// Print rows
	for _, node := range nodes {
		statusColor := colors.Success
		if node.Status != "Ready" {
			statusColor = colors.Error
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			colors.ClusterName(util.ShortClusterName(node.Cluster)),
			node.Name,
			statusColor(node.Status),
			node.Roles,
			node.Age,
			node.Version)
	}

	w.Flush()
	fmt.Fprintf(os.Stdout, "\nTotal: %d nodes\n", len(nodes))

	return nil
}
