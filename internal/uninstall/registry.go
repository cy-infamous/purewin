package uninstall

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/sys/windows/registry"
)

// InstalledApp represents an application found in the Windows registry.
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

// ─── Registry Sources ────────────────────────────────────────────────────────

// registrySource describes one registry hive + path to scan.
type registrySource struct {
	root registry.Key
	path string
}

// uninstallSources are the three standard locations for installed programs.
var uninstallSources = []registrySource{
	{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
	{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
}

// kbPattern matches Windows update identifiers like KB1234567.
var kbPattern = regexp.MustCompile(`(?i)\bKB\d{6,}\b`)

// ─── Public API ──────────────────────────────────────────────────────────────

// GetInstalledApps reads installed applications from the Windows registry.
// If showAll is true, system components and Windows updates are included.
func GetInstalledApps(showAll bool) ([]InstalledApp, error) {
	seen := make(map[string]bool)
	var apps []InstalledApp

	for _, src := range uninstallSources {
		found, err := readAppsFromKey(src.root, src.path)
		if err != nil {
			// Registry path may not exist (e.g., WOW6432Node on 32-bit);
			// skip silently.
			continue
		}

		for _, app := range found {
			// Deduplicate by name + version.
			key := strings.ToLower(app.Name + "|" + app.Version)
			if seen[key] {
				continue
			}
			seen[key] = true

			// Filter unless showAll is set.
			if !showAll {
				if app.Name == "" {
					continue
				}
				if app.IsSystemComponent {
					continue
				}
				if kbPattern.MatchString(app.Name) {
					continue
				}
			}

			apps = append(apps, app)
		}
	}

	// Sort by size descending — largest first.
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].EstimatedSize > apps[j].EstimatedSize
	})

	return apps, nil
}

// ─── Registry Helpers ────────────────────────────────────────────────────────

// readAppsFromKey enumerates subkeys under the given registry path and
// reads application metadata from each.
func readAppsFromKey(root registry.Key, path string) ([]InstalledApp, error) {
	key, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	var apps []InstalledApp
	for _, name := range subkeys {
		app, readErr := readAppFromSubKey(root, path+`\`+name)
		if readErr != nil {
			continue
		}
		if app.Name == "" {
			continue
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// sanitizeRegistryString cleans a registry value for safe use.
// It strips control characters (except tab) and limits length to prevent
// display corruption or injection from malicious registry entries.
func sanitizeRegistryString(s string, maxLen int) string {
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// readAppFromSubKey reads a single application's metadata from a registry key.
// All string values are sanitized to prevent injection or display corruption.
func readAppFromSubKey(root registry.Key, path string) (InstalledApp, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return InstalledApp{}, err
	}
	defer key.Close()

	app := InstalledApp{
		Name:                 sanitizeRegistryString(readStringValue(key, "DisplayName"), 512),
		Version:              sanitizeRegistryString(readStringValue(key, "DisplayVersion"), 128),
		Publisher:            sanitizeRegistryString(readStringValue(key, "Publisher"), 256),
		InstallDate:          sanitizeRegistryString(readStringValue(key, "InstallDate"), 32),
		UninstallString:      sanitizeRegistryString(readStringValue(key, "UninstallString"), 2048),
		QuietUninstallString: sanitizeRegistryString(readStringValue(key, "QuietUninstallString"), 2048),
		InstallLocation:      sanitizeRegistryString(readStringValue(key, "InstallLocation"), 1024),
		BundleID:             sanitizeRegistryString(readStringValue(key, "BundleCachePath"), 1024),
	}

	// EstimatedSize is stored in KB as a DWORD.
	if size, _, sizeErr := key.GetIntegerValue("EstimatedSize"); sizeErr == nil {
		app.EstimatedSize = int64(size) * 1024 // Convert KB → bytes.
	}

	// SystemComponent is a DWORD (1 = system).
	if sc, _, scErr := key.GetIntegerValue("SystemComponent"); scErr == nil {
		app.IsSystemComponent = sc == 1
	}

	return app, nil
}

// readStringValue safely reads a string value from a registry key.
// Returns an empty string on any error.
func readStringValue(key registry.Key, name string) string {
	val, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return val
}
