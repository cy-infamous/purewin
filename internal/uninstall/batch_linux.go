//go:build linux

package uninstall

import (
	"fmt"
	"os/exec"

	"github.com/cy-infamous/purewin/internal/ui"
)

func RunBatchUninstall(apps []InstalledApp, dryRun bool) error {
	if len(apps) == 0 {
		fmt.Println(ui.MutedStyle().Render("  No applications found."))
		return nil
	}

	fmt.Println()
	fmt.Println(ui.BoldStyle().Render("  Installed applications:"))
	fmt.Println()

	max := len(apps)
	if max > 30 {
		max = 30
	}

	for i := 0; i < max; i++ {
		app := apps[i]
		sizeStr := ""
		if app.EstimatedSize > 0 {
			sizeStr = fmt.Sprintf(" (%s)", ui.FormatSize(app.EstimatedSize))
		}
		num := ui.MutedStyle().Render(fmt.Sprintf("  %d.", i+1))
		fmt.Printf("%s %s %s%s\n", num, app.Name, ui.MutedStyle().Render(app.Version), sizeStr)
	}

	if len(apps) > 30 {
		fmt.Printf("  %s... and %d more (use --search to filter)\n",
			ui.MutedStyle().Render(""), len(apps)-30)
	}

	if dryRun {
		fmt.Println()
		fmt.Println(ui.MutedStyle().Render("  DRY RUN — no changes made."))
		return nil
	}

	confirmed, err := ui.Confirm("Remove all listed applications?")
	if err != nil || !confirmed {
		fmt.Println(ui.MutedStyle().Render("  Cancelled."))
		return nil
	}

	pkgMgr := detectPkgManager()
	var pkgNames []string
	for _, app := range apps {
		pkgNames = append(pkgNames, app.Name)
	}

	spin := ui.NewInlineSpinner()
	spin.Start(fmt.Sprintf("Removing %d package(s)...", len(pkgNames)))

	var cmd *exec.Cmd
	switch pkgMgr {
	case "dnf", "yum":
		cmd = exec.Command(pkgMgr, append([]string{"remove", "-y"}, pkgNames...)...)
	case "apt":
		cmd = exec.Command("apt", append([]string{"remove", "-y"}, pkgNames...)...)
	case "pacman":
		cmd = exec.Command("pacman", append([]string{"-R", "--noconfirm"}, pkgNames...)...)
	default:
		spin.StopWithError("No supported package manager")
		return fmt.Errorf("no supported package manager")
	}

	if err := cmd.Run(); err != nil {
		spin.StopWithError(fmt.Sprintf("Removal failed: %s", err))
		return fmt.Errorf("removal failed: %w", err)
	}

	spin.Stop(fmt.Sprintf("Removed %d package(s)", len(pkgNames)))
	return nil
}
