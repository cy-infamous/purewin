//go:build linux

package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func getAssetNamesForPlatform() []string {
	return []string{
		fmt.Sprintf("pw-linux-%s.tar.gz", runtime.GOARCH),
		"pw-linux-amd64.tar.gz",
	}
}

func DownloadUpdate(url string) (string, error) {
	suffix := ".bin"
	if strings.HasSuffix(strings.ToLower(url), ".tar.gz") {
		suffix = ".tar.gz"
	}

	tempFile, err := os.CreateTemp("", "purewin_update_*"+suffix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	_ = os.Remove(tempPath)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write update: %w", err)
	}

	if strings.HasSuffix(strings.ToLower(url), ".tar.gz") {
		extractedPath, err := extractBinaryFromTarGz(tempPath)
		if err != nil {
			return "", err
		}
		_ = os.Remove(tempPath)
		return extractedPath, nil
	}

	return tempPath, nil
}

func extractBinaryFromTarGz(tarPath string) (string, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to decompress: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		base := filepath.Base(hdr.Name)
		if base == "pw" || base == "pw-amd64" || base == "pw-arm64" {
			outPath := filepath.Join(os.TempDir(), "purewin_update_extracted")
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return "", fmt.Errorf("failed to create extracted file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, tr); err != nil {
				return "", fmt.Errorf("failed to extract: %w", err)
			}
			return outPath, nil
		}
	}

	return "", fmt.Errorf("pw binary not found in tar.gz archive")
}

func ApplyUpdate(tempPath string) error {
	currentExePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	currentExePath, err = filepath.EvalSymlinks(currentExePath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	oldPath := currentExePath + ".old"
	_ = os.Remove(oldPath)

	if err := os.Rename(currentExePath, oldPath); err != nil {
		if os.IsPermission(err) {
			return applyUpdateWithSudo(tempPath, currentExePath, oldPath)
		}
		return fmt.Errorf("failed to rename current executable: %w", err)
	}

	if err := copyFile(tempPath, currentExePath); err != nil {
		_ = os.Rename(oldPath, currentExePath)
		if os.IsPermission(err) {
			return applyUpdateWithSudo(tempPath, currentExePath, oldPath)
		}
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	return nil
}

func applyUpdateWithSudo(tempPath, currentExePath, oldPath string) error {
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("permission denied and sudo not available")
	}

	if err := exec.Command(sudo, "mv", currentExePath, oldPath).Run(); err != nil {
		return fmt.Errorf("sudo mv failed: %w", err)
	}

	if err := exec.Command(sudo, "cp", tempPath, currentExePath).Run(); err != nil {
		_ = exec.Command(sudo, "mv", oldPath, currentExePath).Run()
		return fmt.Errorf("sudo cp failed: %w", err)
	}

	_ = exec.Command(sudo, "chmod", "+x", currentExePath).Run()

	return nil
}

func SelfRemove(configDir, cacheDir string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	if configDir != "" {
		if err := os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("failed to remove config directory: %w", err)
		}
	}

	if cacheDir != "" && cacheDir != configDir {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to remove cache directory: %w", err)
		}
	}

	return scheduleFileDeletion(exePath)
}

func scheduleFileDeletion(filePath string) error {
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("(sleep 2; rm -f '%s') &", strings.ReplaceAll(filePath, "'", "'\\''")))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to schedule file deletion: %w", err)
	}
	return nil
}

func RemoveFromPath(exePath string) error {
	return nil
}
