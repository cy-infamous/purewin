//go:build linux

package clean

// ScanRecycleBin returns 0 on Linux — there is no Windows Recycle Bin.
func ScanRecycleBin() (int64, error) {
	return 0, nil
}

// EmptyRecycleBin is a no-op on Linux.
func EmptyRecycleBin(dryRun bool) error {
	return nil
}
