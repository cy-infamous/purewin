<p align="center">
  <img src="assets/logo.svg" alt="PureWin" width="96" height="96" />
</p>

<h1 align="center">PureWin</h1>

<p align="center"><strong>Your system, purified.</strong></p>

<p align="center">
  <a href="https://github.com/lakshaymaurya-felt/purewin/actions/workflows/ci.yml"><img src="https://github.com/lakshaymaurya-felt/purewin/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/lakshaymaurya-felt/purewin"><img src="https://img.shields.io/github/go-mod/go-version/lakshaymaurya-felt/purewin" alt="Go Version" /></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT" /></a>
</p>

---

## What is PureWin?

**PureWin** is a system optimization toolkit built in pure Go. It delivers blazing-fast system cleanup, surgical app removal, real-time monitoring, and performance optimization in a single, dependency-free binary.

It runs on **Windows** and **Linux** with platform-appropriate behavior for each:

| Feature | Windows | Linux |
|---------|---------|-------|
| System cleanup | Temp files, Windows Update cache, delivery optimization, memory dumps | Package cache (`/var/cache`), journal logs, thumbnails, trash |
| Browser cleanup | Chrome, Firefox, Edge cache/cookies/history | Chrome, Firefox cache/cookies/history |
| Dev tool cleanup | `node_modules`, `.gradle`, `.nuget`, `target/` | `node_modules`, `.gradle`, `target/`, Go module cache |
| Service management | Restart DNS, DHCP, Windows Search, Windows Update | Restart NetworkManager, systemd-resolved |
| Optimization | DISM cleanup, SFC scan, icon cache rebuild, search index rebuild | Package cache cleanup, SSD TRIM, icon cache rebuild, journal vacuum, kernel cleanup |
| Startup items | Registry startup programs | Enabled systemd services |
| App uninstall | Registry cleanup, leftover file removal | — |
| Installer cleanup | Orphaned `.exe`, `.msi`, `.msix` | — |
| System monitoring | CPU, memory, disk, network, GPU, battery | CPU, memory, disk, network |
| Disk analyzer | Interactive treemap | Interactive treemap |

Forget bloated GUI tools that slow you down. PureWin gives you a beautiful terminal interface that's faster, more powerful, and infinitely more satisfying to use.

Native. Fast. Cross-platform. This is system optimization done right.

---

## Preview

<p align="center">
  <img src="assets/welcome.png" alt="PureWin Welcome Screen" width="720" />
  <br />
  <em>Welcome screen with mascot, command cards, and quick-start tips</em>
</p>

<p align="center">
  <img src="assets/commands.png" alt="PureWin Command Autocomplete" width="720" />
  <br />
  <em>Interactive shell with slash-command autocomplete</em>
</p>

<p align="center">
  <img src="assets/system-overview.png" alt="PureWin System Overview" width="720" />
  <br />
  <em>Real-time system overview — health score, CPU, memory, disk, and network at a glance</em>
</p>

<p align="center">
  <img src="assets/cpu-monitor.png" alt="PureWin CPU Monitor" width="720" />
  <br />
  <em>Per-core CPU monitoring with live history charts</em>
</p>

<p align="center">
  <img src="assets/disk-analyzer.png" alt="PureWin Disk Analyzer" width="720" />
  <br />
  <em>Interactive disk space analyzer — drill into directories, sort by size, delete on the spot</em>
</p>

---

## Features

- **× Deep System Cleanup** — Obliterate temp files, caches, logs, browser data, and dev tool artifacts
- **× Complete App Removal** — Uninstall apps and wipe their registry entries, configs, and hidden remnants (Windows)
- **◇ Disk Space Analysis** — Interactive treemap visualization that shows exactly where your storage went
- **→ System Optimization** — Refresh caches, restart services, optimize performance with one command
- **● Real-Time Monitoring** — Live dashboard tracking CPU, memory, disk, network, GPU, and battery
- **◆ Installer Cleanup** — Hunt down and remove orphaned installer files lurking in Downloads (Windows)
- **× Dev Tool Cleanup** — Purge build artifacts from node_modules, target/, .gradle, .nuget, Go modules, and more
- **◈ Safety First** — Whitelist protection, dry-run mode, and NEVER_DELETE safeguards for critical paths
- **◇ Beautiful TUI** — Rich interactive menus powered by Bubble Tea that make cleanup feel like a game
- **○ Self-Updating** — Check for and install updates directly from GitHub releases
- **→ Shell Completion** — Tab completion for all commands (PowerShell / Bash)
- **› Interactive Shell** — Persistent shell with slash-command autocomplete for power users

---

## Quick Install

### Via Go Install
```bash
go install github.com/lakshaymaurya-felt/purewin@latest
```

### Via PowerShell (Windows — one-liner)
```powershell
irm https://raw.githubusercontent.com/lakshaymaurya-felt/purewin/main/scripts/install.ps1 | iex
```

