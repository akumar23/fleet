package delete

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseManifestsFromFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name: "single deployment",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 3
`,
			wantCount: 1,
			wantError: false,
		},
		{
			name: "multi-document YAML",
			content: `apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
			wantCount: 2,
			wantError: false,
		},
		{
			name:      "empty file",
			content:   "",
			wantCount: 0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.yaml")

			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			manifests, err := parseManifestsFromFile(tmpFile)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(manifests) != tt.wantCount {
				t.Errorf("expected %d manifests, got %d", tt.wantCount, len(manifests))
			}
		})
	}
}

func TestGetGVRForType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		want         schema.GroupVersionResource
	}{
		{
			name:         "pod",
			resourceType: "pod",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
		},
		{
			name:         "pods (plural)",
			resourceType: "pods",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
		},
		{
			name:         "deployment",
			resourceType: "deployment",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		{
			name:         "deploy (short form)",
			resourceType: "deploy",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		{
			name:         "service",
			resourceType: "service",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "services",
			},
		},
		{
			name:         "svc (short form)",
			resourceType: "svc",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "services",
			},
		},
		{
			name:         "configmap",
			resourceType: "configmap",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "configmaps",
			},
		},
		{
			name:         "cm (short form)",
			resourceType: "cm",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "configmaps",
			},
		},
		{
			name:         "secret",
			resourceType: "secret",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "secrets",
			},
		},
		{
			name:         "namespace",
			resourceType: "namespace",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "namespaces",
			},
		},
		{
			name:         "ns (short form)",
			resourceType: "ns",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "namespaces",
			},
		},
		{
			name:         "ingress",
			resourceType: "ingress",
			want: schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v1",
				Resource: "ingresses",
			},
		},
		{
			name:         "ing (short form)",
			resourceType: "ing",
			want: schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v1",
				Resource: "ingresses",
			},
		},
		{
			name:         "statefulset",
			resourceType: "statefulset",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "statefulsets",
			},
		},
		{
			name:         "sts (short form)",
			resourceType: "sts",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "statefulsets",
			},
		},
		{
			name:         "daemonset",
			resourceType: "daemonset",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "daemonsets",
			},
		},
		{
			name:         "ds (short form)",
			resourceType: "ds",
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "daemonsets",
			},
		},
		{
			name:         "persistentvolumeclaim",
			resourceType: "persistentvolumeclaim",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "persistentvolumeclaims",
			},
		},
		{
			name:         "pvc (short form)",
			resourceType: "pvc",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "persistentvolumeclaims",
			},
		},
		{
			name:         "serviceaccount",
			resourceType: "serviceaccount",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "serviceaccounts",
			},
		},
		{
			name:         "sa (short form)",
			resourceType: "sa",
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "serviceaccounts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getGVRForType(tt.resourceType)

			if got.Group != tt.want.Group {
				t.Errorf("expected group %q, got %q", tt.want.Group, got.Group)
			}
			if got.Version != tt.want.Version {
				t.Errorf("expected version %q, got %q", tt.want.Version, got.Version)
			}
			if got.Resource != tt.want.Resource {
				t.Errorf("expected resource %q, got %q", tt.want.Resource, got.Resource)
			}
		})
	}
}

func TestGetGVRForType_Unknown(t *testing.T) {
	// Test fallback behavior for unknown resource types
	resourceType := "unknownresource"
	got := getGVRForType(resourceType)

	// Should fallback to core v1 with pluralized name
	if got.Group != "" {
		t.Errorf("expected empty group for unknown resource, got %q", got.Group)
	}
	if got.Version != "v1" {
		t.Errorf("expected version v1 for unknown resource, got %q", got.Version)
	}
	// Should add 's' for pluralization
	expectedResource := "unknownresources"
	if got.Resource != expectedResource {
		t.Errorf("expected resource %q, got %q", expectedResource, got.Resource)
	}
}

func TestGetGVR(t *testing.T) {
	tests := []struct {
		name     string
		manifest *unstructured.Unstructured
		want     schema.GroupVersionResource
	}{
		{
			name: "deployment",
			manifest: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			},
			want: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
		},
		{
			name: "service",
			manifest: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			},
			want: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "services",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGVR(tt.manifest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Group != tt.want.Group {
				t.Errorf("expected group %s, got %s", tt.want.Group, got.Group)
			}
			if got.Version != tt.want.Version {
				t.Errorf("expected version %s, got %s", tt.want.Version, got.Version)
			}
			if got.Resource != tt.want.Resource {
				t.Errorf("expected resource %s, got %s", tt.want.Resource, got.Resource)
			}
		})
	}
}

