# Fleet

A powerful command-line tool for managing multiple Kubernetes clusters from a single interface.

## What is Fleet?

Fleet makes it easy to work with multiple Kubernetes clusters at the same time. Instead of switching between clusters and running the same commands repeatedly, Fleet lets you:

- View resources across all your clusters in one command
- Deploy applications to multiple clusters simultaneously
- Delete resources from multiple clusters at once
- Manage cluster configurations easily

Think of it as a control center for all your Kubernetes clusters.

## Features

- **Multi-Cluster Operations**: Execute commands across all your clusters in parallel
- **Concurrent Execution**: Fast operations using worker pools (configurable parallelism)
- **Multiple Output Formats**: View results as tables, JSON, or YAML
- **Safety Features**: Dry-run mode and confirmation prompts for destructive operations
- **Flexible Targeting**: Run commands on all clusters or specific ones
- **Graceful Shutdown**: Handles interruptions cleanly
- **Shell Completions**: Tab completion for Bash, Zsh, Fish, and PowerShell

## Prerequisites

Before you can use Fleet, you need:

1. **Go 1.21 or later** (only needed if building from source)
   - Check your version: `go version`
   - Install from: https://golang.org/dl/

2. **Kubernetes clusters** with `kubectl` configured
   - Fleet uses your existing kubeconfig file (usually at `~/.kube/config`)
   - Verify your config: `kubectl config view`

3. **Access to your clusters**
   - Make sure you can connect to your clusters with `kubectl`
   - Test: `kubectl get nodes`

## Installation

### Option 1: Build from Source (Recommended for Development)

```bash
# Clone the repository
git clone https://github.com/aryankumar/fleet.git
cd fleet

# Build the binary
make build

# The binary will be in the bin/ directory
./bin/fleet --help

# Optional: Install to your PATH
make install
```

### Option 2: Build for Your Platform

```bash
# Build a static binary (portable, no dependencies)
make build-static

# Or build for all platforms
make build-all
```

This creates binaries in the `dist/` directory for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

## Quick Start

### 1. Verify Your Setup

First, make sure Fleet can see your Kubernetes clusters:

```bash
fleet cluster list
```

This shows all the clusters from your kubeconfig file.

### 2. View Resources Across Clusters

Get pods from all clusters:

```bash
fleet get pods
```

Get pods from a specific namespace:

```bash
fleet get pods -n kube-system
```

Get all pods across all namespaces:

```bash
fleet get pods -A
```

### 3. Deploy an Application

Create a deployment file (`deployment.yaml`):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
```

Preview the deployment (dry-run):

```bash
fleet apply -f deployment.yaml --dry-run
```

Deploy to all clusters:

```bash
fleet apply -f deployment.yaml
```

Deploy to specific clusters only:

```bash
fleet apply -f deployment.yaml --clusters prod-east,prod-west
```

### 4. View Your Deployment

```bash
fleet get deployments
```

### 5. Clean Up

Delete the deployment (with confirmation):

```bash
fleet delete -f deployment.yaml
```

Or delete without confirmation (use carefully!):

```bash
fleet delete -f deployment.yaml -y
```

## Common Use Cases

### Deploy to Production Clusters

```bash
# 1. Test with dry-run first
fleet apply -f app.yaml --clusters prod-us,prod-eu --dry-run

# 2. If it looks good, apply for real
fleet apply -f app.yaml --clusters prod-us,prod-eu

# 3. Verify the deployment
fleet get deployments --clusters prod-us,prod-eu
```

### Monitor All Your Clusters

```bash
# See all pods
fleet get pods -A

# See all deployments
fleet get deployments

# See all services
fleet get services

# See all nodes
fleet get nodes
```

### Deploy to Multiple Environments

```bash
# Deploy to development
fleet apply -f app.yaml --clusters dev-cluster -n development

# Deploy to staging
fleet apply -f app.yaml --clusters staging-cluster -n staging

# Deploy to production
fleet apply -f app.yaml --clusters prod-us,prod-eu -n production
```

### Apply Multiple Files

```bash
# Apply all YAML files in a directory
fleet apply -f ./kubernetes-manifests/

