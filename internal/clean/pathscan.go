package clean

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cy-infamous/purewin/pkg/whitelist"
)

// ─── Junk Pattern Definitions ────────────────────────────────────────────────

// junkCategory groups related junk patterns under a label.
type junkCategory struct {
	Name        string
	Label       string // Human-readable display label.
	Extensions  []string
	ExactNames  []string
	DirNames    []string // Entire directories to flag as junk.
	Prefixes    []string // Filename prefixes (e.g., "~$" for Office temp files).
}

// getJunkCategories returns the set of junk file/directory patterns to scan for.
func getJunkCategories() []junkCategory {
	return []junkCategory{
		{
			Name:  "temp",
			Label: "Temporary Files",
			Extensions: []string{
				".tmp", ".temp", ".bak", ".old", ".orig",
				".swp", ".swo", // Vim swap files.
			},
			Prefixes: []string{
				"~$", // Microsoft Office temp files.
				"~",  // Generic temp prefix.
			},
		},
		{
			Name:  "logs",
			Label: "Log Files",
			Extensions: []string{
				".log",
			},
		},
		{
			Name:  "cache",
			Label: "Cache Files",
			Extensions: []string{
				".cache",
			},
			DirNames: []string{
				"__pycache__",
				".cache",
				".sass-cache",
				".parcel-cache",
			},
		},
		{
			Name:  "build",
			Label: "Build Artifacts",
			DirNames: []string{
				"node_modules",
				".next",
				".nuxt",
				"dist",
				"build",
				"bin",
				"obj",   // .NET build output.
				".vs",   // Visual Studio local settings.
				".idea", // JetBrains IDE settings.
			},
		},
		{
			Name:  "os_junk",
			Label: "OS-Generated Junk",
			ExactNames: []string{
				"thumbs.db",
				"desktop.ini",
				".ds_store",
				"$recycle.bin",
			},
		},
		{
			Name:  "debug",
			Label: "Debug & Crash Files",
			Extensions: []string{
				".dmp",  // Crash dumps.
				".mdmp", // Minidumps.
				".pdb",  // Debug symbols (can be large).
			},
		},
	}
}

// ─── Build Artifacts Safety ─────────────────────────────────────────────────

// buildArtifactDirs are directory names that are only flagged if they appear
// to be build output (i.e., they have a project indicator file next to them).
var buildArtifactDirs = map[string]bool{
	"dist":  true,
	"build": true,
	"bin":   true,
	"obj":   true,
}

// projectIndicators are files that, when found alongside a "build artifact"
// directory, confirm it's a project build output rather than a system directory.
var projectIndicators = []string{
	"package.json",   // Node.js
	"go.mod",         // Go
	"Cargo.toml",     // Rust
	"pom.xml",        // Maven
	"build.gradle",   // Gradle
	"CMakeLists.txt", // CMake
	"Makefile",       // Make
	"*.csproj",       // .NET
	"*.sln",          // Visual Studio
	"pyproject.toml", // Python
	"setup.py",       // Python
	"composer.json",  // PHP
	"Gemfile",        // Ruby
}

// hasProjectIndicator checks if the parent directory of a path contains
// a file that indicates it's a software project.
func hasProjectIndicator(parentDir string) bool {
	for _, indicator := range projectIndicators {
		if strings.ContainsRune(indicator, '*') {
			// Glob pattern.
			matches, err := filepath.Glob(filepath.Join(parentDir, indicator))
			if err == nil && len(matches) > 0 {
				return true
			}
		} else {
			if _, err := os.Stat(filepath.Join(parentDir, indicator)); err == nil {
				return true
			}
		}
	}
	return false
}

// ─── Path-Based Scanner ─────────────────────────────────────────────────────

// PathScanResult holds results for a single junk category found during a path scan.
type PathScanResult struct {
	Category  string      // Category key (temp, logs, cache, build, os_junk, debug).
	Label     string      // Human-readable label.
	Items     []CleanItem // Discovered items.
	TotalSize int64
	ItemCount int
}

