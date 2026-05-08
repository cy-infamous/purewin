package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/cy-infamous/purewin/internal/config"
	"github.com/cy-infamous/purewin/internal/core"
	"github.com/cy-infamous/purewin/internal/purge"
	"github.com/cy-infamous/purewin/internal/ui"
	"github.com/cy-infamous/purewin/internal/util"
	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Clean project build artifacts",
	Long:  "Find and remove build artifacts (node_modules, target, build, dist, etc.) from project directories.",
	RunE:  runPurge,
}

func init() {
	purgeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without deleting")
	purgeCmd.Flags().Bool("paths", false, "Configure project scan directories")
	purgeCmd.Flags().Int("min-age", 7, "Minimum age in days (recent projects are skipped)")
	purgeCmd.Flags().String("min-size", "", "Minimum artifact size to show (e.g., 50MB)")
}

func runPurge(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check --paths flag
	managePaths, _ := cmd.Flags().GetBool("paths")
	if managePaths {
		return managePurgePaths(cfg)
	}

	// Start scanning
	fmt.Println()
	fmt.Println(ui.SectionHeader("Project Purge", 50))
	fmt.Println()

	spinner := ui.NewInlineSpinner()
	spinner.Start("Scanning for project artifacts...")

	// Get scan paths
	scanPaths := getScanPaths(cfg)
	if len(scanPaths) == 0 {
		return fmt.Errorf("no scan paths configured — run 'pw purge --paths' to configure")
	}

	// Scan for artifacts
	artifacts, err := purge.ScanProjects(scanPaths)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	spinner.Stop(fmt.Sprintf("Found %d artifacts", len(artifacts)))

	if len(artifacts) == 0 {
		fmt.Println()
		fmt.Println(ui.SuccessStyle().Render(fmt.Sprintf("  %s No project artifacts found!", ui.IconCheck)))
		fmt.Println()
		return nil
	}

	// Convert to selector items
	items := artifactsToSelectorItems(artifacts)

	// Show selector
	selected, err := ui.RunSelector(items, "Select artifacts to delete:")
	if err != nil {
		return fmt.Errorf("selector error: %w", err)
	}

	if selected == nil || len(selected) == 0 {
		fmt.Println()
		fmt.Println(ui.MutedStyle().Render("  No artifacts selected. Exiting."))
		fmt.Println()
		return nil
	}

	// Convert back to artifacts
	selectedArtifacts := make([]purge.ProjectArtifact, 0, len(selected))
	for _, item := range selected {
		// Find the artifact by path
		for _, artifact := range artifacts {
			if artifact.ArtifactPath == item.Value {
				selectedArtifacts = append(selectedArtifacts, artifact)
				break
			}
		}
	}

	// Show summary
	fmt.Println()
	totalSize := int64(0)
	for _, artifact := range selectedArtifacts {
		totalSize += artifact.Size
	}

	fmt.Printf("  %s\n", ui.BoldStyle().Render(fmt.Sprintf("Will delete %d artifacts (%s)",
		len(selectedArtifacts), core.FormatSize(totalSize))))
	fmt.Println()

	// Confirm
	if !dryRun {
		confirmed, err := ui.Confirm("Proceed with deletion?")
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		if !confirmed {
			fmt.Println()
			fmt.Println(ui.MutedStyle().Render("  Cancelled."))
			fmt.Println()
			return nil
		}
	}

	// Delete
	fmt.Println()
	freed, count, purgeErr := purge.PurgeArtifacts(selectedArtifacts, dryRun)

	if dryRun {
		fmt.Println()
		fmt.Println(ui.InfoStyle().Render("  [DRY RUN] No files were deleted"))
		fmt.Printf("  Would free: %s from %d artifacts\n", core.FormatSize(freed), count)
		fmt.Println()
	} else {
		fmt.Println()
		if purgeErr != nil {
			fmt.Printf("%s Completed with errors: %v\n", ui.WarningStyle().Render(ui.IconWarning), purgeErr)
		} else {
			fmt.Printf("%s Success!\n", ui.SuccessStyle().Render(ui.IconSuccess))
		}
		fmt.Printf("  Freed: %s from %d artifacts\n", ui.SuccessStyle().Render(core.FormatSize(freed)), count)
		fmt.Println()
	}

	return nil
}

// getScanPaths returns the list of paths to scan for projects.
func getScanPaths(cfg *config.Config) []string {
	// Try to load custom paths first
	customPaths, err := purge.LoadCustomScanPaths(cfg.ConfigDir)
	if err == nil && len(customPaths) > 0 {
		return customPaths
	}

	// Fall back to defaults
	return purge.GetDefaultScanPaths()
}

// managePurgePaths opens the purge_paths file in the default editor.
func managePurgePaths(cfg *config.Config) error {
	pathsFile := filepath.Join(cfg.ConfigDir, "purge_paths")

	// Create file with defaults if it doesn't exist
	if _, err := os.Stat(pathsFile); os.IsNotExist(err) {
		defaults := purge.GetDefaultScanPaths()
		if err := purge.SaveCustomScanPaths(cfg.ConfigDir, defaults); err != nil {
			return fmt.Errorf("failed to create purge_paths: %w", err)
		}
	}

	// Try to open in default editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	// Try common editors before falling back to notepad.
	if editor == "" {
		for _, candidate := range []string{"code", "code-insiders", "notepad++", "notepad.exe"} {
			if _, err := exec.LookPath(candidate); err == nil {
				editor = candidate
				break
			}
		}
	}
	if editor == "" {
		editor = "notepad.exe"
	}

	cmd := exec.Command(editor, pathsFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println()
	fmt.Printf("  Opening %s in %s...\n", pathsFile, editor)
	fmt.Println()

	if err := cmd.Run(); err != nil {
		fmt.Printf("%s Failed to open editor: %v\n", ui.WarningStyle().Render(ui.IconWarning), err)
		fmt.Printf("  Edit manually: %s\n", pathsFile)
	}

	return nil
}

// artifactsToSelectorItems converts artifacts to selector items.
func artifactsToSelectorItems(artifacts []purge.ProjectArtifact) []ui.SelectorItem {
	// Group by artifact type
	typeGroups := make(map[string][]purge.ProjectArtifact)
	for _, artifact := range artifacts {
		typeGroups[artifact.ArtifactType] = append(typeGroups[artifact.ArtifactType], artifact)
	}

	// Sort types
	types := make([]string, 0, len(typeGroups))
	for t := range typeGroups {
		types = append(types, t)
	}
	sort.Strings(types)

	// Build items
	items := make([]ui.SelectorItem, 0, len(artifacts))
	for _, t := range types {
		group := typeGroups[t]
		// Sort by size descending
		sort.Slice(group, func(i, j int) bool {
			return group[i].Size > group[j].Size
		})

		for _, artifact := range group {
			// Create label with project name
			projectName := filepath.Base(artifact.ProjectPath)
			label := fmt.Sprintf("%s/%s", projectName, artifact.ArtifactType)

			// Age
			age := time.Since(artifact.ModTime)
			ageStr := util.FormatDuration(age)

			item := ui.SelectorItem{
				Label:       label,
				Description: fmt.Sprintf("%s • %s old", artifact.ArtifactPath, ageStr),
				Value:       artifact.ArtifactPath,
				Size:        core.FormatSize(artifact.Size),
				Selected:    !artifact.IsRecent, // Don't select recent artifacts by default
				Disabled:    false,
				Category:    artifact.ArtifactType,
			}

			items = append(items, item)
		}
	}

	return items
}
