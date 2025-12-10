# Fleet Commands Reference

## Table of Contents
- [Apply](#apply-command)
- [Delete](#delete-command)
- [Get](#get-command)
- [Cluster](#cluster-command)
- [Global Flags](#global-flags)

---

## Apply Command

Apply Kubernetes manifests to multiple clusters concurrently.

### Synopsis
```bash
fleet apply -f <file|directory> [flags]
```

### Description
The `apply` command deploys Kubernetes resources defined in YAML or JSON manifests to one or more clusters. It uses server-side apply for better conflict handling and supports both single files and directories.

### Flags
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--filename` | `-f` | Path to manifest file or directory (required) | - |
| `--recursive` | `-R` | Process directories recursively | false |
| `--dry-run` | - | Preview changes without applying | false |
| `--namespace` | `-n` | Override namespace for all resources | - |
| `--yes` | `-y` | Skip confirmation prompt | false |

### Examples

#### Apply a single manifest
```bash
fleet apply -f deployment.yaml
```

#### Apply all manifests in a directory
```bash
fleet apply -f ./manifests/
```

#### Apply recursively with namespace override
```bash
fleet apply -f ./k8s/ -R -n production
```

#### Apply to specific clusters only
```bash
fleet apply -f app.yaml --clusters prod-east,prod-west
```

#### Dry-run to preview changes
```bash
fleet apply -f deployment.yaml --dry-run
```

#### Skip confirmation (for CI/CD)
```bash
fleet apply -f deployment.yaml -y
```

### Behavior

1. **Manifest Parsing**:
   - Supports YAML and JSON files
   - Handles multi-document YAML (separated by `---`)
   - Filters for `.yaml`, `.yml`, and `.json` extensions
   - Continues on parsing errors for individual files

2. **Confirmation Prompt**:
   - Shows all resources to be applied
   - Lists target clusters
   - Requires explicit confirmation (y/yes)
   - Skipped in dry-run mode or with `-y` flag

3. **Concurrent Execution**:
   - Applies to all clusters in parallel
   - Respects `--parallel` flag (default: 5)
   - Context-aware cancellation
   - Progress reporting

4. **Error Handling**:
   - Continues with other clusters on failure
   - Reports all errors at the end
   - Exit code 1 if any failures
   - Partial success possible

### Output Example
```
Applying manifests to 3 cluster(s)...

  ✓ [prod-east] Deployment/nginx configured
  ✓ [prod-west] Deployment/nginx configured
  ✗ [staging] Deployment/nginx: connection timeout

Apply completed: 2 succeeded, 1 failed
```

---

## Delete Command

Delete Kubernetes resources from multiple clusters concurrently.

### Synopsis
```bash
# Delete from manifest file
fleet delete -f <file|directory> [flags]

# Delete by resource type and name
fleet delete <type> <name> [flags]
```

### Description
The `delete` command removes Kubernetes resources from one or more clusters. It supports deletion from manifest files or by specifying resource type and name directly.

### Flags
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--filename` | `-f` | Path to manifest file or directory | - |
| `--recursive` | `-R` | Process directories recursively | false |
| `--dry-run` | - | Preview deletions without deleting | false |
| `--namespace` | `-n` | Namespace of resources to delete | default |
| `--yes` | `-y` | Skip confirmation prompt | false |

### Resource Short Forms
The delete command supports kubectl-style short forms:

| Short Form | Full Name |
|------------|-----------|
| `deploy` | deployment |
| `svc` | service |
| `cm` | configmap |
| `ns` | namespace |
| `sa` | serviceaccount |
| `ing` | ingress |
| `sts` | statefulset |
| `ds` | daemonset |
| `rs` | replicaset |
| `pvc` | persistentvolumeclaim |

### Examples

#### Delete from manifest file
```bash
fleet delete -f deployment.yaml
```

#### Delete from directory
```bash
fleet delete -f ./manifests/ -R
```

#### Delete by type and name
```bash
fleet delete deployment nginx -n default
```

#### Delete using short forms
```bash
fleet delete deploy nginx
fleet delete svc nginx-service
fleet delete cm app-config
```

#### Delete from specific clusters
```bash
fleet delete -f app.yaml --clusters staging
```

#### Dry-run to preview deletions
```bash
fleet delete -f deployment.yaml --dry-run
```

#### Skip confirmation (use with caution!)
```bash
fleet delete -f deployment.yaml -y
```

### Behavior

1. **Confirmation Prompt**:
   - Shows WARNING with resources to be deleted
   - Lists all target clusters
   - Requires explicit confirmation (y/yes)
   - Skipped in dry-run mode or with `-y` flag

2. **Concurrent Execution**:
   - Deletes from all clusters in parallel
   - Respects `--parallel` flag
   - Context-aware cancellation

3. **Error Handling**:
   - Reports "not found" as error
   - Continues with other clusters
   - Aggregated error reporting
   - Non-zero exit code on failures

4. **Safety Features**:
   - WARNING message in confirmation
   - Dry-run mode available
   - Cluster targeting to prevent accidents
   - Explicit confirmation required

### Output Example
```
WARNING: The following resources will be DELETED:

  - Deployment/nginx (default)
  - Service/nginx-service (default)

From 2 cluster(s): prod-east, prod-west

Are you sure you want to delete these resources? [y/N]: y

Deleting resources from 2 cluster(s)...

  ✓ [prod-east] Deployment/nginx deleted
  ✓ [prod-east] Service/nginx-service deleted
  ✓ [prod-west] Deployment/nginx deleted
  ✓ [prod-west] Service/nginx-service deleted

Delete completed: 4 deleted, 0 failed
```

---

## Get Command

Retrieve resources from multiple clusters.

### Synopsis
```bash
fleet get <resource-type> [flags]
```

### Supported Resources
- `pods`
- `nodes`
- `deployments`
- `services`
- `namespaces`

### Flags
| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--namespace` | `-n` | Filter by namespace | default |
| `--all-namespaces` | `-A` | Query all namespaces | false |
| `--selector` | `-l` | Label selector to filter | - |

### Examples

```bash
# Get all pods across all clusters
fleet get pods

# Get pods in specific namespace
fleet get pods -n kube-system

# Get pods across all namespaces
fleet get pods -A

# Get pods with label selector
fleet get pods -l app=nginx

# Get deployments in JSON format
fleet get deployments -o json
```

---

## Cluster Command

Manage cluster configurations.

### Synopsis
```bash
fleet cluster <subcommand> [flags]
```

### Subcommands
- `add` - Add a new cluster configuration
- `list` - List all configured clusters
- `remove` - Remove a cluster configuration
- `switch` - Switch to a different cluster context

### Examples

```bash
# List all clusters
fleet cluster list

# Add a new cluster
fleet cluster add prod-east

# Remove a cluster
fleet cluster remove staging

# Switch to a different context
fleet cluster switch prod-west
```

---

## Global Flags

These flags are available for all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--clusters` | - | Target specific clusters (comma-separated) | all |
| `--config` | - | Config file path | ~/.fleet.yaml |
| `--kubeconfig` | - | Kubeconfig file path | ~/.kube/config |
| `--no-color` | - | Disable colored output | false |
| `--output` | `-o` | Output format (json, yaml, table) | table |
| `--parallel` | `-p` | Number of parallel operations | 5 |
| `--timeout` | - | Timeout for operations | 30s |
| `--verbose` | `-v` | Verbose output with debug logging | false |

### Examples

```bash
# Target specific clusters
fleet apply -f app.yaml --clusters prod-east,prod-west

# Increase parallelism
fleet get pods --parallel 10

# Increase timeout
fleet apply -f large-deployment.yaml --timeout 5m

# Enable verbose logging
fleet delete -f app.yaml -v

# Use different kubeconfig
fleet get pods --kubeconfig /path/to/kubeconfig

# Output in JSON
fleet get deployments -o json

# Disable colors (for CI/CD)
fleet apply -f app.yaml --no-color
```

---

## Common Workflows

### Deploy to Production
```bash
# 1. Dry-run first to verify
fleet apply -f production.yaml --clusters prod-east,prod-west --dry-run

# 2. Apply if everything looks good
fleet apply -f production.yaml --clusters prod-east,prod-west

# 3. Verify deployment
fleet get deployments --clusters prod-east,prod-west
```

### Rollback Deployment
```bash
# 1. Delete current deployment
fleet delete -f new-version.yaml

# 2. Apply previous version
fleet apply -f previous-version.yaml
```

### Multi-Environment Deployment
```bash
# Deploy to dev
fleet apply -f app.yaml -n dev --clusters dev-cluster

# Deploy to staging
fleet apply -f app.yaml -n staging --clusters staging-cluster

# Deploy to production
fleet apply -f app.yaml -n production --clusters prod-east,prod-west
```

### Cleanup Test Resources
```bash
# Delete all test resources
fleet delete -f test-fixtures/ -R --clusters test-cluster -y
```

### Monitor Across Clusters
```bash
# Get all pods
fleet get pods -A

# Check specific namespace
fleet get pods -n kube-system

# Get deployments
fleet get deployments
```

---

## Best Practices

### 1. Always Use Dry-Run First
```bash
# Preview changes before applying
fleet apply -f deployment.yaml --dry-run
```

### 2. Target Specific Clusters in Production
```bash
# Explicit cluster targeting prevents accidents
fleet apply -f app.yaml --clusters prod-east,prod-west
```

### 3. Use Confirmation Prompts
```bash
# Don't skip confirmation unless in CI/CD
fleet apply -f app.yaml  # Will prompt
fleet apply -f app.yaml -y  # Skips prompt
```

### 4. Organize Manifests
```bash
# Use directory structure for organization
fleet apply -f ./k8s/base/ -R
fleet apply -f ./k8s/overlays/production/ -R
```

### 5. Use Verbose Mode for Debugging
```bash
# Enable verbose logging to troubleshoot
fleet apply -f app.yaml -v
```

### 6. Set Appropriate Timeouts
```bash
# Increase timeout for large operations
fleet apply -f large-app.yaml --timeout 10m
```

### 7. Validate Before Deleting
```bash
# Always dry-run delete operations
fleet delete -f app.yaml --dry-run
```

---

## Troubleshooting

### Issue: "No manifests found"
**Solution**: Check file path and ensure files have .yaml, .yml, or .json extension

### Issue: "Connection timeout"
**Solution**: Increase timeout with `--timeout` flag or check cluster connectivity

### Issue: "Permission denied"
**Solution**: Verify RBAC permissions in target clusters

### Issue: "Context cancelled"
**Solution**: Operation took too long, increase `--timeout`

### Issue: "Failed to parse manifest"
**Solution**: Validate YAML/JSON syntax, check for proper indentation

### Issue: "Resource not found" (delete)
**Solution**: Verify resource exists in target clusters, check namespace

---

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Partial or complete failure |
| 2 | Invalid arguments or flags |

---

## Environment Variables

Fleet respects the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `FLEET_KUBECONFIG` | Path to kubeconfig file | ~/.kube/config |
| `FLEET_CONFIG` | Path to fleet config file | ~/.fleet.yaml |
| `FLEET_CLUSTERS` | Comma-separated list of clusters | all |
| `FLEET_PARALLEL` | Number of parallel operations | 5 |

Example:
```bash
export FLEET_PARALLEL=10
export FLEET_CLUSTERS=prod-east,prod-west
fleet apply -f app.yaml
```

---

## Additional Resources

- [GitHub Repository](https://github.com/aryankumar/fleet)
- [Issue Tracker](https://github.com/aryankumar/fleet/issues)
- [Contributing Guide](../CONTRIBUTING.md)
- [Development Guide](../DEVELOPMENT.md)
