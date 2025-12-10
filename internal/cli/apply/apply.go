package apply

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// ApplyResult represents the result of applying a resource
type ApplyResult struct {
	Cluster   string
	Resource  string
	Kind      string
	Name      string
	Namespace string
	Action    string // Created, Updated, Unchanged
	Error     error
}

// NewApplyCmd creates the apply command
func NewApplyCmd() *cobra.Command {
	var filename string
	var recursive bool
	var dryRun bool
	var namespace string
	var skipConfirmation bool

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply manifests to clusters",
		Long: `Apply Kubernetes manifests to multiple clusters concurrently.

Supports applying from files or directories containing YAML/JSON manifests.
Resources are applied using server-side apply for better conflict handling.`,
		Example: `  # Apply a single manifest to all clusters
  fleet apply -f deployment.yaml

  # Apply manifests from a directory
  fleet apply -f ./manifests/

  # Apply manifests recursively from a directory
  fleet apply -f ./manifests/ -R

  # Apply to specific clusters only
  fleet apply -f deployment.yaml --clusters prod-east,prod-west

  # Dry-run to preview changes without applying
  fleet apply -f deployment.yaml --dry-run

  # Override namespace for all resources
  fleet apply -f deployment.yaml -n production

  # Skip confirmation prompt
  fleet apply -f deployment.yaml -y`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("filename is required (-f flag)")
			}

			ctx := cmd.Context()
			return runApply(ctx, filename, recursive, dryRun, namespace, skipConfirmation)
		},
	}

	cmd.Flags().StringVarP(&filename, "filename", "f", "", "Path to manifest file or directory (required)")
	cmd.Flags().BoolVarP(&recursive, "recursive", "R", false, "Process directories recursively")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Override namespace for resources")
	cmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompt")

	cmd.MarkFlagRequired("filename")

	return cmd
}

func runApply(ctx context.Context, filename string, recursive bool, dryRun bool, overrideNamespace string, skipConfirmation bool) error {
	logger := slog.Default()

	logger.Debug("applying manifests",
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
		if !confirmApply(manifests, mgr.GetClientNames()) {
			fmt.Println("Apply cancelled")
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
				return applyManifests(ctx, clusterName, restConfig, manifests, dryRun, logger)
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

	fmt.Printf("\n%s manifests to %d cluster(s)...\n\n",
		map[bool]string{true: "Dry-running", false: "Applying"}[dryRun],
		mgr.Count())

	results := pool.Execute(execCtx)

	// Format and display results
	return formatApplyResults(results, dryRun)
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
// Supports multi-document YAML files
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

// applyManifests applies manifests to a single cluster
func applyManifests(
	ctx context.Context,
	clusterName string,
	restConfig *rest.Config,
	manifests []*unstructured.Unstructured,
	dryRun bool,
	logger *slog.Logger,
) ([]ApplyResult, error) {
	// Create dynamic client for this cluster
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	results := make([]ApplyResult, 0, len(manifests))

	for _, manifest := range manifests {
		result := ApplyResult{
			Cluster:   clusterName,
			Kind:      manifest.GetKind(),
			Name:      manifest.GetName(),
			Namespace: manifest.GetNamespace(),
			Resource:  formatResourceName(manifest),
		}

		// Get the GVR (GroupVersionResource) for this object
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

		// Apply the manifest
		options := metav1.ApplyOptions{
			FieldManager: "fleet",
		}

		if dryRun {
			options.DryRun = []string{metav1.DryRunAll}
		}

		// Convert to JSON for apply
		data, err := manifest.MarshalJSON()
		if err != nil {
			result.Error = fmt.Errorf("failed to marshal manifest: %w", err)
			results = append(results, result)
			continue
		}

		// Apply the resource
		_, err = resourceInterface.Patch(
			ctx,
			manifest.GetName(),
			types.ApplyPatchType,
			data,
			metav1.PatchOptions{
				FieldManager: options.FieldManager,
				DryRun:       options.DryRun,
			},
		)

		if err != nil {
			result.Error = err
			results = append(results, result)
			logger.Debug("failed to apply resource",
				"cluster", clusterName,
				"resource", result.Resource,
				"error", err)
			continue
		}

		// Determine action based on metadata
		// Server-side apply always returns the resource, so we just mark as configured
		result.Action = "configured"

		if dryRun {
			result.Action += " (dry run)"
		}

		results = append(results, result)
		logger.Debug("applied resource",
			"cluster", clusterName,
			"resource", result.Resource,
			"action", result.Action)
	}

	return results, nil
}

// confirmApply prompts the user for confirmation before applying
func confirmApply(manifests []*unstructured.Unstructured, clusters []string) bool {
	fmt.Println("The following resources will be applied:")
	fmt.Println()

	for _, manifest := range manifests {
		fmt.Printf("  - %s\n", formatResourceName(manifest))
	}

	fmt.Println()
	fmt.Printf("To %d cluster(s): %s\n", len(clusters), strings.Join(clusters, ", "))
	fmt.Println()
	fmt.Print("Do you want to continue? [y/N]: ")

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

// formatApplyResults formats and displays apply results
func formatApplyResults(results []executor.Result, dryRun bool) error {
	var allResults []ApplyResult
	var errors []string

	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.ClusterName, result.Error))
			failureCount++
			continue
		}

		if applyResults, ok := result.Data.([]ApplyResult); ok {
			for _, ar := range applyResults {
				if ar.Error != nil {
					fmt.Printf("  %s [%s] %s: %v\n",
						getStatusIcon(false),
						ar.Cluster,
						ar.Resource,
						ar.Error)
					failureCount++
				} else {
					fmt.Printf("  %s [%s] %s %s\n",
						getStatusIcon(true),
						ar.Cluster,
						ar.Resource,
						ar.Action)
					successCount++
				}
				allResults = append(allResults, ar)
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
		fmt.Printf("Dry run completed: %d succeeded, %d failed\n", successCount, failureCount)
	} else {
		fmt.Printf("Apply completed: %d succeeded, %d failed\n", successCount, failureCount)
	}

	if failureCount > 0 {
		return fmt.Errorf("some resources failed to apply")
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

	// Use scheme to get the GVR mapping
	// This is a simplified approach - in production, you'd use RESTMapper
	gvr := schema.GroupVersionResource{
		Group:   gvk.Group,
		Version: gvk.Version,
	}

	// Map Kind to Resource (pluralized, lowercase)
	// This is a simple heuristic - RESTMapper would be more accurate
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
