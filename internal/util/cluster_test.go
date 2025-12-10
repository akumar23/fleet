package util

import "testing"

func TestShortClusterName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AWS EKS ARN",
			input:    "arn:aws:eks:us-east-1:123456789012:cluster/my-cluster",
			expected: "my-cluster",
		},
		{
			name:     "AWS GovCloud EKS ARN",
			input:    "arn:aws-us-gov:eks:us-gov-east-1:144418179842:cluster/gov-mosaic-dev1125-eks",
			expected: "gov-mosaic-dev1125-eks",
		},
		{
			name:     "Simple cluster name",
			input:    "my-cluster",
			expected: "my-cluster",
		},
		{
			name:     "Cluster name with dashes",
			input:    "prod-east-cluster",
			expected: "prod-east-cluster",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "ARN without cluster prefix",
			input:    "arn:aws:eks:us-east-1:123456789012:nodegroup/my-nodegroup",
			expected: "my-nodegroup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortClusterName(tt.input)
			if result != tt.expected {
				t.Errorf("ShortClusterName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
