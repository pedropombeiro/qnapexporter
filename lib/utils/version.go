package utils

// Build-time information about the binary. These values are overridden at
// build time via -ldflags and default to placeholder values for local builds.
var (
	REVISION = "HEAD"
	BRANCH   = "HEAD"
	BUILT    = "unknown"
	VERSION  = "dev"
)
