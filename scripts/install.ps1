#!/usr/bin/env pwsh
# PureWin Installer for Windows
# Usage: irm https://raw.githubusercontent.com/cy-infamous/purewin/main/scripts/install.ps1 | iex
#
# Safety: The entire script is wrapped in a function to prevent partial
# execution if the download is interrupted during pipe-to-shell usage.
# If the script is truncated mid-download, the function definition is
# incomplete and PowerShell raises a syntax error instead of running
# partial commands.

# Enforce TLS 1.2+ for all HTTPS connections.
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

function Install-PureWin {
    $ErrorActionPreference = "Stop"

    Write-Host "Installing PureWin..." -ForegroundColor Cyan

    # Detect architecture
    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
        "arm64"
    } elseif ([Environment]::Is64BitOperatingSystem) {
        "amd64"
    } else {
        "386"
    }
    Write-Host "   Detected architecture: $arch" -ForegroundColor Gray

    # Define install directory
    $installDir = Join-Path $env:LOCALAPPDATA "purewin"
    $binPath = Join-Path $installDir "pw.exe"

    # Create install directory if it doesn't exist
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        Write-Host "   Created directory: $installDir" -ForegroundColor Gray
    }

    # Fetch latest release info from GitHub API
    Write-Host "   Fetching latest release..." -ForegroundColor Gray
    try {
        $releaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/cy-infamous/purewin/releases/latest"
        $version = $releaseInfo.tag_name
        # Strip leading 'v' for asset name (goreleaser uses version without v prefix)
        $versionNum = $version -replace '^v', ''
        Write-Host "   Latest version: $version" -ForegroundColor Green
    } catch {
        Write-Host "   Failed to fetch release info from GitHub API." -ForegroundColor Red
        Write-Host "   Error: $_" -ForegroundColor Red
        exit 1
    }

    # Find the correct asset for the detected architecture
    $assetName = "purewin_${versionNum}_windows_${arch}.zip"
    $asset = $releaseInfo.assets | Where-Object { $_.name -eq $assetName }

    if (-not $asset) {
        Write-Host "   Could not find release asset: $assetName" -ForegroundColor Red
        Write-Host "   Available assets:" -ForegroundColor Yellow
        $releaseInfo.assets | ForEach-Object { Write-Host "     - $($_.name)" -ForegroundColor Yellow }
        exit 1
    }

    # Download the release archive
    $downloadUrl = $asset.browser_download_url
    $expectedSize = $asset.size
    $zipPath = Join-Path $env:TEMP "purewin_latest.zip"

    Write-Host "   Downloading $assetName..." -ForegroundColor Gray
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
        Write-Host "   Downloaded to: $zipPath" -ForegroundColor Gray

        # Verify download size against GitHub API metadata to detect corruption.
        if ($expectedSize) {
            $actualSize = (Get-Item $zipPath).Length
            if ($actualSize -ne $expectedSize) {
                Write-Host "   Download size mismatch: expected $expectedSize bytes, got $actualSize bytes." -ForegroundColor Red
                Remove-Item $zipPath -Force -ErrorAction SilentlyContinue
                exit 1
            }
            Write-Host "   Size verified: $actualSize bytes" -ForegroundColor Gray
        }
    } catch {
        Write-Host "   Failed to download release." -ForegroundColor Red
        Write-Host "   Error: $_" -ForegroundColor Red
        exit 1
    }

    # Extract pw.exe from the archive
    Write-Host "   Extracting pw.exe..." -ForegroundColor Gray
    try {
        Expand-Archive -Path $zipPath -DestinationPath $installDir -Force

        # Find pw.exe in the extracted files (it might be in a subdirectory)
        $extractedExe = Get-ChildItem -Path $installDir -Filter "pw.exe" -Recurse -File | Select-Object -First 1

        if ($extractedExe) {
            # Move to root of install directory if needed
            if ($extractedExe.FullName -ne $binPath) {
                Move-Item -Path $extractedExe.FullName -Destination $binPath -Force
            }
            Write-Host "   Installed to: $binPath" -ForegroundColor Gray
        } else {
            Write-Host "   Could not find pw.exe in the downloaded archive." -ForegroundColor Red
            exit 1
        }
    } catch {
        Write-Host "   Failed to extract archive." -ForegroundColor Red
        Write-Host "   Error: $_" -ForegroundColor Red
        exit 1
    } finally {
        # Clean up downloaded zip
        if (Test-Path $zipPath) {
            Remove-Item $zipPath -Force
        }
    }

    # Add to PATH if not already present
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$installDir*") {
        Write-Host "   Adding to PATH..." -ForegroundColor Gray
        $userPath = $userPath.TrimEnd(';')
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
        $env:Path = "$env:Path;$installDir"
        Write-Host "   Added $installDir to PATH" -ForegroundColor Green
    } else {
        Write-Host "   Already in PATH" -ForegroundColor Gray
    }

    # Verify installation
    Write-Host ""
    Write-Host "PureWin installed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "   Run 'pw' to get started" -ForegroundColor Cyan
    Write-Host "   Run 'pw --help' for available commands" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "   NOTE: You may need to restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
    Write-Host ""

    # Try to run version check
    try {
        & $binPath version
    } catch {
        Write-Host "   Installation complete, but PATH update requires terminal restart." -ForegroundColor Yellow
    }
}

# Execute the installer.
Install-PureWin
