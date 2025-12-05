// Package version provides version information for the application.
package version

// These variables are set at build time using -ldflags
var (
	// Version is the semantic version (e.g., v1.0.0)
	Version = "dev"

	// BuildTime is the time the binary was built
	BuildTime = "unknown"

	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// Info returns version information as a map
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
	}
}
