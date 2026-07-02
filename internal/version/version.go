// Package version exposes build-time metadata that is injected via -ldflags.
package version

// These values are overridden at build time, e.g.:
//
//	go build -ldflags "-X devops-showcase/internal/version.Version=1.2.3 \
//	  -X devops-showcase/internal/version.Commit=$(git rev-parse --short HEAD) \
//	  -X devops-showcase/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	// Version is the semantic version of the build (git tag or "dev").
	Version = "dev"
	// Commit is the short git SHA the binary was built from.
	Commit = "none"
	// BuildDate is the UTC RFC3339 timestamp of the build.
	BuildDate = "unknown"
)

// Info is a serialisable snapshot of the build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
}

// Get returns the current build metadata.
func Get() Info {
	return Info{Version: Version, Commit: Commit, BuildDate: BuildDate}
}
