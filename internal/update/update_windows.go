//go:build windows

package update

import (
	"archive/zip"
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
		fmt.Sprintf("pw-windows-%s.zip", runtime.GOARCH),
		"pw-windows-amd64.zip",
		"pw.exe",
		fmt.Sprintf("pw_%s_%s.exe", runtime.GOOS, runtime.GOARCH),
		fmt.Sprintf("purewin_%s_%s.exe", runtime.GOOS, runtime.GOARCH),
	}
}

func DownloadUpdate(url string) (string, error) {
	tempFile, err := os.CreateTemp("", "purewin_update_*.zip")
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

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write update: %w", err)
	}

	if strings.HasSuffix(strings.ToLower(url), ".zip") {
		extractedPath, err := extractBinaryFromZip(tempPath)
		if err != nil {
			return "", err
		}
		_ = os.Remove(tempPath)
		return extractedPath, nil
	}

	return tempPath, nil
}

func extractBinaryFromZip(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	targetName := "pw.exe"
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == targetName || base == "pw.exe" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to read zip entry: %w", err)
			}
			defer rc.Close()

			outPath := filepath.Join(os.TempDir(), "purewin_update_extracted.exe")
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return "", fmt.Errorf("failed to create extracted file: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return "", fmt.Errorf("failed to extract: %w", err)
			}
			return outPath, nil
		}
	}

	return "", fmt.Errorf("pw.exe not found in zip archive")
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
		return fmt.Errorf("failed to rename current executable: %w", err)
	}

	if err := copyFile(tempPath, currentExePath); err != nil {
		_ = os.Rename(oldPath, currentExePath)
		return fmt.Errorf("failed to copy new executable: %w", err)
	}

	_ = scheduleFileDeletion(oldPath)

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
	escaped := strings.ReplaceAll(filePath, "'", "''")
	psCommand := fmt.Sprintf("Start-Sleep -Seconds 2; Remove-Item -LiteralPath '%s' -Force", escaped)
	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", psCommand)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to schedule file deletion: %w", err)
	}

	return nil
}

func RemoveFromPath(exePath string) error {
	exeDir := filepath.Dir(exePath)

	escaped := strings.ReplaceAll(exeDir, "'", "''")
	psScript := fmt.Sprintf(`
		$exeDir = '%s'
		$path = [Environment]::GetEnvironmentVariable('Path', 'User')
		if ($null -eq $path) { exit 0 }
		$pathParts = $path -split ';'
		$newPath = $pathParts | Where-Object {
			$_ -ne '' -and $_.TrimEnd('\') -ine $exeDir.TrimEnd('\')
		}
		$newPathString = $newPath -join ';'
		if ($newPathString -eq $path) { exit 0 }
		[Environment]::SetEnvironmentVariable('Path', $newPathString, 'User')
	`, escaped)

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove from PATH: %w (output: %s)", err, string(output))
	}

	return nil
}
