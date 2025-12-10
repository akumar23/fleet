package delete_test

import (
	"fmt"
)

// Example demonstrates the delete command usage
func Example() {
	fmt.Println("Command: delete")
	fmt.Println("Short description: Delete resources from clusters")

	// Output:
	// Command: delete
	// Short description: Delete resources from clusters
}

// ExampleDeleteCommand_fromFile demonstrates deleting from a manifest file
func Example_deleteCommand_fromFile() {
	fmt.Println("Example: fleet delete -f deployment.yaml")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Parses the deployment.yaml file")
	fmt.Println("- Shows confirmation prompt with resources to delete")
	fmt.Println("- Deletes resources from all clusters concurrently")
	fmt.Println("- Reports success/failure for each cluster")

	// Output:
	// Example: fleet delete -f deployment.yaml
	//
	// Expected behavior:
	// - Parses the deployment.yaml file
	// - Shows confirmation prompt with resources to delete
	// - Deletes resources from all clusters concurrently
	// - Reports success/failure for each cluster
}

// ExampleDeleteCommand_byName demonstrates deleting by resource type and name
func Example_deleteCommand_byName() {
	fmt.Println("Example: fleet delete deployment nginx -n default")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Deletes 'deployment/nginx' from 'default' namespace")
	fmt.Println("- Prompts for confirmation")
	fmt.Println("- Deletes from all clusters concurrently")
	fmt.Println("- Reports results")

	// Output:
	// Example: fleet delete deployment nginx -n default
	//
	// Expected behavior:
	// - Deletes 'deployment/nginx' from 'default' namespace
	// - Prompts for confirmation
	// - Deletes from all clusters concurrently
	// - Reports results
}

// ExampleDeleteCommand_dryRun demonstrates dry-run mode
func Example_deleteCommand_dryRun() {
	fmt.Println("Example: fleet delete -f deployment.yaml --dry-run")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Shows what would be deleted without making changes")
	fmt.Println("- No confirmation prompt in dry-run mode")
	fmt.Println("- Validates that resources exist")
	fmt.Println("- Safe to run in production")

	// Output:
	// Example: fleet delete -f deployment.yaml --dry-run
	//
	// Expected behavior:
	// - Shows what would be deleted without making changes
	// - No confirmation prompt in dry-run mode
	// - Validates that resources exist
	// - Safe to run in production
}

// ExampleDeleteCommand_skipConfirmation demonstrates skipping confirmation
func Example_deleteCommand_skipConfirmation() {
	fmt.Println("Example: fleet delete -f deployment.yaml -y")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Skips confirmation prompt")
	fmt.Println("- Immediately deletes resources")
	fmt.Println("- Useful for automated cleanup scripts")
	fmt.Println("- Use with caution!")

	// Output:
	// Example: fleet delete -f deployment.yaml -y
	//
	// Expected behavior:
	// - Skips confirmation prompt
	// - Immediately deletes resources
	// - Useful for automated cleanup scripts
	// - Use with caution!
}

// ExampleDeleteCommand_targetClusters demonstrates targeting specific clusters
func Example_deleteCommand_targetClusters() {
	fmt.Println("Example: fleet delete deployment nginx --clusters staging")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Deletes only from 'staging' cluster")
	fmt.Println("- Leaves resources in other clusters untouched")
	fmt.Println("- Useful for cluster-specific cleanup")

	// Output:
	// Example: fleet delete deployment nginx --clusters staging
	//
	// Expected behavior:
	// - Deletes only from 'staging' cluster
	// - Leaves resources in other clusters untouched
	// - Useful for cluster-specific cleanup
}

// ExampleDeleteCommand_shortForms demonstrates using resource short forms
func Example_deleteCommand_shortForms() {
	fmt.Println("Supported short forms:")
	fmt.Println("- deploy (deployment)")
	fmt.Println("- svc (service)")
	fmt.Println("- cm (configmap)")
	fmt.Println("- ns (namespace)")
	fmt.Println("- sa (serviceaccount)")
	fmt.Println("- ing (ingress)")
	fmt.Println("- sts (statefulset)")
	fmt.Println("- ds (daemonset)")
	fmt.Println("- pvc (persistentvolumeclaim)")
	fmt.Println()
	fmt.Println("Example: fleet delete deploy nginx")
	fmt.Println("Same as: fleet delete deployment nginx")

	// Output:
	// Supported short forms:
	// - deploy (deployment)
	// - svc (service)
	// - cm (configmap)
	// - ns (namespace)
	// - sa (serviceaccount)
	// - ing (ingress)
	// - sts (statefulset)
	// - ds (daemonset)
	// - pvc (persistentvolumeclaim)
	//
	// Example: fleet delete deploy nginx
	// Same as: fleet delete deployment nginx
}

