//go:build windows

package optimize

// Task labels for Windows display in the optimize command.
var (
	TaskLabelDISM      = "DISM component cleanup"
	TaskLabelSFC       = "System file integrity check"
	TaskLabelIconCache = "Rebuild icon cache"
	TaskLabelSearch    = "Rebuild search index"
	TaskLabelEventLogs = "Clear event logs"
)
