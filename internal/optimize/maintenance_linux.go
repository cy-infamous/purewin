//go:build linux

package optimize

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunDISMCleanup cleans the package manager cache.
// Auto-detects apt, dnf, pacman, or zypper.
func RunDISMCleanup() error {
	if _, err := exec.LookPath("apt-get"); err == nil {
		return runAptClean()
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return runDnfClean()
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return runPacmanClean()
	}
	if _, err := exec.LookPath("zypper"); err == nil {
		return runZypperClean()
	}
	return fmt.Errorf("no supported package manager found (tried apt, dnf, pacman, zypper)")
}

func runAptClean() error {
	// Clean downloaded package files.
	if output, err := exec.Command("apt-get", "clean").CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get clean: %w\n%s", err, output)
	}
	// Remove orphaned dependencies.
	if output, err := exec.Command("apt-get", "-y", "autoremove").CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get autoremove: %w\n%s", err, output)
	}
	return nil
}

func runDnfClean() error {
	if output, err := exec.Command("dnf", "clean", "all").CombinedOutput(); err != nil {
		return fmt.Errorf("dnf clean all: %w\n%s", err, output)
	}
	if output, err := exec.Command("dnf", "-y", "autoremove").CombinedOutput(); err != nil {
		return fmt.Errorf("dnf autoremove: %w\n%s", err, output)
	}
	return nil
}

func runPacmanClean() error {
	// -Sc: remove uninstalled package cache, keep installed versions.
	if output, err := exec.Command("pacman", "-Sc", "--noconfirm").CombinedOutput(); err != nil {
		return fmt.Errorf("pacman -Sc: %w\n%s", err, output)
	}
	return nil
}

func runZypperClean() error {
	if output, err := exec.Command("zypper", "clean", "-a").CombinedOutput(); err != nil {
		return fmt.Errorf("zypper clean: %w\n%s", err, output)
	}
	return nil
}

// RunSFCCheck runs SSD TRIM to optimize flash storage performance.
func RunSFCCheck() error {
	cmd := exec.Command("fstrim", "-av")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// fstrim may fail on VMs or non-SSD drives — not fatal.
		return fmt.Errorf("fstrim: %w\n%s", err, output)
	}
	return nil
}

// RebuildIconCache rebuilds the desktop icon and MIME type caches.
func RebuildIconCache() error {
	var errs []string

	// Update GTK icon cache for common theme directories.
	for _, dir := range []string{
		"/usr/share/icons",
		fmt.Sprintf("%s/.local/share/icons", homeDir()),
	} {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			themeDir := fmt.Sprintf("%s/%s", dir, e.Name())
			cmd := exec.Command("gtk-update-icon-cache", "-f", "-t", themeDir)
			_ = cmd.Run() // best-effort
		}
	}

	// Update MIME database.
	if _, err := exec.LookPath("update-mime-database"); err == nil {
		cmd := exec.Command("update-mime-database", "/usr/share/mime")
		_ = cmd.Run()
	}

	if len(errs) > 0 {
		return fmt.Errorf("partial failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

// RebuildSearchIndex vacuums old systemd journal entries (older than 7 days).
func RebuildSearchIndex() error {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return fmt.Errorf("journalctl not found")
	}
	cmd := exec.Command("journalctl", "--vacuum-time=7d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("journalctl vacuum: %w\n%s", err, output)
	}
	return nil
}

// ClearEventLogs removes orphaned/old kernel packages.
func ClearEventLogs() error {
	if _, err := exec.LookPath("apt-get"); err == nil {
		cmd := exec.Command("apt-get", "-y", "autoremove", "--purge")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("autoremove kernels: %w\n%s", err, output)
		}
		return nil
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		cmd := exec.Command("dnf", "-y", "autoremove")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("dnf autoremove: %w\n%s", err, output)
		}
		return nil
	}
	// For pacman and others, kernel cleanup is manual — skip gracefully.
	return fmt.Errorf("automatic kernel cleanup not supported for this package manager")
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return "/root"
}
