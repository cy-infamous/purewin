//go:build linux

package optimize

// Task labels for Linux-appropriate display in the optimize command.
var (
	TaskLabelDISM      = "Package cache cleanup"
	TaskLabelSFC       = "SSD TRIM optimization"
	TaskLabelIconCache = "Rebuild icon cache"
	TaskLabelSearch    = "Vacuum journal logs"
	TaskLabelEventLogs = "Remove orphaned packages"
)
