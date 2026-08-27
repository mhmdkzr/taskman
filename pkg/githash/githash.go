// Package githash exposes the main binary's VCS revision recorded by the Go toolchain.
// It returns an empty value when build VCS stamping is unavailable or disabled.
package githash

import "runtime/debug"

var commitHash = getCommitHash()

func getCommitHash() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return revision(info)
}

// revision returns the VCS revision recorded in the build info, or "" when absent.
func revision(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}

// GetCommitHash returns the revision of the service binary's main repository.
// When this package is consumed as a dependency, the value still describes the
// consuming service binary rather than the libs module.
func GetCommitHash() string {
	return commitHash
}
