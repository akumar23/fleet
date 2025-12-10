package delete

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/aryankumar/fleet/internal/cluster"
	"github.com/aryankumar/fleet/internal/config"
	"github.com/aryankumar/fleet/internal/executor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// DeleteResult represents the result of deleting a resource
type DeleteResult struct {
	Cluster   string
	Resource  string
	Kind      string
	Name      string
	Namespace string
	Action    string // Deleted, NotFound, etc.
	Error     error
}

// NewDeleteCmd creates the delete command
func NewDeleteCmd() *cobra.Command {
	var filename string
	var recursive bool
	var dryRun bool
	var namespace string
	var skipConfirmation bool
	var resourceType string
	var resourceName string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resources from clusters",
		Long: `Delete Kubernetes resources from multiple clusters concurrently.

Supports deleting from manifest files or by specifying resource type and name.
Requires confirmation before deletion unless --yes flag is provided.`,
		Example: `  # Delete resources from a manifest file
  fleet delete -f deployment.yaml

  # Delete resources from a directory
  fleet delete -f ./manifests/

  # Delete a specific resource by type and name
  fleet delete deployment nginx -n default

  # Delete from specific clusters only
  fleet delete -f deployment.yaml --clusters prod-east,prod-west

  # Dry-run to preview deletions without deleting
  fleet delete -f deployment.yaml --dry-run

  # Skip confirmation prompt
  fleet delete -f deployment.yaml -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Handle two modes: from file or by type/name
			if filename != "" {
				return runDeleteFromFile(ctx, filename, recursive, dryRun, namespace, skipConfirmation)
			}

			// Delete by type and name from args
			if len(args) < 2 {
				return fmt.Errorf("requires resource type and name, or use -f flag for file-based deletion")
			}

			resourceType = args[0]
			resourceName = args[1]

			return runDeleteByName(ctx, resourceType, resourceName, namespace, dryRun, skipConfirmation)
		},
	}

	cmd.Flags().StringVarP(&filename, "filename", "f", "", "Path to manifest file or directory")
	cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "Process directories recursively")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview deletions without deleting")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace of resources to delete")
	cmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDeleteFromFile(ctx context.Context, filename string, recursive bool, dryRun bool, overrideNamespace string, skipConfirmation bool) error {
	logger := slog.Default()

	logger.Debug("deleting from manifests",
		"filename", filename,
		"recursive", recursive,
		"dry_run", dryRun,
		"override_namespace", overrideNamespace)

	// Parse manifests from file(s)
	manifests, err := parseManifests(filename, recursive)
	if err != nil {
		return fmt.Errorf("failed to parse manifests: %w", err)
	}

	if len(manifests) == 0 {
		return fmt.Errorf("no manifests found in %s", filename)
	}

	logger.Info("parsed manifests", "count", len(manifests))

	// Override namespace if specified
	if overrideNamespace != "" {
		for _, manifest := range manifests {
			manifest.SetNamespace(overrideNamespace)
		}
		logger.Debug("overridden namespace for all resources", "namespace", overrideNamespace)
	}

	// Load kubeconfig and create cluster manager
	kubeconfigPath := viper.GetString("kubeconfig")
	loader := config.NewKubeconfigLoader(kubeconfigPath)

	mgr := cluster.NewManager(loader, logger)
	defer mgr.Close()

	// Determine which clusters to connect to
	targetClusters := viper.GetStringSlice("clusters")
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

	// Show preview and ask for confirmation unless skipped
	if !skipConfirmation && !dryRun {
		if !confirmDelete(manifests, mgr.GetClientNames()) {
			fmt.Println("Delete cancelled")
			return nil
		}
	}

	// Create executor pool
	parallelism := viper.GetInt("parallel")
	pool := executor.NewPool(parallelism, logger)

	// Submit tasks for each cluster
	clients := mgr.GetAllClients()
	for _, client := range clients {
		clusterName := client.Name
		restConfig := client.RestConfig

		task := executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				return deleteManifests(ctx, clusterName, restConfig, manifests, dryRun, logger)
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

	fmt.Printf("\n%s resources from %d cluster(s)...\n\n",
		map[bool]string{true: "Dry-running delete for", false: "Deleting"}[dryRun],
		mgr.Count())

	results := pool.Execute(execCtx)

	// Format and display results
	return formatDeleteResults(results, dryRun)
}

func runDeleteByName(ctx context.Context, resourceType, resourceName, namespace string, dryRun bool, skipConfirmation bool) error {
	logger := slog.Default()

	if namespace == "" {
		namespace = "default"
	}

	logger.Debug("deleting resource by name",
		"type", resourceType,
		"name", resourceName,
		"namespace", namespace,
		"dry_run", dryRun)

	// Load kubeconfig and create cluster manager
	kubeconfigPath := viper.GetString("kubeconfig")
	loader := config.NewKubeconfigLoader(kubeconfigPath)

	mgr := cluster.NewManager(loader, logger)
	defer mgr.Close()

	// Connect to clusters
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

	// Confirm deletion
	if !skipConfirmation && !dryRun {
		if !confirmDeleteByName(resourceType, resourceName, namespace, mgr.GetClientNames()) {
			fmt.Println("Delete cancelled")
			return nil
		}
	}

	// Create executor pool
	parallelism := viper.GetInt("parallel")
	pool := executor.NewPool(parallelism, logger)

	// Submit tasks for each cluster
	clients := mgr.GetAllClients()
	for _, client := range clients {
		clusterName := client.Name
		restConfig := client.RestConfig

		task := executor.Task{
			ClusterName: clusterName,
			Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
				return deleteResource(ctx, clusterName, restConfig, resourceType, resourceName, namespace, dryRun, logger)
			},
		}

		if err := pool.Submit(task); err != nil {
			logger.Error("failed to submit task", "cluster", clusterName, "error", err)
			continue
		}
	}

	// Execute tasks
	timeout := viper.GetDuration("timeout")
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fmt.Printf("\n%s %s/%s from %d cluster(s)...\n\n",
		map[bool]string{true: "Dry-running delete for", false: "Deleting"}[dryRun],
		resourceType,
		resourceName,
		mgr.Count())

	results := pool.Execute(execCtx)

	return formatDeleteResults(results, dryRun)
}

// parseManifests parses YAML/JSON manifests from a file or directory
func parseManifests(path string, recursive bool) ([]*unstructured.Unstructured, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	var manifests []*unstructured.Unstructured

	if info.IsDir() {
		manifests, err = parseManifestsFromDir(path, recursive)
	} else {
		manifests, err = parseManifestsFromFile(path)
	}

	return manifests, err
}

// parseManifestsFromDir parses all YAML/JSON files in a directory
func parseManifestsFromDir(dir string, recursive bool) ([]*unstructured.Unstructured, error) {
	var manifests []*unstructured.Unstructured

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			if !recursive && path != dir {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process YAML/JSON files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}

		fileManifests, err := parseManifestsFromFile(path)
		if err != nil {
			slog.Warn("failed to parse file", "path", path, "error", err)
			return nil // Continue processing other files
		}

		manifests = append(manifests, fileManifests...)
		return nil
	}

	if err := filepath.Walk(dir, walkFunc); err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return manifests, nil
}

// parseManifestsFromFile parses manifests from a single YAML/JSON file
func parseManifestsFromFile(path string) ([]*unstructured.Unstructured, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var manifests []*unstructured.Unstructured
	decoder := yaml.NewYAMLOrJSONDecoder(file, 4096)

	for {
		var obj unstructured.Unstructured
		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode manifest: %w", err)
		}

		// Skip empty objects
		if obj.Object == nil || len(obj.Object) == 0 {
			continue
		}

		manifests = append(manifests, &obj)
	}

	return manifests, nil
}

// deleteManifests deletes manifests from a single cluster
func deleteManifests(
	ctx context.Context,
	clusterName string,
	restConfig *rest.Config,
	manifests []*unstructured.Unstructured,
	dryRun bool,
	logger *slog.Logger,
) ([]DeleteResult, error) {
	// Create dynamic client for this cluster
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	results := make([]DeleteResult, 0, len(manifests))

	for _, manifest := range manifests {
		result := DeleteResult{
			Cluster:   clusterName,
			Kind:      manifest.GetKind(),
			Name:      manifest.GetName(),
			Namespace: manifest.GetNamespace(),
			Resource:  formatResourceName(manifest),
		}

		// Get the GVR for this object
		gvr, err := getGVR(manifest)
		if err != nil {
			result.Error = fmt.Errorf("failed to get GVR: %w", err)
			results = append(results, result)
			continue
		}

		// Get the resource interface
		var resourceInterface dynamic.ResourceInterface
		if manifest.GetNamespace() != "" {
			resourceInterface = dynamicClient.Resource(gvr).Namespace(manifest.GetNamespace())
		} else {
			resourceInterface = dynamicClient.Resource(gvr)
		}

		// Delete options
		deleteOptions := metav1.DeleteOptions{}
		if dryRun {
			deleteOptions.DryRun = []string{metav1.DryRunAll}
		}

		// Delete the resource
		err = resourceInterface.Delete(ctx, manifest.GetName(), deleteOptions)
		if err != nil {
			result.Error = err
			result.Action = "error"
			results = append(results, result)
			logger.Debug("failed to delete resource",
				"cluster", clusterName,
				"resource", result.Resource,
				"error", err)
			continue
		}

		result.Action = "deleted"
		if dryRun {
			result.Action += " (dry run)"
		}

		results = append(results, result)
		logger.Debug("deleted resource",
			"cluster", clusterName,
			"resource", result.Resource,
			"action", result.Action)
	}

	return results, nil
}

// deleteResource deletes a single resource by type and name
func deleteResource(
	ctx context.Context,
	clusterName string,
	restConfig *rest.Config,
	resourceType string,
	resourceName string,
	namespace string,
	dryRun bool,
	logger *slog.Logger,
) ([]DeleteResult, error) {
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	result := DeleteResult{
		Cluster:   clusterName,
		Kind:      resourceType,
		Name:      resourceName,
		Namespace: namespace,
		Resource:  fmt.Sprintf("%s/%s", resourceType, resourceName),
	}

	// Get GVR for the resource type
	gvr := getGVRForType(resourceType)

	// Get the resource interface
	var resourceInterface dynamic.ResourceInterface
	if namespace != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	// Delete options
	deleteOptions := metav1.DeleteOptions{}
	if dryRun {
		deleteOptions.DryRun = []string{metav1.DryRunAll}
	}

	// Delete the resource
	err = resourceInterface.Delete(ctx, resourceName, deleteOptions)
	if err != nil {
		result.Error = err
		result.Action = "error"
	} else {
		result.Action = "deleted"
		if dryRun {
			result.Action += " (dry run)"
		}
	}

	return []DeleteResult{result}, nil
}

// confirmDelete prompts the user for confirmation before deleting
func confirmDelete(manifests []*unstructured.Unstructured, clusters []string) bool {
	fmt.Println("WARNING: The following resources will be DELETED:")
	fmt.Println()

	for _, manifest := range manifests {
		fmt.Printf("  - %s\n", formatResourceName(manifest))
	}

	fmt.Println()
	fmt.Printf("From %d cluster(s): %s\n", len(clusters), strings.Join(clusters, ", "))
	fmt.Println()
	fmt.Print("Are you sure you want to delete these resources? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// confirmDeleteByName prompts for confirmation when deleting by name
func confirmDeleteByName(resourceType, resourceName, namespace string, clusters []string) bool {
	fmt.Printf("WARNING: Deleting %s/%s from namespace '%s'\n", resourceType, resourceName, namespace)
	fmt.Printf("From %d cluster(s): %s\n", len(clusters), strings.Join(clusters, ", "))
	fmt.Println()
	fmt.Print("Are you sure? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// formatResourceName formats a resource for display
func formatResourceName(obj *unstructured.Unstructured) string {
	name := obj.GetName()
	kind := obj.GetKind()
	namespace := obj.GetNamespace()

	if namespace != "" {
		return fmt.Sprintf("%s/%s (%s)", kind, name, namespace)
	}
	return fmt.Sprintf("%s/%s", kind, name)
}

// formatDeleteResults formats and displays delete results
func formatDeleteResults(results []executor.Result, dryRun bool) error {
	var allResults []DeleteResult
	var errors []string

	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			failureCount++
			continue
		}

		if deleteResults, ok := result.Data.([]DeleteResult); ok {
			for _, dr := range deleteResults {
				if dr.Error != nil {
					fmt.Printf("  %s [%s] %s: %v\n",
						getStatusIcon(false),
						dr.Cluster,
						dr.Resource,
						dr.Error)
					failureCount++
				} else {
					fmt.Printf("  %s [%s] %s %s\n",
						getStatusIcon(true),
						dr.Cluster,
						dr.Resource,
						dr.Action)
					successCount++
				}
				allResults = append(allResults, dr)
			}
		}
	}

	// Print errors if any
	if len(errors) > 0 {
		fmt.Println()
		fmt.Println("Cluster errors:")
		for _, errMsg := range errors {
			fmt.Printf("  %s %s\n", getStatusIcon(false), errMsg)
		}
	}

	// Print summary
	fmt.Println()
	if dryRun {
		fmt.Printf("Dry run completed: %d would be deleted, %d failed\n", successCount, failureCount)
	} else {
		fmt.Printf("Delete completed: %d deleted, %d failed\n", successCount, failureCount)
	}

	if failureCount > 0 {
		return fmt.Errorf("some resources failed to delete")
	}

	return nil
}

// getStatusIcon returns a status icon based on success/failure
func getStatusIcon(success bool) string {
	if success {
		return "✓"
	}
	return "✗"
}

// getGVR gets the GroupVersionResource for an unstructured object
func getGVR(obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	gvk := obj.GroupVersionKind()

	gvr := schema.GroupVersionResource{
		Group:   gvk.Group,
		Version: gvk.Version,
	}

	// Map Kind to Resource (pluralized, lowercase)
	kind := gvk.Kind
	resource := strings.ToLower(kind)

	// Common pluralization rules
	if strings.HasSuffix(resource, "s") {
		resource += "es"
	} else if strings.HasSuffix(resource, "y") {
		resource = strings.TrimSuffix(resource, "y") + "ies"
	} else {
		resource += "s"
	}

	gvr.Resource = resource

	return gvr, nil
}

// getGVRForType gets GVR for a resource type string (e.g., "deployment", "pod")
func getGVRForType(resourceType string) schema.GroupVersionResource {
	// Normalize the resource type
	resourceType = strings.ToLower(resourceType)

	// Common resource mappings
	knownResources := map[string]schema.GroupVersionResource{
		"pod":                   {Group: "", Version: "v1", Resource: "pods"},
		"pods":                  {Group: "", Version: "v1", Resource: "pods"},
		"service":               {Group: "", Version: "v1", Resource: "services"},
		"services":              {Group: "", Version: "v1", Resource: "services"},
		"svc":                   {Group: "", Version: "v1", Resource: "services"},
		"deployment":            {Group: "apps", Version: "v1", Resource: "deployments"},
		"deployments":           {Group: "apps", Version: "v1", Resource: "deployments"},
		"deploy":                {Group: "apps", Version: "v1", Resource: "deployments"},
		"replicaset":            {Group: "apps", Version: "v1", Resource: "replicasets"},
		"replicasets":           {Group: "apps", Version: "v1", Resource: "replicasets"},
		"rs":                    {Group: "apps", Version: "v1", Resource: "replicasets"},
		"statefulset":           {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"statefulsets":          {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"sts":                   {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"daemonset":             {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"daemonsets":            {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"ds":                    {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"configmap":             {Group: "", Version: "v1", Resource: "configmaps"},
		"configmaps":            {Group: "", Version: "v1", Resource: "configmaps"},
		"cm":                    {Group: "", Version: "v1", Resource: "configmaps"},
		"secret":                {Group: "", Version: "v1", Resource: "secrets"},
		"secrets":               {Group: "", Version: "v1", Resource: "secrets"},
		"namespace":             {Group: "", Version: "v1", Resource: "namespaces"},
		"namespaces":            {Group: "", Version: "v1", Resource: "namespaces"},
		"ns":                    {Group: "", Version: "v1", Resource: "namespaces"},
		"serviceaccount":        {Group: "", Version: "v1", Resource: "serviceaccounts"},
		"serviceaccounts":       {Group: "", Version: "v1", Resource: "serviceaccounts"},
		"sa":                    {Group: "", Version: "v1", Resource: "serviceaccounts"},
		"ingress":               {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"ingresses":             {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"ing":                   {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"persistentvolumeclaim": {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"persistentvolumeclaims": {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"pvc":                   {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	}

	if gvr, ok := knownResources[resourceType]; ok {
		return gvr
	}

	// Fallback: assume it's a core v1 resource
	// This is a simple heuristic
	resource := resourceType
	if !strings.HasSuffix(resource, "s") {
		resource += "s"
	}

	return schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: resource,
	}
}
