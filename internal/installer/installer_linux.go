//go:build linux

package installer

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cy-infamous/purewin/internal/core"
)

// InstallerFile represents a detected installer or archive file.
type InstallerFile struct {
	Path      string    // Full path to the file
	Name      string    // File name only
	Size      int64     // Size in bytes
	Extension string    // File extension (.deb, .rpm, .AppImage, etc.)
	Source    string    // Source location (Downloads, Desktop, etc.)
	ModTime   time.Time // Last modification time
}

// scanLocation represents a directory to scan for installer files.
type scanLocation struct {
	Path        string // Directory path
	SourceLabel string // User-facing label
}

// GetScanLocations returns all locations to scan for installer files.
func GetScanLocations() []scanLocation {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}

	locations := []scanLocation{
		{Path: filepath.Join(home, "Downloads"), SourceLabel: "Downloads"},
		{Path: filepath.Join(home, "Desktop"), SourceLabel: "Desktop"},
		{Path: os.TempDir(), SourceLabel: "Temp"},
	}

	// Flatpak cache
	flatpakCache := filepath.Join(home, ".local", "share", "flatpak")
	if _, err := os.Stat(flatpakCache); err == nil {
		locations = append(locations, scanLocation{
			Path:        flatpakCache,
			SourceLabel: "Flatpak",
		})
	}

	// apt cache
	aptCache := "/var/cache/apt/archives"
	if _, err := os.Stat(aptCache); err == nil {
		locations = append(locations, scanLocation{
			Path:        aptCache,
			SourceLabel: "APT",
		})
	}

	// dnf/yum cache
	dnfCache := "/var/cache/dnf"
	if _, err := os.Stat(dnfCache); err == nil {
		locations = append(locations, scanLocation{
			Path:        dnfCache,
			SourceLabel: "DNF",
		})
	}

	return locations
}

// ScanInstallers scans for installer files matching the criteria.
// minAge is in days (0 = no age filter)
// minSize is in bytes (0 = no size filter)
func ScanInstallers(minAge int, minSize int64) ([]InstallerFile, error) {
	locations := GetScanLocations()
	var files []InstallerFile

	cutoffTime := time.Time{}
	if minAge > 0 {
		cutoffTime = time.Now().Add(-time.Duration(minAge) * 24 * time.Hour)
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc.Path); os.IsNotExist(err) {
			continue
		}

		err := scanLocationForInstallers(loc.Path, loc.SourceLabel, minSize, cutoffTime, &files)
		if err != nil {
			// Non-fatal: continue scanning other locations
			continue
		}
	}

	return files, nil
}

// scanLocationForInstallers scans a single location for installer files.
func scanLocationForInstallers(path, sourceLabel string, minSize int64, cutoffTime time.Time, files *[]InstallerFile) error {
	return scanDirectoryForInstallers(path, sourceLabel, minSize, cutoffTime, files)
}

// scanDirectoryForInstallers scans a directory (non-recursively) for installer files.
func scanDirectoryForInstallers(path, sourceLabel string, minSize int64, cutoffTime time.Time, files *[]InstallerFile) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Apply size filter
		if minSize > 0 && info.Size() < minSize {
			continue
		}

		// Apply age filter
		if !cutoffTime.IsZero() && info.ModTime().After(cutoffTime) {
			continue
		}

		// Check if file matches our criteria
		ext := strings.ToLower(filepath.Ext(entry.Name()))

		isInstaller := false
		switch ext {
		case ".deb", ".rpm", ".appimage", ".snap", ".flatpakref",
			".bundle", ".run", ".sh", ".bin":
			isInstaller = true
		case ".tar.gz", ".tgz", ".tar.xz", ".tar.bz2", ".zip", ".7z":
			// Only include archives if they're large (>50MB)
			if info.Size() > 50*1024*1024 {
				isInstaller = true
			}
		}

		if !isInstaller {
			continue
		}

		file := InstallerFile{
			Path:      filepath.Join(path, entry.Name()),
			Name:      entry.Name(),
			Size:      info.Size(),
			Extension: ext,
			Source:    sourceLabel,
			ModTime:   info.ModTime(),
		}

		*files = append(*files, file)
	}

	return nil
}

// CleanInstallers deletes the specified installer files.
// Returns total bytes freed, number of files deleted, and any error.
func CleanInstallers(files []InstallerFile, dryRun bool) (int64, int, error) {
	var totalBytes int64
	var totalCount int
	var lastErr error

	for _, file := range files {
		freed, err := core.SafeDelete(file.Path, dryRun)
		if err != nil {
			lastErr = err
			continue
		}
		totalBytes += freed
		totalCount++
	}

	return totalBytes, totalCount, lastErr
}

// GroupBySource groups installer files by their source location.
func GroupBySource(files []InstallerFile) map[string][]InstallerFile {
	groups := make(map[string][]InstallerFile)
	for _, file := range files {
		groups[file.Source] = append(groups[file.Source], file)
	}
	return groups
}

// GetTotalSize calculates the total size of all files.
func GetTotalSize(files []InstallerFile) int64 {
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return total
}
