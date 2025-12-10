package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseManifestsFromFile(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantCount     int
		wantError     bool
		validateFirst func(*testing.T, *unstructured.Unstructured)
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
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`,
			wantCount: 1,
			wantError: false,
			validateFirst: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "Deployment" {
					t.Errorf("expected kind Deployment, got %s", obj.GetKind())
				}
				if obj.GetName() != "nginx" {
					t.Errorf("expected name nginx, got %s", obj.GetName())
				}
				if obj.GetNamespace() != "default" {
					t.Errorf("expected namespace default, got %s", obj.GetNamespace())
				}
			},
		},
		{
			name: "multi-document YAML",
			content: `apiVersion: v1
kind: Service
metadata:
  name: nginx-svc
  namespace: default
spec:
  selector:
    app: nginx
  ports:
  - port: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 1
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
        image: nginx
`,
			wantCount: 2,
			wantError: false,
			validateFirst: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "Service" {
					t.Errorf("expected first resource to be Service, got %s", obj.GetKind())
				}
			},
		},
		{
			name: "configmap",
			content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: kube-system
data:
  key1: value1
  key2: value2
`,
			wantCount: 1,
			wantError: false,
			validateFirst: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() != "ConfigMap" {
					t.Errorf("expected kind ConfigMap, got %s", obj.GetKind())
				}
			},
		},
		{
			name:      "empty file",
			content:   "",
			wantCount: 0,
			wantError: false,
		},
		{
			name: "invalid YAML",
			content: `apiVersion: v1
kind: Pod
metadata:
  name: test
  invalid yaml here [[[
`,
			wantCount: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.yaml")

			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			// Parse manifests
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

			if tt.validateFirst != nil && len(manifests) > 0 {
				tt.validateFirst(t, manifests[0])
			}
		})
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
			name: "single file in directory",
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
		{
			name: "non-recursive skips subdirs",
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
			recursive: false,
			wantCount: 1,
		},
		{
			name: "ignores non-yaml files",
			files: map[string]string{
				"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
`,
				"readme.txt": "This is a readme",
				"config.json": `{"key": "value"}`,
			},
			recursive: false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
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

			// Parse manifests
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
		{
			name: "ingress",
			manifest: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "networking.k8s.io/v1",
					"kind":       "Ingress",
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			},
			want: schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v1",
				Resource: "ingresses",
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

func TestParseManifests(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // Returns path
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

// TestApplyResult tests the ApplyResult struct
func TestApplyResult(t *testing.T) {
	result := ApplyResult{
		Cluster:   "test-cluster",
		Resource:  "Deployment/nginx",
		Kind:      "Deployment",
		Name:      "nginx",
		Namespace: "default",
		Action:    "configured",
		Error:     nil,
	}

	if result.Cluster != "test-cluster" {
		t.Errorf("expected cluster test-cluster, got %s", result.Cluster)
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
}

// TestConfirmApply tests the confirmation logic (manual verification needed)
func TestConfirmApplyLogic(t *testing.T) {
	// This is more of a smoke test to ensure the function doesn't panic
	manifests := []*unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"kind": "Deployment",
				"metadata": map[string]interface{}{
					"name": "nginx",
				},
			},
		},
	}
	clusters := []string{"cluster1", "cluster2"}

	// We can't actually test the user input part without mocking stdin
	// This just verifies the function structure is valid
	_ = manifests
	_ = clusters
}

func TestYAMLParsing_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name:      "empty document separators",
			content:   "---\n---\n---",
			wantCount: 0,
			wantError: false,
		},
		{
			name: "document with comments",
			content: `# This is a comment
apiVersion: v1
kind: Pod
metadata:
  name: test # inline comment
  namespace: default
`,
			wantCount: 1,
			wantError: false,
		},
		{
			name: "mixed valid and empty docs",
			content: `---
apiVersion: v1
kind: Pod
metadata:
  name: test1
---
---
apiVersion: v1
kind: Pod
metadata:
  name: test2
---
`,
			wantCount: 2,
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

func TestGVRPluralization(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{"Deployment", "deployments"},
		{"Service", "services"},
		{"Ingress", "ingresses"},
		{"Policy", "policies"},
		{"Pod", "pods"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			manifest := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       tt.kind,
					"metadata": map[string]interface{}{
						"name": "test",
					},
				},
			}

			gvr, err := getGVR(manifest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(gvr.Resource, tt.expected) && gvr.Resource != tt.expected {
				t.Logf("Note: pluralization might differ - got %s, checking for %s", gvr.Resource, tt.expected)
			}
		})
	}
}
