package clean

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cy-infamous/purewin/internal/core"
	"github.com/cy-infamous/purewin/internal/ui"
	"github.com/cy-infamous/purewin/pkg/whitelist"
)

const (
	// serviceCommandTimeout is the maximum time to wait for net stop/start.
	serviceCommandTimeout = 60 * time.Second
)

// systemRoot returns the Windows directory from the environment.
func systemRoot() string {
	if sr := os.Getenv("SystemRoot"); sr != "" {
		return sr
	}
	return `C:\Windows`
}

// programData returns the ProgramData directory from the environment.
func programData() string {
	if pd := os.Getenv("ProgramData"); pd != "" {
		return pd
	}
	return `C:\ProgramData`
}

// systemDrive returns the system drive root (e.g. "C:\").
func systemDrive() string {
	sr := systemRoot()
	if len(sr) >= 3 {
		return sr[:3]
	}
	return `C:\`
}

// ─── System Cache Scanning ───────────────────────────────────────────────────

// ScanSystemCaches scans system-level caches that require admin privileges.
// Returns nil immediately if the process is not elevated.
func ScanSystemCaches(wl *whitelist.Whitelist) []CleanItem {
	if !core.IsElevated() {
		return nil
	}

	type systemTarget struct {
		name        string
		paths       []string
		description string
	}

	sr := systemRoot()
	pd := programData()

	targets := []systemTarget{
		{
			name:        "WindowsTemp",
			paths:       []string{filepath.Join(sr, "Temp")},
			description: "System temporary files",
		},
		{
			name:        "WUCache",
			paths:       []string{filepath.Join(sr, "SoftwareDistribution", "Download")},
			description: "Windows Update download cache",
		},
		{
			name:        "CBSLogs",
			paths:       []string{filepath.Join(sr, "Logs", "CBS")},
			description: "CBS servicing logs",
		},
		{
			name:        "DISMLogs",
			paths:       []string{filepath.Join(sr, "Logs", "DISM")},
			description: "DISM operation logs",
		},
		{
			name: "WERReports",
			paths: []string{
				filepath.Join(pd, "Microsoft", "Windows", "WER", "ReportQueue"),
				filepath.Join(pd, "Microsoft", "Windows", "WER", "Temp"),
			},
			description: "Windows Error Reporting",
		},
		{
			name:        "DeliveryOptimization",
			paths:       []string{filepath.Join(sr, "SoftwareDistribution", "DeliveryOptimization")},
			description: "Delivery Optimization cache",
		},
	}

	var items []CleanItem
	for _, t := range targets {
		for _, p := range t.paths {
			if _, err := os.Stat(p); err != nil {
				continue
			}
			if wl != nil && wl.IsWhitelisted(p) {
				continue
			}
			dirItems := scanDirectory(p, "system", t.description, wl)
			items = append(items, dirItems...)
		}
	}

	return items
}

// ─── Memory Dumps ────────────────────────────────────────────────────────────

// ScanMemoryDumps scans for kernel and minidump crash files.
// Returns nil if not elevated.
func ScanMemoryDumps() []CleanItem {
	if !core.IsElevated() {
		return nil
	}

	sr := systemRoot()
	var items []CleanItem

	// Full memory dump.
	memDump := filepath.Join(sr, "MEMORY.DMP")
	if info, err := os.Stat(memDump); err == nil {
		items = append(items, CleanItem{
			Path:        memDump,
			Size:        info.Size(),
			Category:    "system",
			Description: "Kernel memory dump",
		})
	}

	// Minidumps.
	minidumpDir := filepath.Join(sr, "Minidump")
	if _, err := os.Stat(minidumpDir); err == nil {
		dirItems := scanDirectory(minidumpDir, "system", "Minidump crash files", nil)
		items = append(items, dirItems...)
	}

	return items
}

// CleanMemoryDumps removes kernel and minidump crash files.
// Returns total bytes freed. Requires admin privileges.
func CleanMemoryDumps(dryRun bool) (int64, error) {
	if !core.IsElevated() {
		return 0, fmt.Errorf("cleaning memory dumps requires administrator privileges")
	}

	sr := systemRoot()
	var totalFreed int64

	// Full memory dump.
	memDump := filepath.Join(sr, "MEMORY.DMP")
	freed, err := core.SafeDelete(memDump, dryRun)
	if err == nil {
		totalFreed += freed
	}

	// Minidumps.
	minidumpDir := filepath.Join(sr, "Minidump")
	freed, _, err = core.SafeCleanDir(minidumpDir, "*", dryRun)
	if err == nil {
		totalFreed += freed
	}

	return totalFreed, nil
}

// ─── Windows Update Cache ────────────────────────────────────────────────────

// CleanWindowsUpdate stops the Windows Update service, cleans the download
// cache, and restarts the service. Requires admin privileges.
func CleanWindowsUpdate(dryRun bool) (int64, error) {
	if !core.IsElevated() {
		return 0, fmt.Errorf("cleaning Windows Update cache requires administrator privileges")
	}

	downloadDir := filepath.Join(systemRoot(), "SoftwareDistribution", "Download")

	// Calculate size first.
	size, _ := core.GetDirSize(downloadDir)

	if dryRun {
		return size, nil
	}

	// Stop Windows Update service.
	if err := runServiceCommand("stop", "wuauserv"); err != nil {
		return 0, fmt.Errorf("failed to stop wuauserv: %w", err)
	}

	// Clean the download cache.
	freed, _, cleanErr := core.SafeCleanDir(downloadDir, "*", false)

	// Always restart the service, even if cleaning failed.
	if restartErr := runServiceCommand("start", "wuauserv"); restartErr != nil {
		if cleanErr != nil {
			return 0, fmt.Errorf("clean failed: %w; also failed to restart wuauserv: %v", cleanErr, restartErr)
		}
		return freed, fmt.Errorf("cleaned %s but failed to restart wuauserv: %w",
			core.FormatSize(freed), restartErr)
	}

	if cleanErr != nil {
		return 0, fmt.Errorf("failed to clean WU cache: %w", cleanErr)
	}

	return freed, nil
}

// runServiceCommand executes `net <action> <service>` with a timeout.
func runServiceCommand(action, service string) error {
	ctx, cancel := context.WithTimeout(context.Background(), serviceCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "net", action, service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("net %s %s: %w\n%s", action, service, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// ─── Windows.old ─────────────────────────────────────────────────────────────

// WindowsOldSize returns the size of C:\Windows.old if it exists.
// Returns 0 if not present or not elevated.
func WindowsOldSize() int64 {
	if !core.IsElevated() {
		return 0
	}

	dir := filepath.Join(systemDrive(), "Windows.old")
	if _, err := os.Stat(dir); err != nil {
		return 0
	}

	size, err := core.GetDirSize(dir)
	if err != nil {
		return 0
	}
	return size
}

// CleanWindowsOld removes C:\Windows.old after requiring a DangerConfirm
// from the user. This is irreversible. Requires admin privileges.
func CleanWindowsOld(dryRun bool) (int64, error) {
	if !core.IsElevated() {
		return 0, fmt.Errorf("removing Windows.old requires administrator privileges")
	}

	dir := filepath.Join(systemDrive(), "Windows.old")
	if _, err := os.Stat(dir); err != nil {
		return 0, nil // Not present.
	}

	size, _ := core.GetDirSize(dir)

	if dryRun {
		return size, nil
	}

	// Require explicit dangerous confirmation.
	confirmed, err := ui.DangerConfirm(fmt.Sprintf(
		"Delete Windows.old (%s)? This is IRREVERSIBLE and removes your ability to roll back.",
		core.FormatSize(size),
	))
	if err != nil || !confirmed {
		return 0, nil // User declined.
	}

	freed, delErr := core.SafeDelete(dir, false)
	if delErr != nil {
		return 0, fmt.Errorf("failed to delete Windows.old: %w", delErr)
	}

	return freed, nil
}

// ─── WER User Reports ────────────────────────────────────────────────────────

// ScanWERUserReports scans Windows Error Reporting directories that are
// accessible without admin (user-level WER paths).
func ScanWERUserReports(wl *whitelist.Whitelist) []CleanItem {
	local := os.Getenv("LOCALAPPDATA")

	werPaths := []string{
		filepath.Join(local, "Microsoft", "Windows", "WER", "ReportArchive"),
		filepath.Join(local, "Microsoft", "Windows", "WER", "ReportQueue"),
	}

	var items []CleanItem
	for _, p := range werPaths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if wl != nil && wl.IsWhitelisted(p) {
			continue
		}
		dirItems := scanDirectory(p, "system", "Windows Error Reports (user)", wl)
		items = append(items, dirItems...)
	}

	return items
}
