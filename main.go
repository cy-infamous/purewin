package main

import (
	"os"

	"github.com/cy-infamous/purewin/cmd"
	"github.com/cy-infamous/purewin/internal/update"
)

// Version info set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Clean up .old binary from a previous self-update, if present.
	update.CleanupOldBinary()

	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
