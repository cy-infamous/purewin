# WinMole

```
     /\_/\     
    / o o \    
   (  =^=  )   
    )     (    
   (       )   
  ( /|   |\ )  
   \| |_| |/   
    \_____/    
```

**Deep clean and optimize your Windows.**

[![CI](https://github.com/lakshaymaurya-felt/winmole/actions/workflows/ci.yml/badge.svg)](https://github.com/lakshaymaurya-felt/winmole/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lakshaymaurya-felt/winmole)](https://github.com/lakshaymaurya-felt/winmole)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

---

## What is WinMole?

**WinMole** is a native Windows port of [Mole](https://github.com/tw93/Mole), a powerful system cleanup and optimization toolkit. It provides an all-in-one CLI for deep cleaning, disk analysis, app uninstallation, system monitoring, and performance optimization—all from the comfort of your terminal.

Built with Go, WinMole is fast, lightweight, and designed specifically for Windows systems. Whether you're reclaiming disk space, removing stubborn apps, or monitoring system health in real-time, WinMole has you covered.

---

## Features

- **🧹 Deep System Cleanup** — Remove temp files, caches, logs, browser data, and dev tool artifacts
- **🗑️ Complete App Removal** — Uninstall apps and wipe their registry entries, configs, and hidden remnants
- **📊 Disk Space Analysis** — Interactive treemap visualization of disk usage
- **⚡ System Optimization** — Refresh caches, restart services, optimize performance
- **📈 Real-Time Monitoring** — Live dashboard for CPU, memory, disk, network, GPU, and battery
- **🔍 Installer Cleanup** — Find and remove orphaned installer files from Downloads and caches
- **🛠️ Dev Tool Cleanup** — Purge build artifacts from node_modules, target/, .gradle, .nuget, and more
- **🔒 Safety First** — Whitelist protection, dry-run mode, and NEVER_DELETE safeguards
- **🎨 Beautiful TUI** — Rich interactive menus powered by Bubble Tea
- **🔄 Self-Updating** — Check for and install updates directly from GitHub releases
- **⚙️ PowerShell Completion** — Tab completion for all commands

---

## Quick Install

### Via Go Install
```bash
go install github.com/lakshaymaurya-felt/winmole@latest
```

### Via PowerShell (one-liner)
```powershell
irm https://raw.githubusercontent.com/lakshaymaurya-felt/winmole/main/scripts/install.ps1 | iex
```

### Via GitHub Releases
Download the latest `.zip` from [Releases](https://github.com/lakshaymaurya-felt/winmole/releases), extract `wm.exe`, and add to your PATH.

---

## Usage

### Interactive Menu
Run `wm` without arguments to launch the full-screen interactive menu:

```bash
wm
```

### Direct Commands
Run specific operations directly:

```bash
# Clean system caches (preview mode)
wm clean --dry-run

# Clean all categories (requires admin for system caches)
wm clean --all

# Clean only browser caches
wm clean --browser

# Uninstall an app
wm uninstall

# Analyze disk usage
wm analyze C:\

# Monitor system health in real-time
wm status

# Remove installer files
wm installer

# Optimize system performance
wm optimize

# Clean dev tool build artifacts
wm purge

# Update WinMole to latest version
wm update

# Show version info
wm version
```

---

## Commands Reference

| Command      | Description                                                  | Admin Required |
|--------------|--------------------------------------------------------------|----------------|
| `clean`      | Deep cleanup of caches, logs, temp files, browser leftovers | Partial*       |
| `uninstall`  | Remove apps completely with registry and leftover cleanup   | Yes            |
| `analyze`    | Interactive disk space analyzer with visual tree view       | No             |
| `optimize`   | Refresh caches, restart services, optimize performance      | Yes            |
| `status`     | Real-time dashboard for CPU, memory, disk, network, GPU     | No             |
| `installer`  | Find and remove installer files (.exe, .msi, .msix)         | No             |
| `purge`      | Clean project build artifacts (node_modules, target/, etc.) | No             |
| `update`     | Check for and install latest WinMole version                | No             |
| `remove`     | Uninstall WinMole and remove config/cache                   | No             |
| `completion` | Generate PowerShell tab completion                          | No             |
| `version`    | Show installed version                                      | No             |

*`clean --system` requires admin; `--user`, `--browser`, `--dev` do not.

---

## Safety

WinMole is designed with safety as a top priority:

### NEVER_DELETE Protection
Critical system paths are hardcoded as off-limits:
- `C:\Windows`
- `C:\Program Files`
- `C:\Program Files (x86)`
- User profile root directories

### Whitelist System
Protect specific caches you want to keep:
```bash
wm clean --whitelist
```
Whitelisted items are persisted in your config and skipped during cleanup.

### Dry-Run Mode
Preview exactly what will be deleted before committing:
```bash
wm clean --dry-run
```
Enable persistent dry-run mode in config:
```bash
# Edit %LOCALAPPDATA%\winmole\config.toml
dry_run = true
```

### Clear Confirmation Prompts
Every destructive operation requires explicit user confirmation with detailed previews.

---

## Building from Source

```bash
git clone https://github.com/lakshaymaurya-felt/winmole.git
cd winmole
go build -o wm.exe .
```

### Build with Version Info
```bash
go build -ldflags="-X github.com/lakshaymaurya-felt/winmole/cmd.appVersion=1.0.0" -o wm.exe .
```

---

## Configuration

WinMole stores its config at `%LOCALAPPDATA%\winmole\config.toml`:

```toml
# Enable persistent dry-run mode (preview only, never delete)
dry_run = false

# Whitelisted caches (never cleaned)
whitelist = [
    "C:\\Users\\You\\AppData\\Local\\SomeApp\\cache"
]

# Auto-update check interval (hours)
update_check_interval = 24
```

---

## License

[MIT](LICENSE) — Free to use, modify, and distribute.

---

## Credits

WinMole is inspired by and ported from [Mole](https://github.com/tw93/Mole) by [Tw93](https://github.com/tw93). Huge thanks to the original author for creating such a useful tool!

---

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or submit a PR.

---

**Made with ❤️ for Windows users who love clean systems.**
