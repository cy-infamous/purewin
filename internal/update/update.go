package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// GitHubAPIURL is the GitHub API endpoint for releases
	GitHubAPIURL = "https://api.github.com/repos/cy-infamous/purewin/releases/latest"

	// UpdateCheckCacheFile stores the last update check result
	UpdateCheckCacheFile = "last_update_check.json"

	// UpdateCheckInterval is how often to check for updates (24 hours)
	UpdateCheckInterval = 24 * time.Hour
)

// ReleaseInfo holds information about a GitHub release.
type ReleaseInfo struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	URL         string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Asset represents a release asset (downloadable file).
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateCheckCache stores the last update check result.
type UpdateCheckCache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	DownloadURL   string    `json:"download_url"`
}

// CheckForUpdate checks GitHub for the latest release.
// Returns the latest version, download URL, and any error.
func CheckForUpdate(currentVersion string) (latestVersion string, downloadURL string, err error) {
	// Normalize version strings (remove 'v' prefix if present)
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// Make HTTP request to GitHub API
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(GitHubAPIURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse response
	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion = strings.TrimPrefix(release.TagName, "v")

	// Find the appropriate asset for this platform
	assetName := getAssetNameForPlatform()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no asset found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return latestVersion, downloadURL, nil
}

// CheckForUpdateBackground performs a non-blocking update check and caches the result.
// This is meant to be called at startup to check for updates without blocking the user.
func CheckForUpdateBackground(currentVersion string, cacheDir string) {
	go func() {
		// Check if we need to perform a check
		cachePath := filepath.Join(cacheDir, UpdateCheckCacheFile)
		cache, err := loadUpdateCache(cachePath)
		if err == nil && time.Since(cache.LastCheck) < UpdateCheckInterval {
			// Recent check, skip
			return
		}

		// Perform the check
		latestVersion, downloadURL, err := CheckForUpdate(currentVersion)
		if err != nil {
			return
		}

		// Save to cache
		newCache := UpdateCheckCache{
			LastCheck:     time.Now(),
			LatestVersion: latestVersion,
			DownloadURL:   downloadURL,
		}
		_ = saveUpdateCache(cachePath, newCache)
	}()
}

// loadUpdateCache reads the cached update check result.
func loadUpdateCache(path string) (*UpdateCheckCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cache UpdateCheckCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// saveUpdateCache writes the update check result to cache.
func saveUpdateCache(path string, cache UpdateCheckCache) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// getAssetNameForPlatform returns the expected asset name for the current platform.
func getAssetNameForPlatform() string {
	// Expected format: purewin_windows_amd64.exe, purewin_windows_arm64.exe, etc.
	return fmt.Sprintf("purewin_%s_%s.exe", runtime.GOOS, runtime.GOARCH)
}

// DownloadUpdate downloads the update from the given URL to a temporary file.
// Returns the path to the downloaded file.
func DownloadUpdate(url string) (string, error) {
	// Create temp file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "purewin_update.exe")

	// Download
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Write to file
	out, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write update: %w", err)
	}

	return tempFile, nil
}

// ApplyUpdate replaces the current binary with the downloaded update.
// On Windows, this uses the rename trick to handle the "can't delete running exe" issue.
func ApplyUpdate(tempPath string) error {
	// Get current executable path
	currentExePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Resolve symlinks
	currentExePath, err = filepath.EvalSymlinks(currentExePath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Rename current exe to .old
	oldPath := currentExePath + ".old"

	// Remove any existing .old file
	_ = os.Remove(oldPath)

	// Rename current to .old
	if err := os.Rename(currentExePath, oldPath); err != nil {
		return fmt.Errorf("failed to rename current executable: %w", err)
	}

	// Copy new binary to the original location
	if err := copyFile(tempPath, currentExePath); err != nil {
		// Try to restore the old binary
		_ = os.Rename(oldPath, currentExePath)
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	// Schedule deletion of .old file on next run
	// We'll handle this in the cleanup logic

	return nil
}

// copyFile copies a file from src to dst. The destination file is explicitly
// closed (not deferred) so that write errors during Close are caught.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}

// CleanupOldBinary removes the .old file left from a previous update.
func CleanupOldBinary() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return
	}

	oldPath := exePath + ".old"
	_ = os.Remove(oldPath)
}

// SelfRemove removes the binary, config, and cache directories.
// Returns an error if removal fails.
func SelfRemove(configDir, cacheDir string) error {
	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Remove config directory
	if configDir != "" {
		if err := os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("failed to remove config directory: %w", err)
		}
	}

	// Remove cache directory (if different from config)
	if cacheDir != "" && cacheDir != configDir {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to remove cache directory: %w", err)
		}
	}

	// Schedule binary deletion using cmd.exe
	// We can't delete ourselves while running, so we spawn a process that waits
	// and then deletes the binary
	return scheduleBinaryDeletion(exePath)
}

