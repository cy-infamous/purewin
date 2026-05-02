package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/lakshaymaurya-felt/purewin/internal/config"
	"github.com/lakshaymaurya-felt/purewin/internal/ui"
	"github.com/lakshaymaurya-felt/purewin/pkg/whitelist"
)

var whitelistCmd = &cobra.Command{
	Use:   "whitelist",
	Short: "Manage cleanup whitelist",
	Long:  "Add, remove, or list path patterns that are protected from cleanup operations.",
	Run:   runWhitelistList,
}

var whitelistAddCmd = &cobra.Command{
	Use:   "add <pattern>",
	Short: "Add a path pattern to the whitelist",
	Args:  cobra.ExactArgs(1),
	Run:   runWhitelistAdd,
}

var whitelistRemoveCmd = &cobra.Command{
	Use:   "remove <pattern>",
	Short: "Remove a path pattern from the whitelist",
	Args:  cobra.ExactArgs(1),
	Run:   runWhitelistRemove,
}

var whitelistListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all whitelist patterns",
	Run:   runWhitelistList,
}

var whitelistResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset whitelist to default patterns",
	Run:   runWhitelistReset,
}

func init() {
	whitelistCmd.AddCommand(whitelistAddCmd)
	whitelistCmd.AddCommand(whitelistRemoveCmd)
	whitelistCmd.AddCommand(whitelistListCmd)
	whitelistCmd.AddCommand(whitelistResetCmd)
}

// loadWhitelist loads config and whitelist. The whitelist file is stored
// alongside the config in the PureWin config directory.
func loadWhitelist() (*whitelist.Whitelist, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	wlPath := filepath.Join(cfg.ConfigDir, "whitelist.txt")
	return whitelist.Load(wlPath)
}

func runWhitelistList(cmd *cobra.Command, args []string) {
	wl, err := loadWhitelist()
	if err != nil {
		fmt.Printf("%s %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	patterns := wl.List()
	fmt.Println()
	if len(patterns) == 0 {
		fmt.Println(ui.MutedStyle().Render("  No patterns in whitelist."))
	} else {
		fmt.Println(ui.BoldStyle().Render(fmt.Sprintf("  Whitelist Patterns (%d):", len(patterns))))
		fmt.Println()
		for _, p := range patterns {
			fmt.Printf("  %s %s\n", ui.MutedStyle().Render(ui.IconBullet), p)
		}
	}
	fmt.Println()
}

func runWhitelistAdd(cmd *cobra.Command, args []string) {
	wl, err := loadWhitelist()
	if err != nil {
		fmt.Printf("%s %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	pattern := args[0]
	if err := wl.Add(pattern); err != nil {
		fmt.Printf("%s %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	if err := wl.Save(); err != nil {
		fmt.Printf("%s Failed to save: %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	fmt.Printf("%s Added: %s\n", ui.SuccessStyle().Render(ui.IconSuccess), ui.BoldStyle().Render(pattern))
}

func runWhitelistRemove(cmd *cobra.Command, args []string) {
	wl, err := loadWhitelist()
	if err != nil {
		fmt.Printf("%s %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	pattern := args[0]
	if err := wl.Remove(pattern); err != nil {
		fmt.Printf("%s %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	if err := wl.Save(); err != nil {
		fmt.Printf("%s Failed to save: %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	fmt.Printf("%s Removed: %s\n", ui.SuccessStyle().Render(ui.IconSuccess), ui.BoldStyle().Render(pattern))
}

func runWhitelistReset(cmd *cobra.Command, args []string) {
	confirmed, err := ui.Confirm("Reset whitelist to defaults? This removes all custom patterns.")
	if err != nil || !confirmed {
		fmt.Println(ui.MutedStyle().Render("  Cancelled."))
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("%s %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	// Delete the whitelist file so it gets recreated with defaults on next load.
	wlPath := filepath.Join(cfg.ConfigDir, "whitelist.txt")
	if err := os.Remove(wlPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("%s Failed to reset: %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	// Trigger recreation with defaults.
	wl, err := whitelist.Load(wlPath)
	if err != nil {
		fmt.Printf("%s Failed to recreate whitelist: %v\n", ui.ErrorStyle().Render(ui.IconError), err)
		return
	}

	fmt.Printf("%s Whitelist reset to %d default patterns\n",
		ui.SuccessStyle().Render(ui.IconSuccess), len(wl.List()))
}