func TestFormatResourceName(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
		want string
	}{
		{
			name: "namespaced resource",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"metadata": map[string]interface{}{
						"name":      "nginx",
						"namespace": "default",
					},
				},
			},
			want: "Deployment/nginx (default)",
		},
		{
			name: "cluster-scoped resource",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Namespace",
					"metadata": map[string]interface{}{
						"name": "kube-system",
					},
				},
			},
			want: "Namespace/kube-system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatResourceName(tt.obj)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		want    string
	}{
		{
			name:    "success icon",
			success: true,
			want:    "✓",
		},
		{
			name:    "failure icon",
			success: false,
			want:    "✗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusIcon(tt.success)
			if got != tt.want {
				t.Errorf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestDeleteResult(t *testing.T) {
	result := DeleteResult{
		Cluster:   "test-cluster",
		Resource:  "Deployment/nginx",
		Kind:      "Deployment",
		Name:      "nginx",
		Namespace: "default",
		Action:    "deleted",
		Error:     nil,
	}

	if result.Cluster != "test-cluster" {
		t.Errorf("expected cluster test-cluster, got %s", result.Cluster)
	}
	if result.Action != "deleted" {
		t.Errorf("expected action deleted, got %s", result.Action)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
}

func TestParseManifestsFromDir(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		recursive bool
		wantCount int
	}{
		{
			name: "single file",
			files: map[string]string{
				"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
			},
			recursive: false,
			wantCount: 1,
		},
		{
			name: "multiple files",
			files: map[string]string{
				"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				"service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: nginx
`,
			},
			recursive: false,
			wantCount: 2,
		},
		{
			name: "recursive directory",
			files: map[string]string{
				"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				"subdir/service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: nginx
`,
			},
			recursive: true,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				dir := filepath.Dir(filePath)

				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}

				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			manifests, err := parseManifestsFromDir(tmpDir, tt.recursive)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(manifests) != tt.wantCount {
				t.Errorf("expected %d manifests, got %d", tt.wantCount, len(manifests))
			}
		})
	}
}

func TestParseManifests(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		recursive bool
		wantCount int
		wantError bool
	}{
		{
			name: "file path",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "test.yaml")
				content := `apiVersion: v1
kind: Pod
metadata:
  name: test
`
				if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return tmpFile
			},
			recursive: false,
			wantCount: 1,
			wantError: false,
		},
		{
			name: "directory path",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				content := `apiVersion: v1
kind: Pod
metadata:
  name: test
`
				if err := os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return tmpDir
			},
			recursive: false,
			wantCount: 1,
			wantError: false,
		},
		{
			name: "non-existent path",
			setupFunc: func(t *testing.T) string {
				return "/non/existent/path"
			},
			recursive: false,
			wantCount: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			manifests, err := parseManifests(path, tt.recursive)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(manifests) != tt.wantCount {
				t.Errorf("expected %d manifests, got %d", tt.wantCount, len(manifests))
			}
		})
	}
}

func TestResourceTypeCaseInsensitive(t *testing.T) {
	tests := []struct {
		resourceType string
		wantResource string
	}{
		{"Deployment", "deployments"},
		{"DEPLOYMENT", "deployments"},
		{"deployment", "deployments"},
		{"Pod", "pods"},
		{"POD", "pods"},
		{"pod", "pods"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			gvr := getGVRForType(tt.resourceType)
			if gvr.Resource != tt.wantResource {
				t.Errorf("expected resource %q, got %q", tt.wantResource, gvr.Resource)
			}
		})
	}
}

func TestShortFormAliases(t *testing.T) {
	aliases := map[string]string{
		"deploy": "deployments",
		"svc":    "services",
		"cm":     "configmaps",
		"ns":     "namespaces",
		"sa":     "serviceaccounts",
		"ing":    "ingresses",
		"sts":    "statefulsets",
		"ds":     "daemonsets",
		"rs":     "replicasets",
		"pvc":    "persistentvolumeclaims",
	}

	for shortForm, expectedResource := range aliases {
		t.Run(shortForm, func(t *testing.T) {
			gvr := getGVRForType(shortForm)
			if gvr.Resource != expectedResource {
				t.Errorf("short form %q: expected resource %q, got %q",
					shortForm, expectedResource, gvr.Resource)
			}
		})
	}
}
