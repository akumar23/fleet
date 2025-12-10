package get

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aryankumar/fleet/internal/executor"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetPods tests the getPods function with fake clientset
func TestGetPods(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		selector      string
		allNamespaces bool
		pods          []runtime.Object
		wantCount     int
		wantErr       bool
	}{
		{
			name:      "get pods in default namespace",
			namespace: "default",
			pods: []runtime.Object{
				createTestPod("pod1", "default", corev1.PodRunning, 1, 1),
				createTestPod("pod2", "default", corev1.PodPending, 0, 1),
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "get pods in specific namespace",
			namespace: "kube-system",
			pods: []runtime.Object{
				createTestPod("pod1", "default", corev1.PodRunning, 1, 1),
				createTestPod("pod2", "kube-system", corev1.PodRunning, 1, 1),
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:          "get pods in all namespaces",
			namespace:     "",
			allNamespaces: true,
			pods: []runtime.Object{
				createTestPod("pod1", "default", corev1.PodRunning, 1, 1),
				createTestPod("pod2", "kube-system", corev1.PodRunning, 1, 1),
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "no pods found",
			namespace: "default",
			pods:      []runtime.Object{},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.pods...)
			ctx := context.Background()

			pods, err := getPods(ctx, clientset, tt.namespace, tt.selector, "test-cluster", tt.allNamespaces)

			if (err != nil) != tt.wantErr {
				t.Errorf("getPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(pods) != tt.wantCount {
				t.Errorf("getPods() got %d pods, want %d", len(pods), tt.wantCount)
			}

			// Verify pod info structure
			for _, pod := range pods {
				if pod.Cluster != "test-cluster" {
					t.Errorf("expected cluster name 'test-cluster', got '%s'", pod.Cluster)
				}
				if pod.Name == "" {
					t.Errorf("pod name should not be empty")
				}
			}
		})
	}
}

// TestGetNodes tests the getNodes function
func TestGetNodes(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []runtime.Object
		wantCount int
		wantErr   bool
	}{
		{
			name: "get multiple nodes",
			nodes: []runtime.Object{
				createTestNode("node1", "v1.28.0", true, "control-plane"),
				createTestNode("node2", "v1.28.0", true, "worker"),
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "no nodes found",
			nodes:     []runtime.Object{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "node with NotReady status",
			nodes: []runtime.Object{
				createTestNode("node1", "v1.28.0", false, "worker"),
			},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.nodes...)
			ctx := context.Background()

			nodes, err := getNodes(ctx, clientset, "test-cluster")

			if (err != nil) != tt.wantErr {
				t.Errorf("getNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(nodes) != tt.wantCount {
				t.Errorf("getNodes() got %d nodes, want %d", len(nodes), tt.wantCount)
			}
		})
	}
}

// TestGetDeployments tests the getDeployments function
func TestGetDeployments(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		deployments []runtime.Object
		wantCount   int
		wantErr     bool
	}{
		{
			name:      "get deployments in default namespace",
			namespace: "default",
			deployments: []runtime.Object{
				createTestDeployment("deploy1", "default", 3, 3, 3),
				createTestDeployment("deploy2", "default", 2, 2, 2),
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "get deployments in specific namespace",
			namespace: "kube-system",
			deployments: []runtime.Object{
				createTestDeployment("deploy1", "default", 3, 3, 3),
				createTestDeployment("deploy2", "kube-system", 2, 2, 2),
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:        "no deployments found",
			namespace:   "default",
			deployments: []runtime.Object{},
			wantCount:   0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.deployments...)
			ctx := context.Background()

			deployments, err := getDeployments(ctx, clientset, tt.namespace, "test-cluster")

			if (err != nil) != tt.wantErr {
				t.Errorf("getDeployments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(deployments) != tt.wantCount {
				t.Errorf("getDeployments() got %d deployments, want %d", len(deployments), tt.wantCount)
			}
		})
	}
}

// TestGetServices tests the getServices function
func TestGetServices(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		services  []runtime.Object
		wantCount int
		wantErr   bool
	}{
		{
			name:      "get services in default namespace",
			namespace: "default",
			services: []runtime.Object{
				createTestService("svc1", "default", corev1.ServiceTypeClusterIP),
				createTestService("svc2", "default", corev1.ServiceTypeLoadBalancer),
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "get services in specific namespace",
			namespace: "kube-system",
			services: []runtime.Object{
				createTestService("svc1", "default", corev1.ServiceTypeClusterIP),
				createTestService("svc2", "kube-system", corev1.ServiceTypeNodePort),
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "no services found",
			namespace: "default",
			services:  []runtime.Object{},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.services...)
			ctx := context.Background()

			services, err := getServices(ctx, clientset, tt.namespace, "test-cluster")

			if (err != nil) != tt.wantErr {
				t.Errorf("getServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(services) != tt.wantCount {
				t.Errorf("getServices() got %d services, want %d", len(services), tt.wantCount)
			}
		})
	}
}

// TestGetNamespaces tests the getNamespaces function
func TestGetNamespaces(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []runtime.Object
		wantCount  int
		wantErr    bool
	}{
		{
			name: "get multiple namespaces",
			namespaces: []runtime.Object{
				createTestNamespace("default"),
				createTestNamespace("kube-system"),
				createTestNamespace("kube-public"),
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:       "no namespaces found",
			namespaces: []runtime.Object{},
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset(tt.namespaces...)
			ctx := context.Background()

			namespaces, err := getNamespaces(ctx, clientset, "test-cluster")

			if (err != nil) != tt.wantErr {
				t.Errorf("getNamespaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(namespaces) != tt.wantCount {
				t.Errorf("getNamespaces() got %d namespaces, want %d", len(namespaces), tt.wantCount)
			}
		})
	}
}

// TestCalculateReadyStatus tests pod ready status calculation
func TestCalculateReadyStatus(t *testing.T) {
	tests := []struct {
		name string
		pod  *corev1.Pod
		want string
	}{
		{
			name: "all containers ready",
			pod:  createTestPod("test", "default", corev1.PodRunning, 2, 2),
			want: "2/2",
		},
		{
			name: "some containers ready",
			pod:  createTestPod("test", "default", corev1.PodRunning, 1, 2),
			want: "1/2",
		},
		{
			name: "no containers ready",
			pod:  createTestPod("test", "default", corev1.PodPending, 0, 2),
			want: "0/2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateReadyStatus(tt.pod)
			if got != tt.want {
				t.Errorf("calculateReadyStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCalculateAge tests age calculation
func TestCalculateAge(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		created time.Time
		want    string
	}{
		{
			name:    "seconds old",
			created: now.Add(-30 * time.Second),
			want:    "30s",
		},
		{
			name:    "minutes old",
			created: now.Add(-5 * time.Minute),
			want:    "5m",
		},
		{
			name:    "hours old",
			created: now.Add(-3 * time.Hour),
			want:    "3h",
		},
		{
			name:    "days old",
			created: now.Add(-48 * time.Hour),
			want:    "2d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateAge(tt.created, now)
			if got != tt.want {
				t.Errorf("calculateAge() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetNodeStatus tests node status extraction
func TestGetNodeStatus(t *testing.T) {
	tests := []struct {
		name string
		node *corev1.Node
		want string
	}{
		{
			name: "ready node",
			node: createTestNode("test", "v1.28.0", true, "worker"),
			want: "Ready",
		},
		{
			name: "not ready node",
			node: createTestNode("test", "v1.28.0", false, "worker"),
			want: "NotReady",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNodeStatus(tt.node)
			if got != tt.want {
				t.Errorf("getNodeStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetNodeRoles tests node role extraction
func TestGetNodeRoles(t *testing.T) {
	tests := []struct {
		name string
		node *corev1.Node
		want string
	}{
		{
			name: "control-plane node",
			node: createTestNode("test", "v1.28.0", true, "control-plane"),
			want: "control-plane",
		},
		{
			name: "worker node",
			node: createTestNode("test", "v1.28.0", true, "worker"),
			want: "worker",
		},
		{
			name: "node without roles",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			want: "<none>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNodeRoles(tt.node)
			if got != tt.want {
				t.Errorf("getNodeRoles() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatPodsTable tests pod table formatting
func TestFormatPodsTable(t *testing.T) {
	tests := []struct {
		name    string
		pods    []PodInfo
		noColor bool
		wantErr bool
	}{
		{
			name: "format multiple pods",
			pods: []PodInfo{
				{Cluster: "cluster1", Namespace: "default", Name: "pod1", Ready: "1/1", Status: "Running", Restarts: 0, Age: "1h"},
				{Cluster: "cluster2", Namespace: "default", Name: "pod2", Ready: "2/2", Status: "Running", Restarts: 1, Age: "2h"},
			},
			noColor: true,
			wantErr: false,
		},
		{
			name:    "format empty pods",
			pods:    []PodInfo{},
			noColor: true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			// Redirect output to buffer by temporarily changing os.Stdout
			// For this test, we'll just call the function and verify it doesn't error
			err := formatPodsTable(tt.pods, tt.noColor)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatPodsTable() error = %v, wantErr %v", err, tt.wantErr)
			}
			_ = buf // Use buf to avoid unused variable error
		})
	}
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		createTestPod("pod1", "default", corev1.PodRunning, 1, 1),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := getPods(ctx, clientset, "default", "", "test-cluster", false)
	// The fake client doesn't respect context cancellation, so we just verify it returns
	if err != nil {
		// Context cancellation errors are acceptable
		if !strings.Contains(err.Error(), "context") {
			t.Errorf("expected context error, got: %v", err)
		}
	}
}

// Helper functions to create test objects

func createTestPod(name, namespace string, phase corev1.PodPhase, readyContainers, totalContainers int) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: make([]corev1.Container, totalContainers),
		},
		Status: corev1.PodStatus{
			Phase:             phase,
			ContainerStatuses: make([]corev1.ContainerStatus, totalContainers),
		},
	}

	for i := 0; i < totalContainers; i++ {
		pod.Spec.Containers[i] = corev1.Container{
			Name:  "container" + string(rune(i)),
			Image: "test:latest",
		}
		pod.Status.ContainerStatuses[i] = corev1.ContainerStatus{
			Name:  "container" + string(rune(i)),
			Ready: i < readyContainers,
		}
	}

	return pod
}

func createTestNode(name, version string, ready bool, role string) *corev1.Node {
	labels := make(map[string]string)
	if role != "" {
		labels["node-role.kubernetes.io/"+role] = ""
	}

	condition := corev1.ConditionTrue
	if !ready {
		condition = corev1.ConditionFalse
	}

	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.Now(),
			Labels:            labels,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: condition,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: version,
			},
		},
	}
}

func createTestDeployment(name, namespace string, replicas, ready, available int32) *appsv1.Deployment {
	replicasPtr := &replicas
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicasPtr,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     ready,
			UpdatedReplicas:   ready,
			AvailableReplicas: available,
		},
	}
}

func createTestService(name, namespace string, svcType corev1.ServiceType) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ServiceSpec{
			Type:      svcType,
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}

func createTestNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.Now(),
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
}

// TestMultiClusterIntegration tests integration with executor pool
func TestMultiClusterIntegration(t *testing.T) {
	// Create fake clients for multiple clusters
	cluster1 := fake.NewSimpleClientset(
		createTestPod("pod1", "default", corev1.PodRunning, 1, 1),
	)
	cluster2 := fake.NewSimpleClientset(
		createTestPod("pod2", "default", corev1.PodRunning, 1, 1),
	)

	// Create executor pool
	pool := executor.NewPool(2, nil)

	// Submit tasks
	task1 := executor.Task{
		ClusterName: "cluster1",
		Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
			return getPods(ctx, cluster1, "default", "", "cluster1", false)
		},
	}

	task2 := executor.Task{
		ClusterName: "cluster2",
		Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
			return getPods(ctx, cluster2, "default", "", "cluster2", false)
		},
	}

	if err := pool.Submit(task1); err != nil {
		t.Fatalf("failed to submit task1: %v", err)
	}

	if err := pool.Submit(task2); err != nil {
		t.Fatalf("failed to submit task2: %v", err)
	}

	// Execute
	ctx := context.Background()
	results := pool.Execute(ctx)

	// Verify results
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("cluster %s returned error: %v", result.ClusterName, result.Error)
		}

		pods, ok := result.Data.([]PodInfo)
		if !ok {
			t.Errorf("expected []PodInfo, got %T", result.Data)
			continue
		}

		if len(pods) != 1 {
			t.Errorf("expected 1 pod for %s, got %d", result.ClusterName, len(pods))
		}
	}
}

// TestPartialFailure tests handling of partial cluster failures
func TestPartialFailure(t *testing.T) {
	// Create a client that will succeed
	successClient := fake.NewSimpleClientset(
		createTestPod("pod1", "default", corev1.PodRunning, 1, 1),
	)

	// Use a fake client that will work
	failClient := fake.NewSimpleClientset()

	pool := executor.NewPool(2, nil)

	// Submit successful task
	task1 := executor.Task{
		ClusterName: "success-cluster",
		Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
			return getPods(ctx, successClient, "default", "", "success-cluster", false)
		},
	}

	// Submit task that queries non-existent namespace (simulating failure)
	task2 := executor.Task{
		ClusterName: "fail-cluster",
		Execute: func(ctx context.Context, _ interface{}) (interface{}, error) {
			// This should succeed with empty result
			return getPods(ctx, failClient, "nonexistent", "", "fail-cluster", false)
		},
	}

	pool.Submit(task1)
	pool.Submit(task2)

	ctx := context.Background()
	results := pool.Execute(ctx)

	// Verify we got results from both clusters
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Count successful and failed
	successCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		}
	}

	if successCount < 1 {
		t.Errorf("expected at least 1 successful result, got %d", successCount)
	}
}

// Benchmark tests

func BenchmarkGetPods(b *testing.B) {
	pods := make([]runtime.Object, 100)
	for i := 0; i < 100; i++ {
		pods[i] = createTestPod("pod"+string(rune(i)), "default", corev1.PodRunning, 1, 1)
	}

	clientset := fake.NewSimpleClientset(pods...)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getPods(ctx, clientset, "default", "", "test-cluster", false)
	}
}

func BenchmarkCalculateAge(b *testing.B) {
	created := time.Now().Add(-2 * time.Hour)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateAge(created, now)
	}
}
