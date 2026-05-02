//go:build linux

package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lakshaymaurya-felt/purewin/internal/core"
	"github.com/lakshaymaurya-felt/purewin/pkg/whitelist"
)

// ScanNonSystemDrives scans non-root mount points for temp files.
func ScanNonSystemDrives(wl *whitelist.Whitelist) []CleanItem {
	// On Linux, scan /tmp and /var/tmp on non-root mounts.
	// For simplicity, return nil — these are covered by config targets.
	return nil
}

// ScanBrowserCaches is handled by config-driven targets on Linux.
func ScanBrowserCaches(wl *whitelist.Whitelist) []CleanItem {
	return nil
}

// ScanDevCaches is handled by config-driven targets on Linux.
func ScanDevCaches(wl *whitelist.Whitelist) []CleanItem {
	return nil
}

// ScanMemoryDumps returns nil on Linux (no Windows memory dumps).
func ScanMemoryDumps() []CleanItem {
	return nil
}

// ScanWERUserReports returns nil on Linux (no Windows Error Reporting).
func ScanWERUserReports(wl *whitelist.Whitelist) []CleanItem {
	return nil
}

// WindowsOldSize returns 0 on Linux.
func WindowsOldSize() int64 {
	return 0
}

// CleanWindowsOld returns 0 on Linux.
func CleanWindowsOld(dryRun bool) (int64, error) {
	return 0, nil
}

// GoModCacheSize returns the size of the Go module cache.
func GoModCacheSize() int64 {
	cacheDir := goModCachePath()
	if cacheDir == "" {
		return 0
	}
	size, err := core.GetDirSize(cacheDir)
	if err != nil {
		return 0
	}
	return size
}

// CleanGoModCache runs `go clean -modcache`.
func CleanGoModCache(dryRun bool) (int64, error) {
	if _, err := exec.LookPath("go"); err != nil {
		return 0, nil
	}
	cacheDir := goModCachePath()
	if cacheDir == "" {
		return 0, nil
	}
	size, _ := core.GetDirSize(cacheDir)
	if dryRun {
		return size, nil
	}
	cmd := exec.Command("go", "clean", "-modcache")
	if _, err := cmd.CombinedOutput(); err != nil {
		return 0, err
	}
	return size, nil
}

func goModCachePath() string {
	if _, err := exec.LookPath("go"); err == nil {
		cmd := exec.Command("go", "env", "GOMODCACHE")
		output, err := cmd.Output()
		if err == nil {
			modCache := strings.TrimSpace(string(output))
			if modCache != "" {
				if _, err := os.Stat(modCache); err == nil {
					return modCache
				}
			}
		}
	}
	if modCache := os.Getenv("GOMODCACHE"); modCache != "" {
		if _, err := os.Stat(modCache); err == nil {
			return modCache
		}
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}
	cacheDir := filepath.Join(gopath, "pkg", "mod", "cache")
	if _, err := os.Stat(cacheDir); err == nil {
		return cacheDir
	}
	return ""
}

// IsDockerAvailable returns true if the docker CLI is on PATH.
func IsDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// DockerBuildCacheSize returns Docker build cache size on Linux.
func DockerBuildCacheSize() int64 {
	if _, err := exec.LookPath("docker"); err != nil {
		return 0
	}
	cmd := exec.Command("docker", "system", "df", "--format", "{{.Type}}\t{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Build Cache") {
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				return parseDockerSize(strings.TrimSpace(parts[1]))
			}
		}
	}
	return 0
}

func parseDockerSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" || sizeStr == "0B" {
		return 0
	}
	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "B") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
	}
	var value float64
	if _, err := fmt.Sscanf(sizeStr, "%f", &value); err != nil {
		return 0
	}
	return int64(value * float64(multiplier))
}

// CleanDockerBuildCache runs docker builder prune on Linux.
func CleanDockerBuildCache(dryRun bool) (int64, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return 0, nil
	}
	if dryRun {
		return DockerBuildCacheSize(), nil
	}
	cmd := exec.Command("docker", "builder", "prune", "-af")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}
	_ = output
	return 0, nil
}
