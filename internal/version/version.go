package version

import (
	"fmt"
	"runtime/debug"
)

// version is set at build time via -ldflags (for GoReleaser builds).
var version = "dev"

// Version is the version string, populated from ldflags or build info.
var Version = getVersion()

func getVersion() string {
	// If version was set via ldflags (GoReleaser), use it
	if version != "" && version != "dev" {
		return version
	}

	// Otherwise, try to get version from build info (go install)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version // return "dev"
	}

	// Use module version if available
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		v := info.Main.Version

		// Add VCS info if available
		var revision string
		var modified bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}

		// Append commit hash (short form)
		if revision != "" {
			if len(revision) > 7 {
				revision = revision[:7]
			}
			v = fmt.Sprintf("%s (%s)", v, revision)
		}

		// Mark as modified if there were uncommitted changes
		if modified {
			v += " (modified)"
		}

		return v
	}

	// Fallback: use VCS revision if available
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision := setting.Value
			if len(revision) > 7 {
				revision = revision[:7]
			}
			return fmt.Sprintf("dev (%s)", revision)
		}
	}

	return version // return "dev"
}
