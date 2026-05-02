//go:build linux

package optimize

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/lakshaymaurya-felt/purewin/internal/ui"
)

// ListStartupItems displays all enabled systemd services (startup items).
func ListStartupItems() {
	fmt.Println()
	fmt.Println(ui.SectionHeader("Startup Services (systemd)", 50))
	fmt.Println()

	cmd := exec.Command("systemctl", "list-unit-files", "--type=service", "--state=enabled", "--no-pager", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("  %s Failed to list services: %v\n", ui.IconError, err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println(ui.MutedStyle().Render("  No enabled services found."))
		return
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			name := fields[0]
			fmt.Printf("  %s %s\n", ui.MutedStyle().Render(ui.IconBullet), name)
		}
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle().Render(fmt.Sprintf("  %d enabled service(s)", len(lines))))
	fmt.Println()
}