// Example_deleteErrorHandling demonstrates error handling
func Example_deleteErrorHandling() {
	fmt.Println("Error handling features:")
	fmt.Println()
	fmt.Println("1. Resource not found:")
	fmt.Println("   - Reports as error but continues with other clusters")
	fmt.Println("   - Shows which clusters succeeded/failed")
	fmt.Println()
	fmt.Println("2. Partial failures:")
	fmt.Println("   - Deletes from all reachable clusters")
	fmt.Println("   - Reports failures separately")
	fmt.Println("   - Exit code indicates failure")
	fmt.Println()
	fmt.Println("3. Permission errors:")
	fmt.Println("   - Clear error messages about RBAC issues")
	fmt.Println("   - Shows which resources failed to delete")

	// Output:
	// Error handling features:
	//
	// 1. Resource not found:
	//    - Reports as error but continues with other clusters
	//    - Shows which clusters succeeded/failed
	//
	// 2. Partial failures:
	//    - Deletes from all reachable clusters
	//    - Reports failures separately
	//    - Exit code indicates failure
	//
	// 3. Permission errors:
	//    - Clear error messages about RBAC issues
	//    - Shows which resources failed to delete
}

// Example_deleteSafetyFeatures demonstrates safety features
func Example_deleteSafetyFeatures() {
	fmt.Println("Safety features:")
	fmt.Println()
	fmt.Println("1. Confirmation prompt (default):")
	fmt.Println("   - Shows exactly what will be deleted")
	fmt.Println("   - Lists all target clusters")
	fmt.Println("   - Requires explicit 'y' or 'yes'")
	fmt.Println()
	fmt.Println("2. Dry-run mode:")
	fmt.Println("   - Preview deletions without making changes")
	fmt.Println("   - Validates resource existence")
	fmt.Println()
	fmt.Println("3. Cluster targeting:")
	fmt.Println("   - Explicitly specify clusters with --clusters")
	fmt.Println("   - Prevents accidental deletion across all clusters")

	// Output:
	// Safety features:
	//
	// 1. Confirmation prompt (default):
	//    - Shows exactly what will be deleted
	//    - Lists all target clusters
	//    - Requires explicit 'y' or 'yes'
	//
	// 2. Dry-run mode:
	//    - Preview deletions without making changes
	//    - Validates resource existence
	//
	// 3. Cluster targeting:
	//    - Explicitly specify clusters with --clusters
	//    - Prevents accidental deletion across all clusters
}

// Example_deleteUseCases demonstrates common use cases
func Example_deleteUseCases() {
	fmt.Println("Common use cases:")
	fmt.Println()
	fmt.Println("1. Clean up a deployment:")
	fmt.Println("   fleet delete -f app.yaml")
	fmt.Println()
	fmt.Println("2. Remove a specific resource:")
	fmt.Println("   fleet delete pod failing-pod-123 -n default")
	fmt.Println()
	fmt.Println("3. Clean up testing resources:")
	fmt.Println("   fleet delete -f test-fixtures/ -R --clusters test")
	fmt.Println()
	fmt.Println("4. Force delete in CI/CD:")
	fmt.Println("   fleet delete -f deployment.yaml -y --clusters staging")
	fmt.Println()
	fmt.Println("5. Preview cleanup:")
	fmt.Println("   fleet delete -f old-resources.yaml --dry-run")

	// Output:
	// Common use cases:
	//
	// 1. Clean up a deployment:
	//    fleet delete -f app.yaml
	//
	// 2. Remove a specific resource:
	//    fleet delete pod failing-pod-123 -n default
	//
	// 3. Clean up testing resources:
	//    fleet delete -f test-fixtures/ -R --clusters test
	//
	// 4. Force delete in CI/CD:
	//    fleet delete -f deployment.yaml -y --clusters staging
	//
	// 5. Preview cleanup:
	//    fleet delete -f old-resources.yaml --dry-run
}
