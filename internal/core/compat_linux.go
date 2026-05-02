//go:build linux

package core

// GetWindowsVersion returns dummy values on Linux.
// Version checks are irrelevant outside Windows.
func GetWindowsVersion() (major, minor, build uint32) {
	return 10, 0, 0
}

// IsWindows10Plus always returns true on Linux.
func IsWindows10Plus() bool {
	return true
}
