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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PodInfo represents pod information for display
type PodInfo struct {
	Cluster   string
	Namespace string
	Name      string
	Ready     string
	Status    string
	Restarts  int32
	Age       string
}

func newGetPodsCmd() *cobra.Command {
	var namespace string
	var selector string
	var allNamespaces bool

	cmd := &cobra.Command{
		Use:   "pods",
		Short: "Get pods across clusters",
		Long: `Get pods from all connected Kubernetes clusters.

Supports filtering by namespace and label selectors. Results are displayed
with cluster name, namespace, pod name, ready status, phase, restart count, and age.`,
		Example: `  # Get all pods in the default namespace
  fleet get pods

  # Get all pods in kube-system namespace
  fleet get pods -n kube-system

  # Get all pods across all namespaces
  fleet get pods -A

  # Get pods with label selector
  fleet get pods -l app=nginx

  # Get pods in specific clusters
  fleet get pods --clusters prod-east,prod-west

  # Get pods in JSON format
  fleet get pods -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runGetPods(ctx, namespace, selector, allNamespaces)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Filter by namespace")
	cmd.Flags().StringVarP(&selector, "selector", "l", "", "Label selector to filter pods")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Query all namespaces")

	return cmd
}

func runGetPods(ctx context.Context, namespace, selector string, allNamespaces bool) error {
	logger := slog.Default()

	// Determine namespace to query
	queryNamespace := namespace
	if allNamespaces {
		queryNamespace = "" // Empty string means all namespaces in client-go
	} else if queryNamespace == "" {
		queryNamespace = "default"
	}

	logger.Debug("getting pods",
		"namespace", queryNamespace,
		"selector", selector,
		"all_namespaces", allNamespaces)

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
		// Connect to all clusters
		err = mgr.ConnectAll(ctx)
	} else {
		// Connect to specific clusters
		err = mgr.Connect(ctx, targetClusters)
	}

	if err != nil {
		// Continue with partial connections
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
		// Capture variables for closure
		clusterName := client.Name
		clientset := client.Clientset

		task := executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				return getPods(ctx, clientset, queryNamespace, selector, clusterName, allNamespaces)
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
	return formatPodResults(results)
}

func getPods(ctx context.Context, clientset kubernetes.Interface, namespace, selector, clusterName string, allNamespaces bool) ([]PodInfo, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	pods := make([]PodInfo, 0, len(podList.Items))
	now := time.Now()

	for _, pod := range podList.Items {
		podInfo := PodInfo{
			Cluster:   clusterName,
			Namespace: pod.Namespace,
			Name:      pod.Name,
			Ready:     calculateReadyStatus(&pod),
			Status:    string(pod.Status.Phase),
			Restarts:  calculateRestarts(&pod),
			Age:       calculateAge(pod.CreationTimestamp.Time, now),
		}
		pods = append(pods, podInfo)
	}

	return pods, nil
}

func calculateReadyStatus(pod *corev1.Pod) string {
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Ready {
			readyContainers++
		}
	}

	return fmt.Sprintf("%d/%d", readyContainers, totalContainers)
}

func calculateRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

func calculateAge(created time.Time, now time.Time) string {
	duration := now.Sub(created)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

func formatPodResults(results []executor.Result) error {
	// Collect all pods from successful results
	var allPods []PodInfo
	var errors []string

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			continue
		}

		if pods, ok := result.Data.([]PodInfo); ok {
			allPods = append(allPods, pods...)
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
		return formatPodsTable(allPods, noColor)
	}

	// For JSON/YAML, output the pod data
	return formatter.Format(os.Stdout, allPods)
}

func formatPodsTable(pods []PodInfo, noColor bool) error {
	if len(pods) == 0 {
		fmt.Println("No pods found")
		return nil
	}

	colors := output.NewColorScheme(os.Stdout, noColor)

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print header
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		colors.Header("CLUSTER"),
		colors.Header("NAMESPACE"),
		colors.Header("NAME"),
		colors.Header("READY"),
		colors.Header("STATUS"),
		colors.Header("RESTARTS"),
		colors.Header("AGE"))

	for _, pod := range pods {
		statusColor := colors.Success
		if pod.Status != "Running" {
			statusColor = colors.Warning
		}
		if pod.Status == "Failed" || pod.Status == "Unknown" {
			statusColor = colors.Error
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			colors.ClusterName(util.ShortClusterName(pod.Cluster)),
			pod.Namespace,
			pod.Name,
			pod.Ready,
			statusColor(pod.Status),
			pod.Restarts,
			pod.Age)
	}

	w.Flush()
	fmt.Fprintf(os.Stdout, "\nTotal: %d pods\n", len(pods))

	return nil
}
