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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeploymentInfo represents deployment information for display
type DeploymentInfo struct {
	Cluster   string
	Namespace string
	Name      string
	Ready     string
	UpToDate  int32
	Available int32
	Age       string
}

func newGetDeploymentsCmd() *cobra.Command {
	var namespace string
	var allNamespaces bool

	cmd := &cobra.Command{
		Use:     "deployments",
		Aliases: []string{"deploy", "deployment"},
		Short:   "Get deployments across clusters",
		Long: `Get deployments from all connected Kubernetes clusters.

Displays deployment name, ready status, up-to-date replicas, available replicas,
and age for each deployment in the selected clusters.`,
		Example: `  # Get all deployments in the default namespace
  fleet get deployments

  # Get all deployments in kube-system namespace
  fleet get deployments -n kube-system

  # Get all deployments across all namespaces
  fleet get deployments -A

  # Get deployments from specific clusters
  fleet get deployments --clusters prod-east,prod-west

  # Get deployments in JSON format
  fleet get deployments -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runGetDeployments(ctx, namespace, allNamespaces)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Filter by namespace")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Query all namespaces")

	return cmd
}

func runGetDeployments(ctx context.Context, namespace string, allNamespaces bool) error {
	logger := slog.Default()

	// Determine namespace to query
	queryNamespace := namespace
	if allNamespaces {
		queryNamespace = ""
	} else if queryNamespace == "" {
		queryNamespace = "default"
	}

	logger.Debug("getting deployments",
		"namespace", queryNamespace,
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
				return getDeployments(ctx, clientset, queryNamespace, clusterName)
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
	return formatDeploymentResults(results)
}

func getDeployments(ctx context.Context, clientset kubernetes.Interface, namespace, clusterName string) ([]DeploymentInfo, error) {
	deploymentList, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]DeploymentInfo, 0, len(deploymentList.Items))
	now := time.Now()

	for _, deploy := range deploymentList.Items {
		deployInfo := DeploymentInfo{
			Cluster:   clusterName,
			Namespace: deploy.Namespace,
			Name:      deploy.Name,
			Ready:     calculateDeploymentReady(&deploy),
			UpToDate:  deploy.Status.UpdatedReplicas,
			Available: deploy.Status.AvailableReplicas,
			Age:       calculateAge(deploy.CreationTimestamp.Time, now),
		}
		deployments = append(deployments, deployInfo)
	}

	return deployments, nil
}

func calculateDeploymentReady(deploy *appsv1.Deployment) string {
	desired := int32(0)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}
	ready := deploy.Status.ReadyReplicas
	return fmt.Sprintf("%d/%d", ready, desired)
}

func formatDeploymentResults(results []executor.Result) error {
	// Collect all deployments from successful results
	var allDeployments []DeploymentInfo
	var errors []string

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			continue
		}

		if deployments, ok := result.Data.([]DeploymentInfo); ok {
			allDeployments = append(allDeployments, deployments...)
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
		return formatDeploymentsTable(allDeployments, noColor)
	}

	// For JSON/YAML, output the deployment data
	return formatter.Format(os.Stdout, allDeployments)
}

func formatDeploymentsTable(deployments []DeploymentInfo, noColor bool) error {
	if len(deployments) == 0 {
		fmt.Println("No deployments found")
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
		colors.Header("UP-TO-DATE"),
		colors.Header("AVAILABLE"),
		colors.Header("AGE"))

	// Print rows
	for _, deploy := range deployments {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			colors.ClusterName(util.ShortClusterName(deploy.Cluster)),
			deploy.Namespace,
			deploy.Name,
			deploy.Ready,
			deploy.UpToDate,
			deploy.Available,
			deploy.Age)
	}

	w.Flush()
	fmt.Fprintf(os.Stdout, "\nTotal: %d deployments\n", len(deployments))

	return nil
}