// scheduleBinaryDeletion spawns a detached cmd.exe process that waits a few
// seconds and then deletes the binary. The entire shell command is passed as
// a single string to "cmd /C" so that redirection and chaining operators are
// interpreted correctly.
func scheduleBinaryDeletion(exePath string) error {
	// Validate the path doesn't contain double-quotes which would break
	// the cmd /C quoting and allow command injection.
	if strings.ContainsRune(exePath, '"') {
		return fmt.Errorf("executable path contains invalid character: %s", exePath)
	}

	script := fmt.Sprintf(`ping localhost -n 3 > nul & del /f /q "%s"`, exePath)
	cmd := exec.Command("cmd", "/C", script)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to schedule binary deletion: %w", err)
	}

	return nil
}

// IsNewerVersion compares two semver version strings and returns true if
// newer > current. Versions may optionally have a "v" prefix.
// Handles unequal segment counts (e.g. "1.2" vs "1.2.1").
func IsNewerVersion(current, newer string) bool {
	current = strings.TrimPrefix(current, "v")
	newer = strings.TrimPrefix(newer, "v")

	if current == newer {
		return false
	}

	curParts := strings.Split(current, ".")
	newParts := strings.Split(newer, ".")

	// Compare up to the length of the longer version string.
	maxLen := len(curParts)
	if len(newParts) > maxLen {
		maxLen = len(newParts)
	}

	for i := 0; i < maxLen; i++ {
		var c, n int
		if i < len(curParts) {
			c = atoiSafe(curParts[i])
		}
		if i < len(newParts) {
			n = atoiSafe(newParts[i])
		}
		if n > c {
			return true
		}
		if n < c {
			return false
		}
	}

	return false
}

// DownloadAndVerifyUpdate downloads the update binary and verifies its
// integrity. It checks the file size against the GitHub API metadata and,
// if a checksums file exists in the release assets, verifies the SHA256 hash.
// This prevents corrupted or tampered binaries from being applied.
func DownloadAndVerifyUpdate(release *ReleaseInfo) (string, error) {
	assetName := getAssetNameForPlatform()

	// Find the download URL and expected size from the release assets.
	var downloadURL string
	var expectedSize int64
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			expectedSize = asset.Size
			break
		}
	}
	if downloadURL == "" {
		return "", fmt.Errorf("asset %s not found in release %s", assetName, release.TagName)
	}

	// Download the binary.
	path, err := DownloadUpdate(downloadURL)
	if err != nil {
		return "", err
	}

	// Verify file size matches GitHub API metadata.
	info, err := os.Stat(path)
	if err != nil {
		os.Remove(path)
		return "", fmt.Errorf("cannot stat downloaded file: %w", err)
	}
	if expectedSize > 0 && info.Size() != expectedSize {
		os.Remove(path)
		return "", fmt.Errorf("download size mismatch: expected %d bytes, got %d", expectedSize, info.Size())
	}

	// Look for a SHA256 checksums file in the release assets.
	expectedHash, hashErr := fetchExpectedHash(release, assetName)
	if hashErr == nil && expectedHash != "" {
		actualHash, err := hashFileSHA256(path)
		if err != nil {
			os.Remove(path)
			return "", fmt.Errorf("cannot hash downloaded file: %w", err)
		}
		if !strings.EqualFold(actualHash, expectedHash) {
			os.Remove(path)
			return "", fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	}

	return path, nil
}

// fetchExpectedHash looks for a checksums file in the release assets and
// extracts the expected SHA256 hash for the named asset.
func fetchExpectedHash(release *ReleaseInfo, assetName string) (string, error) {
	var checksumURL string
	for _, asset := range release.Assets {
		lower := strings.ToLower(asset.Name)
		if strings.Contains(lower, "checksum") ||
			strings.HasSuffix(lower, ".sha256") ||
			lower == "sha256sums" || lower == "sha256sums.txt" {
			checksumURL = asset.BrowserDownloadURL
			break
		}
	}
	if checksumURL == "" {
		return "", fmt.Errorf("no checksum file in release")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}

	// Parse "hash  filename" or "hash filename" format.
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && strings.EqualFold(fields[len(fields)-1], assetName) {
			return strings.ToLower(fields[0]), nil
		}
	}

	return "", fmt.Errorf("hash for %s not found in checksums file", assetName)
}

// hashFileSHA256 returns the hex-encoded SHA256 hash of the file at path.
func hashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CheckForUpdateFull is like CheckForUpdate but returns the full ReleaseInfo
// for use with DownloadAndVerifyUpdate.
func CheckForUpdateFull(currentVersion string) (*ReleaseInfo, error) {
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(GitHubAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// atoiSafe converts a string to int, returning 0 for non-numeric values.
func atoiSafe(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
