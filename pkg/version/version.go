package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set at build time via ldflags)
	Version = "dev"
	// Commit is the git commit hash (set at build time via ldflags)
	Commit = "unknown"
	// BuildTime is the build timestamp (set at build time via ldflags)
	BuildTime = "unknown"
	// GoVersion is the Go version used to build (set at build time via ldflags)
	GoVersion = runtime.Version()
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("Fleet CLI\n  Version:    %s\n  Commit:     %s\n  Build Time: %s\n  Go Version: %s\n  Platform:   %s",
		i.Version, i.Commit, i.BuildTime, i.GoVersion, i.Platform)
}

// JSON returns version info as JSON string
func (i Info) JSON() (string, error) {
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
