package version

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.Commit == "" {
		t.Error("Commit should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("Platform = %s, want %s", info.Platform, expectedPlatform)
	}
}

func TestString(t *testing.T) {
	info := Get()
	output := info.String()

	if !strings.Contains(output, "Fleet CLI") {
		t.Error("String output should contain 'Fleet CLI'")
	}

	if !strings.Contains(output, info.Version) {
		t.Errorf("String output should contain version %s", info.Version)
	}

	if !strings.Contains(output, info.Commit) {
		t.Errorf("String output should contain commit %s", info.Commit)
	}
}

func TestJSON(t *testing.T) {
	info := Get()
	jsonStr, err := info.JSON()

	if err != nil {
		t.Fatalf("JSON() returned error: %v", err)
	}

	// Verify it's valid JSON by unmarshaling
	var result map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	// Check key fields exist
	if result["version"] != info.Version {
		t.Errorf("JSON version = %s, want %s", result["version"], info.Version)
	}

	if result["commit"] != info.Commit {
		t.Errorf("JSON commit = %s, want %s", result["commit"], info.Commit)
	}
}
