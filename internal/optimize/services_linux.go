//go:build linux

package optimize

import (
	"fmt"
	"os/exec"
)

// ManagedService describes a systemd service that PureWin can manage.
type ManagedService struct {
	Name        string
	DisplayName string
}

// GetManagedServices returns the list of systemd services that PureWin can restart.
func GetManagedServices() []ManagedService {
	return []ManagedService{
		{Name: "NetworkManager", DisplayName: "Network Manager"},
		{Name: "systemd-resolved", DisplayName: "DNS Resolver"},
	}
}

// FlushDNS clears the DNS resolver cache.
func FlushDNS() error {
	// Try resolvectl first (modern systemd-resolved).
	if _, err := exec.LookPath("resolvectl"); err == nil {
		cmd := exec.Command("resolvectl", "flush-caches")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("resolvectl flush-caches failed: %w\n%s", err, output)
		}
		return nil
	}

	// Fallback to systemd-resolve (older versions).
	if _, err := exec.LookPath("systemd-resolve"); err == nil {
		cmd := exec.Command("systemd-resolve", "--flush-caches")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("systemd-resolve --flush-caches failed: %w\n%s", err, output)
		}
		return nil
	}

	return fmt.Errorf("no supported DNS cache manager found (tried resolvectl, systemd-resolve)")
}

// RestartService restarts a systemd service.
func RestartService(name string) error {
	cmd := exec.Command("systemctl", "restart", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl restart %s: %w\n%s", name, err, output)
	}
	return nil
}
