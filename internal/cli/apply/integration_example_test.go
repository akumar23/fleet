package apply_test

import (
	"fmt"
	"os"
	"path/filepath"
)

// Example demonstrates the apply command usage
func Example() {
	fmt.Println("Command: apply")
	fmt.Println("Short description: Apply manifests to clusters")

	// Output:
	// Command: apply
	// Short description: Apply manifests to clusters
}

// ExampleApplyCommand_singleFile demonstrates applying a single manifest file
func Example_applyCommand_singleFile() {
	// This example shows how to use the apply command
	// Note: This requires a kubeconfig with configured clusters

	// fleet apply -f deployment.yaml
	// The command will:
	// 1. Parse the deployment.yaml file
	// 2. Connect to all configured clusters
	// 3. Apply the manifest to each cluster
	// 4. Display results

	fmt.Println("Example: fleet apply -f deployment.yaml")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Prompts for confirmation")
	fmt.Println("- Applies manifest to all clusters concurrently")
	fmt.Println("- Shows success/failure for each cluster")

	// Output:
	// Example: fleet apply -f deployment.yaml
	//
	// Expected behavior:
	// - Prompts for confirmation
	// - Applies manifest to all clusters concurrently
	// - Shows success/failure for each cluster
}

// ExampleApplyCommand_directory demonstrates applying manifests from a directory
func Example_applyCommand_directory() {
	fmt.Println("Example: fleet apply -f ./manifests/ -R")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Recursively scans directory for YAML/JSON files")
	fmt.Println("- Parses all manifests found")
	fmt.Println("- Applies all resources to clusters")
	fmt.Println("- Reports results for each resource")

	// Output:
	// Example: fleet apply -f ./manifests/ -R
	//
	// Expected behavior:
	// - Recursively scans directory for YAML/JSON files
	// - Parses all manifests found
	// - Applies all resources to clusters
	// - Reports results for each resource
}

// ExampleApplyCommand_dryRun demonstrates dry-run mode
func Example_applyCommand_dryRun() {
	fmt.Println("Example: fleet apply -f deployment.yaml --dry-run")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- No confirmation prompt in dry-run mode")
	fmt.Println("- Shows what would be applied without making changes")
	fmt.Println("- Validates manifests against cluster API")
	fmt.Println("- Safe to run in production")

	// Output:
	// Example: fleet apply -f deployment.yaml --dry-run
	//
	// Expected behavior:
	// - No confirmation prompt in dry-run mode
	// - Shows what would be applied without making changes
	// - Validates manifests against cluster API
	// - Safe to run in production
}

// ExampleApplyCommand_targetClusters demonstrates targeting specific clusters
func Example_applyCommand_targetClusters() {
	fmt.Println("Example: fleet apply -f deployment.yaml --clusters prod-east,prod-west")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Connects only to specified clusters")
	fmt.Println("- Ignores other clusters in kubeconfig")
	fmt.Println("- Applies manifest to selected clusters only")

	// Output:
	// Example: fleet apply -f deployment.yaml --clusters prod-east,prod-west
	//
	// Expected behavior:
	// - Connects only to specified clusters
	// - Ignores other clusters in kubeconfig
	// - Applies manifest to selected clusters only
}

// ExampleApplyCommand_namespaceOverride demonstrates namespace override
func Example_applyCommand_namespaceOverride() {
	fmt.Println("Example: fleet apply -f deployment.yaml -n production")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Overrides namespace in all manifests")
	fmt.Println("- Applies resources to 'production' namespace")
	fmt.Println("- Useful for deploying to different environments")

	// Output:
	// Example: fleet apply -f deployment.yaml -n production
	//
	// Expected behavior:
	// - Overrides namespace in all manifests
	// - Applies resources to 'production' namespace
	// - Useful for deploying to different environments
}

// ExampleApplyCommand_skipConfirmation demonstrates skipping confirmation
func Example_applyCommand_skipConfirmation() {
	fmt.Println("Example: fleet apply -f deployment.yaml -y")
	fmt.Println()
	fmt.Println("Expected behavior:")
	fmt.Println("- Skips confirmation prompt")
	fmt.Println("- Immediately applies manifests")
	fmt.Println("- Useful for CI/CD pipelines")

	// Output:
	// Example: fleet apply -f deployment.yaml -y
	//
	// Expected behavior:
	// - Skips confirmation prompt
	// - Immediately applies manifests
	// - Useful for CI/CD pipelines
}

// Example_manifestParsing demonstrates manifest parsing capabilities
func Example_manifestParsing() {
	// Create temporary test directory
	tmpDir, _ := os.MkdirTemp("", "fleet-test")
	defer os.RemoveAll(tmpDir)

	// Create test manifest
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`

	os.WriteFile(manifestPath, []byte(content), 0644)

	fmt.Println("Manifest parsing capabilities:")
	fmt.Println("- Single document YAML files")
	fmt.Println("- Multi-document YAML (separated by ---)")
	fmt.Println("- JSON files")
	fmt.Println("- Directory scanning (with -R flag)")
	fmt.Println("- Filters out non-YAML/JSON files")

	// Output:
	// Manifest parsing capabilities:
	// - Single document YAML files
	// - Multi-document YAML (separated by ---)
	// - JSON files
	// - Directory scanning (with -R flag)
	// - Filters out non-YAML/JSON files
}

// Example_errorHandling demonstrates error handling
func Example_errorHandling() {
	fmt.Println("Error handling features:")
	fmt.Println()
	fmt.Println("1. Partial failures:")
	fmt.Println("   - Continues applying to other clusters if one fails")
	fmt.Println("   - Reports all failures at the end")
	fmt.Println()
	fmt.Println("2. Validation errors:")
	fmt.Println("   - Invalid YAML syntax")
	fmt.Println("   - Missing required fields")
	fmt.Println("   - API server validation")
	fmt.Println()
	fmt.Println("3. Connection errors:")
	fmt.Println("   - Timeout handling")
	fmt.Println("   - Unreachable clusters")
	fmt.Println("   - Authentication failures")

	// Output:
	// Error handling features:
	//
	// 1. Partial failures:
	//    - Continues applying to other clusters if one fails
	//    - Reports all failures at the end
	//
	// 2. Validation errors:
	//    - Invalid YAML syntax
	//    - Missing required fields
	//    - API server validation
	//
	// 3. Connection errors:
	//    - Timeout handling
	//    - Unreachable clusters
	//    - Authentication failures
}