# Apply recursively (includes subdirectories)
fleet apply -f ./k8s/ -R
```

## Configuration

### Basic Configuration

Fleet uses your existing kubeconfig file by default (`~/.kube/config`). You can specify a different one:

```bash
fleet get pods --kubeconfig /path/to/kubeconfig
```

Or set it via environment variable:

```bash
export FLEET_KUBECONFIG=/path/to/kubeconfig
fleet get pods
```

### Advanced Configuration

Create a Fleet configuration file at `~/.fleet.yaml`:

```yaml
# Default timeout for operations
defaults:
  timeout: 30s
  parallel: 5
  outputFormat: table
  noColor: false

# Cluster metadata and aliases
clusters:
  my-prod-cluster:
    context: my-prod-cluster
    alias: production
    enabled: true
    labels:
      env: production
      region: us-east-1

  my-staging-cluster:
    context: my-staging-cluster
    alias: staging
    enabled: true
    labels:
      env: staging
      region: us-west-2
```

See `configs/fleet.yaml.example` for a complete example.

## Available Commands

### Cluster Management

```bash
# List all configured clusters
fleet cluster list

# Add a cluster
fleet cluster add my-new-cluster

# Remove a cluster
fleet cluster remove my-old-cluster

# Switch context
fleet cluster switch my-prod-cluster
```

### Get Resources

```bash
# Get pods
fleet get pods [-n namespace] [-A] [-l label=value]

# Get deployments
fleet get deployments

# Get services
fleet get services

# Get nodes
fleet get nodes

# Get namespaces
fleet get namespaces
```

### Apply Resources

```bash
# Apply from file
fleet apply -f deployment.yaml

# Apply from directory
fleet apply -f ./manifests/

# Apply recursively
fleet apply -f ./k8s/ -R

# Apply with namespace override
fleet apply -f app.yaml -n production

# Dry-run (preview only)
fleet apply -f app.yaml --dry-run

# Skip confirmation
fleet apply -f app.yaml -y
```

### Delete Resources

```bash
# Delete from file
fleet delete -f deployment.yaml

# Delete by type and name
fleet delete deployment nginx

# Delete with short forms
fleet delete deploy nginx
fleet delete svc nginx-service

# Delete from directory
fleet delete -f ./manifests/ -R

# Dry-run
fleet delete -f app.yaml --dry-run
```

### Other Commands

```bash
# Print version
fleet version

# Generate shell completions
fleet completion bash
fleet completion zsh
fleet completion fish
fleet completion powershell
```

## Global Flags

These flags work with all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--clusters` | Target specific clusters (comma-separated) | All clusters |
| `--config` | Path to Fleet config file | `~/.fleet.yaml` |
| `--kubeconfig` | Path to kubeconfig file | `~/.kube/config` |
| `-o, --output` | Output format (table, json, yaml) | `table` |
| `-p, --parallel` | Number of parallel operations | `5` |
| `--timeout` | Timeout for operations | `30s` |
| `-v, --verbose` | Enable verbose/debug logging | `false` |
| `--no-color` | Disable colored output | `false` |

## Shell Completions

### Bash

```bash
# Generate completions
fleet completion bash > /etc/bash_completion.d/fleet

# Or add to your .bashrc
echo 'source <(fleet completion bash)' >> ~/.bashrc
```

### Zsh

