package shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lakshaymaurya-felt/winmole/internal/ui"
)

// ─── Vercel-Inspired Cappuccino Shell Palette ────────────────────────────────
// Extends the global palette with shell-specific accent styles.

var (
	// Prompt accent: dusty mauve (matches status dashboard).
	promptColor = lipgloss.AdaptiveColor{Light: "#8c6f7e", Dark: "#b89aab"}

	// Completion highlight: warm periwinkle.
	compHighlight = lipgloss.AdaptiveColor{Light: "#7a7899", Dark: "#a3a1be"}

	// Dim chrome: warm gray.
	dimColor = lipgloss.AdaptiveColor{Light: "#8a7e76", Dark: "#6b6360"}

	// Styles.
	promptStyle     = lipgloss.NewStyle().Foreground(promptColor).Bold(true)
	dimTextStyle    = lipgloss.NewStyle().Foreground(dimColor)
	compActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#3b2618", Dark: "#f2e8dc"}).
			Background(compHighlight).
			Bold(true).
			Padding(0, 1)
	compInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#7d6e63", Dark: "#a89889"}).
				Padding(0, 1)
	compDescStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)
	statusBarStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)
	bannerNameStyle = lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true)
	bannerTagStyle = lipgloss.NewStyle().
			Foreground(ui.ColorMuted)
	outputStyle = lipgloss.NewStyle().
			Foreground(ui.ColorText)
	scrollIndicator = lipgloss.NewStyle().
			Foreground(ui.ColorMuted).
			Italic(true)
)

// View renders the complete shell interface.
func (m ShellModel) View() string {
	if m.Quitting {
		return ""
	}

	w := m.Width
	if w < 40 {
		w = 40
	}

	var s strings.Builder

	// ── Welcome Banner (only on first launch, before any output) ──
	if len(m.OutputLines) <= 1 {
		s.WriteString(m.renderBanner(w))
	}

	// ── Output Viewport ──
	s.WriteString(m.renderOutput(w))

	// ── Completions Popup (overlays above prompt) ──
	if m.completions.IsOpen() {
		s.WriteString(m.renderCompletions(w))
	}

	// ── Prompt Line ──
	s.WriteString(m.renderPrompt(w))

	// ── Status Bar ──
	s.WriteString(m.renderStatusBar(w))

	return s.String()
}

// ─── Banner ──────────────────────────────────────────────────────────────────

func (m ShellModel) renderBanner(w int) string {
	var s strings.Builder

	s.WriteString("\n")

	// Compact mole face.
	mole := dimTextStyle.Render("  /\\_/\\")
	s.WriteString(mole + "\n")
	mole2 := dimTextStyle.Render(" ( o.o )")
	s.WriteString(mole2 + "\n")
	mole3 := dimTextStyle.Render("  > ^ <")
	s.WriteString(mole3 + "\n")

	s.WriteString("\n")
	s.WriteString("  " + bannerNameStyle.Render("WinMole") +
		"  " + bannerTagStyle.Render(m.Version) + "\n")
	s.WriteString("  " + dimTextStyle.Render("Deep clean and optimize your Windows.") + "\n")
	s.WriteString("  " + dimTextStyle.Render("Type / for commands, /help for details.") + "\n")
	s.WriteString("\n")
	s.WriteString(dimTextStyle.Render("  "+strings.Repeat("─", w-4)) + "\n")
	s.WriteString("\n")

	return s.String()
}

// ─── Output Viewport ─────────────────────────────────────────────────────────

func (m ShellModel) renderOutput(w int) string {
	if len(m.OutputLines) == 0 {
		return ""
	}

	vpHeight := m.viewportHeight()
	lines := m.OutputLines

	// Calculate visible window.
	totalLines := len(lines)
	startIdx := totalLines - vpHeight - m.scrollPos
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + vpHeight
	if endIdx > totalLines {
		endIdx = totalLines
	}

	var s strings.Builder

	for i := startIdx; i < endIdx; i++ {
		// Truncate long lines.
		line := lines[i]
		if len(line) > w-2 {
			line = line[:w-5] + "..."
		}
		s.WriteString(outputStyle.Render(line) + "\n")
	}

	// Scroll indicator.
	if m.scrollPos > 0 {
		hiddenAbove := startIdx
		indicator := fmt.Sprintf("  ↑ %d more lines above", hiddenAbove)
		s.WriteString(scrollIndicator.Render(indicator) + "\n")
	}

	return s.String()
}

// ─── Completions Popup ───────────────────────────────────────────────────────

func (m ShellModel) renderCompletions(w int) string {
	filtered := m.completions.Filtered()
	if len(filtered) == 0 {
		return ""
	}

	cursor := m.completions.Cursor()

	var s strings.Builder
	s.WriteString("\n")

	// Border top.
	boxWidth := 50
	if w < 56 {
		boxWidth = w - 6
	}
	s.WriteString("  " + dimTextStyle.Render(strings.Repeat("─", boxWidth)) + "\n")

	// Render each completion item (max 8 visible).
	maxVisible := 8
	if len(filtered) < maxVisible {
		maxVisible = len(filtered)
	}

	// Scroll the list if cursor is beyond visible range.
	startIdx := 0
	if cursor >= maxVisible {
		startIdx = cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := startIdx; i < endIdx; i++ {
		cmd := filtered[i]
		name := "/" + cmd.Name
		desc := cmd.Description

		// Truncate description to fit.
		maxDesc := boxWidth - len(name) - 6
		if maxDesc < 4 {
			desc = ""
		} else if len(desc) > maxDesc {
			desc = desc[:maxDesc-3] + "..."
		}

		if i == cursor {
			// Active item.
			s.WriteString("  " + compActiveStyle.Render(fmt.Sprintf("%-12s", name)) +
				"  " + compDescStyle.Render(desc) + "\n")
		} else {
			// Inactive item.
			s.WriteString("  " + compInactiveStyle.Render(fmt.Sprintf("%-12s", name)) +
				"  " + compDescStyle.Render(desc) + "\n")
		}
	}

	// Border bottom.
	s.WriteString("  " + dimTextStyle.Render(strings.Repeat("─", boxWidth)) + "\n")

	return s.String()
}

// ─── Prompt ──────────────────────────────────────────────────────────────────

func (m ShellModel) renderPrompt(w int) string {
	prompt := promptStyle.Render("wm ❯ ")
	input := m.textInput.View()
	return "\n" + prompt + input + "\n"
}

// ─── Status Bar ──────────────────────────────────────────────────────────────

func (m ShellModel) renderStatusBar(w int) string {
	var parts []string

	// Admin status.
	if m.IsAdmin {
		adminStyle := lipgloss.NewStyle().Foreground(ui.ColorWarning)
		parts = append(parts, adminStyle.Render("admin"))
	}

	// Hints.
	parts = append(parts, "/ commands")
	parts = append(parts, "↑↓ history")
	parts = append(parts, "pgup/pgdn scroll")
	parts = append(parts, "ctrl+c quit")

	return statusBarStyle.Render("  "+strings.Join(parts, "  │  ")) + "\n"
}