// ScanPath walks the given directory tree and identifies junk files/directories
// matching known patterns. It respects the whitelist and skips inaccessible
// entries. The maxDepth parameter limits how deep to recurse (0 = unlimited).
func ScanPath(root string, wl *whitelist.Whitelist, maxDepth int) []PathScanResult {
	categories := getJunkCategories()

	// Pre-build lookup maps for fast matching.
	extToCategory := make(map[string]int)    // extension -> category index
	nameToCategory := make(map[string]int)   // lowercase exact name -> category index
	dirToCategory := make(map[string]int)    // lowercase dir name -> category index
	prefixToCategory := make(map[string]int) // prefix -> category index

	for i, cat := range categories {
		for _, ext := range cat.Extensions {
			extToCategory[strings.ToLower(ext)] = i
		}
		for _, name := range cat.ExactNames {
			nameToCategory[strings.ToLower(name)] = i
		}
		for _, dir := range cat.DirNames {
			dirToCategory[strings.ToLower(dir)] = i
		}
		for _, pfx := range cat.Prefixes {
			prefixToCategory[pfx] = i
		}
	}

	// Collect items per category.
	buckets := make([][]CleanItem, len(categories))

	rootClean := filepath.Clean(root)
	rootDepth := strings.Count(rootClean, string(os.PathSeparator))

	_ = filepath.WalkDir(rootClean, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible.
		}

		// Skip the root itself.
		if path == rootClean {
			return nil
		}

		// Enforce max depth.
		if maxDepth > 0 {
			pathDepth := strings.Count(path, string(os.PathSeparator))
			if pathDepth-rootDepth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip whitelisted paths.
		if wl != nil && wl.IsWhitelisted(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		nameLower := strings.ToLower(name)

		// ── Directory matching ──────────────────────────────────────────
		if d.IsDir() {
			if catIdx, ok := dirToCategory[nameLower]; ok {
				// For generic build artifact dirs, only match if in a project.
				if buildArtifactDirs[nameLower] {
					parentDir := filepath.Dir(path)
					if !hasProjectIndicator(parentDir) {
						return nil // Not a project dir — don't flag, but keep walking.
					}
				}

				dirSize := dirSize(path)
				if dirSize > 0 {
					buckets[catIdx] = append(buckets[catIdx], CleanItem{
						Path:        path,
						Size:        dirSize,
						Category:    categories[catIdx].Name,
						Description: categories[catIdx].Label,
					})
				}
				return filepath.SkipDir // Don't walk inside flagged directories.
			}

			// Check exact-name match for dirs (e.g., $Recycle.Bin).
			if _, ok := nameToCategory[nameLower]; ok {
				return filepath.SkipDir // Skip OS junk dirs entirely.
			}

			return nil
		}

		// ── File matching ───────────────────────────────────────────────
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}

		// Exact name match.
		if catIdx, ok := nameToCategory[nameLower]; ok {
			buckets[catIdx] = append(buckets[catIdx], CleanItem{
				Path:        path,
				Size:        info.Size(),
				Category:    categories[catIdx].Name,
				Description: categories[catIdx].Label,
			})
			return nil
		}

		// Extension match.
		ext := strings.ToLower(filepath.Ext(name))
		if ext != "" {
			if catIdx, ok := extToCategory[ext]; ok {
				buckets[catIdx] = append(buckets[catIdx], CleanItem{
					Path:        path,
					Size:        info.Size(),
					Category:    categories[catIdx].Name,
					Description: categories[catIdx].Label,
				})
				return nil
			}
		}

		// Prefix match.
		for pfx, catIdx := range prefixToCategory {
			if strings.HasPrefix(name, pfx) {
				buckets[catIdx] = append(buckets[catIdx], CleanItem{
					Path:        path,
					Size:        info.Size(),
					Category:    categories[catIdx].Name,
					Description: categories[catIdx].Label,
				})
				return nil
			}
		}

		return nil
	})

	// Build results for non-empty categories.
	var results []PathScanResult
	for i, items := range buckets {
		if len(items) == 0 {
			continue
		}
		var total int64
		for _, item := range items {
			total += item.Size
		}
		results = append(results, PathScanResult{
			Category:  categories[i].Name,
			Label:     categories[i].Label,
			Items:     items,
			TotalSize: total,
			ItemCount: len(items),
		})
	}

	return results
}

// dirSize calculates the total size of all files in a directory tree.
// Returns 0 on any error.
func dirSize(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

// PathScanTotalSize returns the combined size across all path scan results.
func PathScanTotalSize(results []PathScanResult) int64 {
	var total int64
	for _, r := range results {
		total += r.TotalSize
	}
	return total
}

// PathScanTotalItems returns the combined item count across all path scan results.
func PathScanTotalItems(results []PathScanResult) int {
	var total int
	for _, r := range results {
		total += r.ItemCount
	}
	return total
}
