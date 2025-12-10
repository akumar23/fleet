# Fleet Quick Reference

## Common Commands

### Apply
```bash
# Apply single manifest
fleet apply -f deployment.yaml

# Apply directory
fleet apply -f ./manifests/ -R

# Dry-run
fleet apply -f app.yaml --dry-run

# Skip confirmation
fleet apply -f app.yaml -y

# Target clusters
fleet apply -f app.yaml --clusters prod-east,prod-west

# Override namespace
fleet apply -f app.yaml -n production
```

### Delete
```bash
# Delete from manifest
fleet delete -f deployment.yaml

# Delete by type/name
fleet delete deployment nginx -n default

# Using short forms
fleet delete deploy nginx
fleet delete svc nginx-service
fleet delete cm app-config

# Dry-run
fleet delete -f app.yaml --dry-run

# Skip confirmation (careful!)
fleet delete -f app.yaml -y
```

### Get
```bash
# Get pods
fleet get pods
fleet get pods -n kube-system
fleet get pods -A

# Get with labels
fleet get pods -l app=nginx

# Get other resources
fleet get deployments
fleet get services
fleet get nodes
fleet get namespaces
```

### Cluster
```bash
# List clusters
fleet cluster list

# Add cluster
fleet cluster add prod-east

# Remove cluster
fleet cluster remove staging

# Switch context
fleet cluster switch prod-west
```

## Global Flags
```bash
--clusters <list>      # Target specific clusters
--parallel <n>         # Concurrent operations (default: 5)
--timeout <duration>   # Operation timeout (default: 30s)
--verbose, -v          # Debug logging
--output, -o <format>  # Output format (json, yaml, table)
--no-color             # Disable colors
```

## Resource Short Forms
```
deploy  → deployment
svc     → service
cm      → configmap
ns      → namespace
sa      → serviceaccount
ing     → ingress
sts     → statefulset
ds      → daemonset
rs      → replicaset
pvc     → persistentvolumeclaim
```

## Workflows

### Deploy to Production
```bash
# 1. Dry-run
fleet apply -f prod.yaml --clusters prod-east,prod-west --dry-run

# 2. Apply
fleet apply -f prod.yaml --clusters prod-east,prod-west

# 3. Verify
fleet get deployments --clusters prod-east,prod-west
```

### Multi-Environment
```bash
# Dev
fleet apply -f app.yaml -n dev --clusters dev

# Staging
fleet apply -f app.yaml -n staging --clusters staging

# Production
fleet apply -f app.yaml -n production --clusters prod-east,prod-west
```

### Cleanup
```bash
# Delete specific resources
fleet delete -f old-app.yaml

# Delete by name
fleet delete deploy old-deployment -n default

# Bulk delete
fleet delete -f ./old-manifests/ -R -y
```

## Tips

1. **Always dry-run first**: `--dry-run`
2. **Target clusters explicitly**: `--clusters`
3. **Use confirmation prompts**: default behavior
4. **Increase timeout for large ops**: `--timeout 10m`
5. **Enable verbose for debugging**: `-v`
6. **Organize manifests**: use directories
7. **Validate before delete**: `--dry-run`

## Exit Codes
- `0` = Success
- `1` = Failure
- `2` = Invalid arguments
