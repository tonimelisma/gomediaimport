package main

import "runtime/debug"

// version is stamped by the release workflow with -ldflags "-X main.version=vX.Y.Z".
var version string

// resolveVersion returns the stamped version, or, when nothing was stamped, the module version
// Go records in the build info: `go install ...@vX.Y.Z` reports that tag, a plain `go build` of
// a checkout reports (devel).
func resolveVersion(stamped string) string {
	if stamped != "" {
		return stamped
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}