### Via GitHub Releases
Download the latest release for your platform from [Releases](https://github.com/lakshaymaurya-felt/purewin/releases):

| Platform | File |
|----------|------|
| Windows x64 | `pw-windows-amd64.zip` |
| Linux x64 | `pw-linux-amd64.tar.gz` |

Extract and add to your PATH.

---

## Usage

### Interactive Menu
Launch the full-screen interactive menu:

```bash
pw
```

Navigate with arrow keys, select with Enter, and watch PureWin work its magic.

### Direct Commands
Run specific operations directly:

```bash
# Preview what will be cleaned (safe mode)
pw clean --dry-run

# Clean everything (requires admin for system caches)
pw clean --all

# Clean only browser caches
pw clean --browser

# Uninstall an app completely (Windows)
pw uninstall

# Analyze disk usage with visual treemap
pw analyze C:\          # Windows
pw analyze /            # Linux

# Monitor system health in real-time
pw status

# Remove orphaned installer files (Windows)
pw installer

# Optimize system performance
pw optimize

# View enabled startup services
pw optimize --startup

# Clean dev tool build artifacts
pw purge

# Manage whitelist
pw whitelist add /path/to/protect
pw whitelist remove /path/to/unprotect
pw whitelist list
pw whitelist reset

# Update PureWin to latest version
pw update

# Show version info
pw version
```

---

## Commands Reference

| Command | Description | Admin Required |
|---------|-------------|----------------|
| `clean` | Deep cleanup of caches, logs, temp files, browser leftovers | Partial* |
| `uninstall` | Remove apps completely with registry and leftover cleanup (Windows) | Yes |
| `analyze` | Interactive disk space analyzer with visual tree view | No |
| `optimize` | Refresh caches, restart services, optimize performance | Yes |
| `optimize --startup` | List enabled startup programs / services | No |
| `optimize --services` | Restart system services only | Yes |
| `optimize --maintenance` | Run maintenance tasks only | Yes |
| `status` | Real-time dashboard for CPU, memory, disk, network, GPU | No |
| `installer` | Find and remove installer files (Windows) | No |
| `purge` | Clean project build artifacts (node_modules, target/, etc.) | No |
| `whitelist` | Manage whitelist (add, remove, list, reset) | No |
| `update` | Check for and install latest PureWin version | No |
| `remove` | Uninstall PureWin and remove config/cache | No |
| `completion` | Generate shell tab completion | No |
| `version` | Show installed version | No |

*`clean --system` requires admin; `--user`, `--browser`, `--dev` do not.

---

## Platform Behavior

### Linux Optimization (`pw optimize`)

| Task | What it does |
|------|-------------|
| Flush DNS cache | `resolvectl flush-caches` or `systemd-resolve --flush-caches` |
| Restart services | `systemctl restart NetworkManager` and `systemd-resolved` |
| Package cache cleanup | Auto-detects apt, dnf, pacman, or zypper and cleans cache + autoremove |
| SSD TRIM | `fstrim -av` to optimize flash storage |
| Rebuild icon cache | `gtk-update-icon-cache` + `update-mime-database` |
| Vacuum journal logs | `journalctl --vacuum-time=7d` |
| Remove orphaned packages | Package manager autoremove (apt/dnf) |

### Linux Cleanup Targets (`pw clean`)

| Category | Paths |
|----------|-------|
| System caches | `/var/cache`, `/var/tmp`, `/var/log`, `/var/spool` |
| User caches | `~/.cache`, `~/.local/share/Trash`, `~/.thumbnails` |
| Dev tools | `node_modules`, `target/`, `.gradle`, Go module cache, `~/.cargo/registry` |
| Browser data | Chrome, Firefox, Chromium cache directories |

---

## Safety

PureWin is engineered with safety as the foundation:

### NEVER_DELETE Protection
Critical system paths are hardcoded as off-limits. PureWin will refuse to touch:

**Windows:**
- `C:\Windows`
- `C:\Program Files`
- `C:\Program Files (x86)`
- User profile root directories

**Linux:**
- `/usr`, `/bin`, `/sbin`, `/lib`, `/lib64`
- `/etc`, `/boot`, `/dev`, `/proc`, `/sys`
- `/home` root, `/root`

### Whitelist System
Protect specific paths you want to keep:
```bash
# Add a path to the whitelist
pw whitelist add ~/.cache/some-app

# Remove a path from the whitelist
pw whitelist remove ~/.cache/some-app

# View current whitelist
pw whitelist list

# Clear the entire whitelist
pw whitelist reset
```
Whitelisted items are persisted in your config and skipped during cleanup.

### Dry-Run Mode
Preview exactly what will be deleted before committing:
```bash
pw clean --dry-run
```
Enable persistent dry-run mode in config:
```json
{
  "dry_run": true
}
```

### Clear Confirmation Prompts
Every destructive operation requires explicit user confirmation with detailed previews. No surprises.

---

## Building from Source

```bash
git clone https://github.com/lakshaymaurya-felt/purewin.git
cd purewin

# Build for current platform
go build -o pw .

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o pw.exe .
```

### Build with Version Info
```bash
go build -ldflags="-X github.com/lakshaymaurya-felt/purewin/cmd.appVersion=1.0.0" -o pw .
```

---

## Configuration

Config is stored at:

| Platform | Path |
|----------|------|
| Windows | `%LOCALAPPDATA%\purewin\config.json` |
| Linux | `~/.config/purewin/config.json` |

```json
{
  "dry_run": false,
  "whitelist": [
    "/home/user/.cache/some-app"
  ],
  "update_check_interval": 24
}
```

---

## License

[MIT](LICENSE) — Free to use, modify, and distribute.

---

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to open an issue or submit a PR.

---

**Built for users who refuse to settle for bloat.**