```bash
# Add to your .zshrc
echo 'source <(fleet completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### Fish

```bash
fleet completion fish > ~/.config/fish/completions/fleet.fish
```

### PowerShell

```powershell
fleet completion powershell | Out-String | Invoke-Expression
```

## Environment Variables

Fleet respects these environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `FLEET_KUBECONFIG` | Path to kubeconfig file | `~/.kube/config` |
| `FLEET_CONFIG` | Path to Fleet config file | `~/.fleet.yaml` |
| `FLEET_CLUSTERS` | Default target clusters | `prod-east,prod-west` |
| `FLEET_PARALLEL` | Default parallelism | `10` |

Example:

```bash
export FLEET_PARALLEL=10
export FLEET_CLUSTERS=prod-east,prod-west
fleet apply -f app.yaml
```

## Project Structure

```
fleet/
├── cmd/fleet/              # Application entry point
├── internal/
│   ├── cli/               # CLI commands (apply, delete, get, etc.)
│   ├── cluster/           # Cluster connection management
│   ├── config/            # Configuration loading
│   ├── executor/          # Concurrent execution engine
│   ├── output/            # Output formatting (table, JSON, YAML)
│   └── util/              # Utility functions
├── pkg/version/           # Version information
├── configs/               # Example configuration files
├── docs/                  # Additional documentation
├── testdata/              # Test fixtures
├── Makefile               # Build automation
└── README.md              # This file
```

## Development

### Building

```bash
# Build for current platform
make build

# Build static binary
make build-static

# Build for all platforms
make build-all
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run benchmarks
make test-bench

# Generate coverage report
make coverage
```

### Linting

```bash
# Run linters (requires golangci-lint)
make lint
```

### Cleanup

```bash
# Remove build artifacts
make clean
```

## Architecture

Fleet is built following Go best practices:

- **Worker Pool Pattern**: Concurrent operations use bounded worker pools instead of spawning unlimited goroutines
- **Context-Aware Cancellation**: All operations respect context deadlines and can be gracefully cancelled
- **Graceful Shutdown**: SIGINT/SIGTERM signals are handled cleanly
- **Structured Logging**: Uses Go's `log/slog` for structured, leveled logging
- **Single Binary**: Compiles to a single static binary with no external dependencies
- **Comprehensive Testing**: Unit tests, integration tests, and benchmarks

## Troubleshooting

### "No manifests found"

Make sure your files have `.yaml`, `.yml`, or `.json` extensions and the path is correct.

### "Connection timeout"

Increase the timeout:

```bash
fleet apply -f app.yaml --timeout 5m
```

Or check your cluster connectivity:

```bash
kubectl cluster-info
```

### "Permission denied"

Verify you have the necessary RBAC permissions in your clusters:

```bash
kubectl auth can-i create deployments
```

### "Failed to parse manifest"

Validate your YAML syntax:

```bash
# Using kubectl
kubectl apply --dry-run=client -f deployment.yaml
```

### "Context cancelled"

The operation timed out. Increase the timeout with `--timeout` flag.

## Best Practices

1. **Always use dry-run first** for production deployments
   ```bash
   fleet apply -f app.yaml --dry-run
   ```

2. **Target specific clusters** in production to avoid accidents
   ```bash
   fleet apply -f app.yaml --clusters prod-east,prod-west
   ```

3. **Use confirmation prompts** (don't skip with `-y` unless in CI/CD)

4. **Organize manifests** in directories for easier management

5. **Enable verbose mode** when troubleshooting
   ```bash
   fleet apply -f app.yaml -v
   ```

6. **Set appropriate timeouts** for large operations
   ```bash
   fleet apply -f large-app.yaml --timeout 10m
   ```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Run linters (`make lint`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## Additional Resources

- **Detailed Command Reference**: See [docs/COMMANDS.md](docs/COMMANDS.md)
- **Quick Reference**: See [docs/QUICK_REFERENCE.md](docs/QUICK_REFERENCE.md)
- **Configuration Example**: See [configs/fleet.yaml.example](configs/fleet.yaml.example)

## Support

If you encounter issues or have questions:

1. Check the [troubleshooting section](#troubleshooting)
2. Search [existing issues](https://github.com/aryankumar/fleet/issues)
3. Create a [new issue](https://github.com/aryankumar/fleet/issues/new) with:
   - Fleet version (`fleet version`)
   - Command you ran
   - Expected vs actual behavior
   - Relevant logs (use `-v` flag for verbose output)

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

Fleet is built with these excellent open-source projects:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes API client
- [tablewriter](https://github.com/olekukonko/tablewriter) - Table formatting
- [color](https://github.com/fatih/color) - Colored terminal output

---

**Made with ❤️ for Kubernetes operators who manage multiple clusters**
