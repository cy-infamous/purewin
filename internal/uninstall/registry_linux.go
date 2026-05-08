//go:build linux

package uninstall

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type InstalledApp struct {
	Name                 string
	Version              string
	Publisher            string
	InstallDate          string
	EstimatedSize        int64
	UninstallString      string
	QuietUninstallString string
	InstallLocation      string
	BundleID             string
	IsSystemComponent    bool
}

func detectPkgManager() string {
	if _, err := exec.LookPath("dnf"); err == nil {
		return "dnf"
	}
	if _, err := exec.LookPath("yum"); err == nil {
		return "yum"
	}
	if _, err := exec.LookPath("apt"); err == nil {
		return "apt"
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return "pacman"
	}
	if _, err := exec.LookPath("zypper"); err == nil {
		return "zypper"
	}
	return ""
}

func GetInstalledApps(showAll bool) ([]InstalledApp, error) {
	pkgMgr := detectPkgManager()
	switch pkgMgr {
	case "dnf", "yum":
		return getRpmApps(showAll)
	case "apt":
		return getDpkgApps(showAll)
	case "pacman":
		return getPacmanApps(showAll)
	case "zypper":
		return getRpmApps(showAll)
	default:
		return nil, fmt.Errorf("no supported package manager found (tried dnf, yum, apt, pacman, zypper)")
	}
}

func getRpmApps(showAll bool) ([]InstalledApp, error) {
	cmd := exec.Command("rpm", "-qa", "--qf",
		"%{NAME}\t%{VERSION}-%{RELEASE}\t%{VENDOR}\t%{INSTALLTIME}\t%{SIZE}\t%{ARCH}\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rpm query failed: %w", err)
	}

	var apps []InstalledApp
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 6 {
			continue
		}

		name := fields[0]
		version := fields[1]
		publisher := fields[2]
		arch := fields[5]

		var size int64
		if s, err := strconv.ParseInt(fields[4], 10, 64); err == nil {
			size = s
		}

		var installDate string
		if ts, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			installDate = time.Unix(ts, 0).Format("2006-01-02")
		}

		if isSystemPackage(name) && !showAll {
			continue
		}

		apps = append(apps, InstalledApp{
			Name:              name,
			Version:           version,
			Publisher:         publisher,
			InstallDate:       installDate,
			EstimatedSize:     size,
			InstallLocation:   arch,
			UninstallString:   "dnf remove " + name,
			QuietUninstallString: "dnf remove -y " + name,
		})
	}

	return apps, nil
}

func getDpkgApps(showAll bool) ([]InstalledApp, error) {
	cmd := exec.Command("dpkg-query", "-W", "-f",
		"${Package}\t${Version}\t${Maintainer}\t${Installed-Size}\t${Architecture}\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("dpkg-query failed: %w", err)
	}

	var apps []InstalledApp
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 5 {
			continue
		}

		name := fields[0]
		version := fields[1]
		publisher := fields[2]
		arch := fields[4]

		var size int64
		if s, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			size = s * 1024
		}

		if isSystemPackage(name) && !showAll {
			continue
		}

		apps = append(apps, InstalledApp{
			Name:              name,
			Version:           version,
			Publisher:         publisher,
			EstimatedSize:     size,
			InstallLocation:   arch,
			UninstallString:   "apt remove " + name,
			QuietUninstallString: "apt remove -y " + name,
		})
	}

	return apps, nil
}

func getPacmanApps(showAll bool) ([]InstalledApp, error) {
	cmd := exec.Command("pacman", "-Q")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pacman query failed: %w", err)
	}

	var apps []InstalledApp
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), " ", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := parts[1]

		if isSystemPackage(name) && !showAll {
			continue
		}

		apps = append(apps, InstalledApp{
			Name:              name,
			Version:           version,
			UninstallString:   "pacman -R " + name,
			QuietUninstallString: "pacman -R --noconfirm " + name,
		})
	}

	return apps, nil
}

func isSystemPackage(name string) bool {
	system := []string{
		"kernel", "grub", "systemd", "glibc", "bash", "coreutils",
		"filesystem", "setup", "basesystem", "fedora-release", "filesystem",
		"tzdata", "ca-certificates", "p11-kit", "crypto-policies",
	}
	lower := strings.ToLower(name)
	for _, s := range system {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func UninstallApp(app InstalledApp, quiet bool) error {
	var cmdStr string
	if quiet && app.QuietUninstallString != "" {
		cmdStr = app.QuietUninstallString
	} else {
		cmdStr = app.UninstallString
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("no uninstall command for %s", app.Name)
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	if app.InstallLocation != "" {
		if info, err := os.Stat(app.InstallLocation); err == nil && info.IsDir() {
			cmd.Dir = app.InstallLocation
		}
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
