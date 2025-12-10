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

// ServiceInfo represents service information for display
type ServiceInfo struct {
	Cluster    string
	Namespace  string
	Name       string
	Type       string
	ClusterIP  string
	ExternalIP string
	Ports      string
	Age        string
}

func newGetServicesCmd() *cobra.Command {
	var namespace string
	var allNamespaces bool

	cmd := &cobra.Command{
		Use:     "services",
		Aliases: []string{"svc", "service"},
		Short:   "Get services across clusters",
		Long: `Get services from all connected Kubernetes clusters.

Displays service name, type, cluster IP, external IP, ports, and age
for each service in the selected clusters.`,
		Example: `  # Get all services in the default namespace
  fleet get services

  # Get all services in kube-system namespace
  fleet get services -n kube-system

  # Get all services across all namespaces
  fleet get services -A

  # Get services from specific clusters
  fleet get services --clusters prod-east,prod-west

  # Get services in JSON format
  fleet get services -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			return runGetServices(ctx, namespace, allNamespaces)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Filter by namespace")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Query all namespaces")

	return cmd
}

func runGetServices(ctx context.Context, namespace string, allNamespaces bool) error {
	logger := slog.Default()

	// Determine namespace to query
	queryNamespace := namespace
	if allNamespaces {
		queryNamespace = ""
	} else if queryNamespace == "" {
		queryNamespace = "default"
	}

	logger.Debug("getting services",
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
				return getServices(ctx, clientset, queryNamespace, clusterName)
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
	return formatServiceResults(results)
}

func getServices(ctx context.Context, clientset kubernetes.Interface, namespace, clusterName string) ([]ServiceInfo, error) {
	serviceList, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	services := make([]ServiceInfo, 0, len(serviceList.Items))
	now := time.Now()

	for _, svc := range serviceList.Items {
		svcInfo := ServiceInfo{
			Cluster:    clusterName,
			Namespace:  svc.Namespace,
			Name:       svc.Name,
			Type:       string(svc.Spec.Type),
			ClusterIP:  svc.Spec.ClusterIP,
			ExternalIP: getServiceExternalIP(&svc),
			Ports:      getServicePorts(&svc),
			Age:        calculateAge(svc.CreationTimestamp.Time, now),
		}
		services = append(services, svcInfo)
	}

	return services, nil
}

func getServiceExternalIP(svc *corev1.Service) string {
	// For LoadBalancer type
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		ips := make([]string, 0)
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ips = append(ips, ingress.IP)
			} else if ingress.Hostname != "" {
				ips = append(ips, ingress.Hostname)
			}
		}
		if len(ips) > 0 {
			return strings.Join(ips, ",")
		}
		return "<pending>"
	}

	// For ExternalName type
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		return svc.Spec.ExternalName
	}

	// For services with external IPs
	if len(svc.Spec.ExternalIPs) > 0 {
		return strings.Join(svc.Spec.ExternalIPs, ",")
	}

	return "<none>"
}

func getServicePorts(svc *corev1.Service) string {
	if len(svc.Spec.Ports) == 0 {
		return "<none>"
	}

	ports := make([]string, 0, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		portStr := fmt.Sprintf("%d", port.Port)
		if port.NodePort != 0 {
			portStr = fmt.Sprintf("%d:%d", port.Port, port.NodePort)
		}
		if port.Protocol != corev1.ProtocolTCP {
			portStr = fmt.Sprintf("%s/%s", portStr, port.Protocol)
		}
		ports = append(ports, portStr)
	}

	return strings.Join(ports, ",")
}

func formatServiceResults(results []executor.Result) error {
	// Collect all services from successful results
	var allServices []ServiceInfo
	var errors []string

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			continue
		}

		if services, ok := result.Data.([]ServiceInfo); ok {
			allServices = append(allServices, services...)
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
		return formatServicesTable(allServices, noColor)
	}

	// For JSON/YAML, output the service data
	return formatter.Format(os.Stdout, allServices)
}

func formatServicesTable(services []ServiceInfo, noColor bool) error {
	if len(services) == 0 {
		fmt.Println("No services found")
		return nil
	}

	colors := output.NewColorScheme(os.Stdout, noColor)

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print header
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		colors.Header("CLUSTER"),
		colors.Header("NAMESPACE"),
		colors.Header("NAME"),
		colors.Header("TYPE"),
		colors.Header("CLUSTER-IP"),
		colors.Header("EXTERNAL-IP"),
		colors.Header("PORT(S)"),
		colors.Header("AGE"))

	// Print rows
	for _, svc := range services {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			colors.ClusterName(util.ShortClusterName(svc.Cluster)),
			svc.Namespace,
			svc.Name,
			svc.Type,
			svc.ClusterIP,
			svc.ExternalIP,
			svc.Ports,
			svc.Age)
	}

	w.Flush()
	fmt.Fprintf(os.Stdout, "\nTotal: %d services\n", len(services))

	return nil
}
