package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cy-infamous/purewin/internal/analyze"
	"github.com/cy-infamous/purewin/internal/ui"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [path]",
	Short: "Explore disk usage",
	Long: `Interactive disk space analyzer with visual tree view.

Defaults to the current working directory when no path is given.

Examples:
  pw analyze              Analyze current directory
  pw analyze D:\Projects  Analyze a specific directory
  pw analyze C:\          Analyze an entire drive`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runAnalyze,
}

func init() {
	analyzeCmd.Flags().Int("depth", 0, "Maximum directory depth to display")
	analyzeCmd.Flags().String("min-size", "", "Minimum size to display (e.g., 100MB)")
	analyzeCmd.Flags().StringSlice("exclude", nil, "Directories to exclude from scan")
}

func runAnalyze(cmd *cobra.Command, args []string) {
	// Determine target path (default: current working directory).
	target := ""
	if len(args) > 0 {
		target = args[0]
	}
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot determine current directory: %v\n", err)
			os.Exit(1)
		}
		target = cwd
	}

	// Validate the path exists.
	if _, err := os.Stat(target); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access %s: %v\n", target, err)
		os.Exit(1)
	}

	// Parse exclude list.
	exclude, _ := cmd.Flags().GetStringSlice("exclude")

	// Try loading from cache first.
	root, err := analyze.LoadCache(target)
	if err != nil {
		// No valid cache — run a fresh scan with a progress spinner.
		scanner := analyze.NewScanner(8, exclude)

		done := make(chan struct{})
		go func() {
			frame := 0
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					frame = (frame + 1) % len(ui.SpinnerFrames)
					count := scanner.ScannedCount()
					fmt.Fprintf(os.Stderr, "\r  %s Scanning %s … %d entries",
						ui.SpinnerFrames[frame], target, count)
				}
			}
		}()

		root, err = scanner.Scan(target)
		close(done)
		fmt.Fprint(os.Stderr, "\r\033[K") // clear spinner line

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
			os.Exit(1)
		}

		// Persist results for next time.
		_ = analyze.SaveCache(root, target)
	}

	// Launch the TUI.
	model := analyze.NewAnalyzeModel(root)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
