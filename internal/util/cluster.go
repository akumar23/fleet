package util

import "strings"

// ShortClusterName extracts the short cluster name from an ARN or returns the original name.
// AWS EKS ARNs have the format: arn:aws:eks:region:account-id:cluster/cluster-name
// or arn:aws-gov:eks:region:account-id:cluster/cluster-name for GovCloud.
func ShortClusterName(name string) string {
	// Check if it's an ARN (starts with "arn:")
	if !strings.HasPrefix(name, "arn:") {
		return name
	}

	// Look for "cluster/" which precedes the cluster name in EKS ARNs
	if idx := strings.LastIndex(name, "cluster/"); idx != -1 {
		return name[idx+len("cluster/"):]
	}

	// For other ARN formats, try to extract the last segment after the last "/"
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		return name[idx+1:]
	}

	// If no "/" found, return the last segment after ":"
	if idx := strings.LastIndex(name, ":"); idx != -1 {
		return name[idx+1:]
	}

	return name
}
