package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	GitHubAPIURL         = "https://api.github.com/repos/cy-infamous/purewin/releases/latest"
	UpdateCheckCacheFile = "last_update_check.json"
	UpdateCheckInterval  = 24 * time.Hour
)

type ReleaseInfo struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	URL         string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type UpdateCheckCache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
	DownloadURL   string    `json:"download_url"`
}

func CheckForUpdate(currentVersion string) (latestVersion string, downloadURL string, err error) {
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(GitHubAPIURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to parse release info: %w", err)
	}

	latestVersion = strings.TrimPrefix(release.TagName, "v")

	assetNames := getAssetNamesForPlatform()
	for _, asset := range release.Assets {
		for _, name := range assetNames {
			if strings.EqualFold(asset.Name, name) {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
		if downloadURL != "" {
			break
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no asset found for platform %s/%s (tried: %s)",
			runtime.GOOS, runtime.GOARCH, strings.Join(assetNames, ", "))
	}

	return latestVersion, downloadURL, nil
}

func CheckForUpdateBackground(currentVersion string, cacheDir string) {
	go func() {
		cachePath := filepath.Join(cacheDir, UpdateCheckCacheFile)
		cache, err := loadUpdateCache(cachePath)
		if err == nil && time.Since(cache.LastCheck) < UpdateCheckInterval {
			return
		}

		latestVersion, downloadURL, err := CheckForUpdate(currentVersion)
		if err != nil {
			return
		}

		newCache := UpdateCheckCache{
			LastCheck:     time.Now(),
			LatestVersion: latestVersion,
			DownloadURL:   downloadURL,
		}
		_ = saveUpdateCache(cachePath, newCache)
	}()
}

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

func saveUpdateCache(path string, cache UpdateCheckCache) error {
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}

func IsNewerVersion(current, newer string) bool {
	current = strings.TrimPrefix(current, "v")
	newer = strings.TrimPrefix(newer, "v")

	currentParts := strings.Split(current, ".")
	newerParts := strings.Split(newer, ".")

	maxLen := len(currentParts)
	if len(newerParts) > maxLen {
		maxLen = len(newerParts)
	}

	for i := 0; i < maxLen; i++ {
		currentVal := 0
		if i < len(currentParts) {
			numericPart := currentParts[i]
			if idx := strings.IndexFunc(numericPart, func(r rune) bool { return r < '0' || r > '9' }); idx > 0 {
				numericPart = numericPart[:idx]
			}
			if numericPart != "" {
				fmt.Sscanf(numericPart, "%d", &currentVal)
			}
		}

		newerVal := 0
		if i < len(newerParts) {
			numericPart := newerParts[i]
			if idx := strings.IndexFunc(numericPart, func(r rune) bool { return r < '0' || r > '9' }); idx > 0 {
				numericPart = numericPart[:idx]
			}
			if numericPart != "" {
				fmt.Sscanf(numericPart, "%d", &newerVal)
			}
		}

		if newerVal > currentVal {
			return true
		} else if newerVal < currentVal {
			return false
		}
	}

	return false
}
