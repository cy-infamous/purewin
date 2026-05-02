//go:build linux

package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// SafeDelete removes a file or directory after safety validation.
// In dryRun mode, it calculates and returns the size without deleting.
func SafeDelete(path string, dryRun bool) (int64, error) {
	if err := ValidatePath(path); err != nil {
		return 0, fmt.Errorf("safety check failed for %s: %w", path, err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("cannot stat %s: %w", path, err)
	}

	var size int64
	if info.IsDir() {
		size, err = GetDirSize(path)
		if err != nil {
			size = 0
		}
	} else {
		size = info.Size()
	}

	if dryRun {
		return size, nil
	}

	if info.IsDir() {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to delete %s: %w", path, err)
	}
	return size, nil
}

// SafeDeleteWithWhitelist removes a file or directory after checking
// the user's whitelist and then performing safety validation.
func SafeDeleteWithWhitelist(path string, dryRun bool, isWhitelisted func(string) bool) (int64, error) {
	if isWhitelisted != nil && isWhitelisted(path) {
		return 0, fmt.Errorf("path is whitelisted and will be skipped: %s", path)
	}
	return SafeDelete(path, dryRun)
}

// SafeCleanDir removes files matching a glob pattern within a directory.
func SafeCleanDir(dir string, pattern string, dryRun bool) (int64, int, error) {
	if err := ValidatePath(dir); err != nil {
		return 0, 0, fmt.Errorf("safety check failed for %s: %w", dir, err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("cannot stat directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("not a directory: %s", dir)
	}

	globPattern := filepath.Join(dir, pattern)
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid glob pattern %s: %w", globPattern, err)
	}

	var totalBytes int64
	var totalFiles int

	for _, match := range matches {
		freed, delErr := SafeDelete(match, dryRun)
		if delErr != nil {
			continue
		}
		totalBytes += freed
		totalFiles++
	}

	return totalBytes, totalFiles, nil
}

// GetDirSize calculates the total size of all files in a directory tree.
func GetDirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr != nil {
				return nil
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("error walking directory %s: %w", path, err)
	}
	return total, nil
}

// GetFileSize returns the size of a single file.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("cannot stat file %s: %w", path, err)
	}
	if info.IsDir() {
		return 0, fmt.Errorf("path is a directory, use GetDirSize: %s", path)
	}
	return info.Size(), nil
}

// FormatSize returns a human-readable representation of a byte count.
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
