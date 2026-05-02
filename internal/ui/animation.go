//go:build windows

package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"golang.org/x/sys/windows"
)

// ─── Mascot — Pixel ghost (6x7) ──────────────────────────────────────────────

var mascotLines = []string{
	`       ██            `,
	`     ██████          `,
	`    ████████         `,
	`   ██████████        `,
	`   ██████████        `,
	`  ████████████       `,
	`  ████▓▓██████       `,
	`  ████████████       `,
	`  ████▓██████        `,
	`  ████████████       `,
	`  ████████████       `,
	`  ████████████       `,
	`  ████████████       `,
	`  ████████████       `,
	`   ██████████        `,
	`    ████████         `,
}

var waveFrame1 = []string{
	` ████  ██  ████      `,
	`████  ████  ████     `,
	`██   ██  ██   ██    `,
}

var waveFrame2 = []string{
	`████  ████  ████     `,
	` ██  ██  ██  ██     `,
	`  ████  ████  ██    `,
}

var groundLine = `                      `
var moleLines = mascotLines

var brandLines = []string{
	"  ____                  __        ___       ",
	" |  _ \\ _   _ _ __ ___ \\ \\      / (_)_ __  ",
	" | |_) | | | | '__/ _ \\ \\ \\ /\\ / /| | '_ \\ ",
	" |  __/| |_| | | |  __/  \\ V  V / | | | | |",
	" |_|    \\__,_|_|  \\___|   \\_/\\_/  |_|_| |_|",
}

const tagline = "Deep clean and optimize your system."

// ─── Terminal Detection ──────────────────────────────────────────────────────

var vtEnabled bool

func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func EnableVTProcessing() bool {
	windows.SetConsoleOutputCP(65001)

	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32

	if err := windows.GetConsoleMode(stdout, &mode); err != nil {
		vtEnabled = false
		return false
	}

	if err := windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		vtEnabled = false
		return false
	}

	vtEnabled = true
	return true
}

func IsVTEnabled() bool {
	return vtEnabled
}

func GhostImage() string {
	return ""
}

// ─── Intro Animation ─────────────────────────────────────────────────────────

func ShowMoleIntro() {
	if !IsTerminal() {
		return
	}

	EnableVTProcessing()

	ghostStyle := lipgloss.NewStyle().Foreground(ColorSecondary)
	nameStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	fmt.Print("\033[2J\033[H")

	for _, line := range mascotLines {
		fmt.Println(ghostStyle.Render(line))
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println()
	for _, line := range brandLines {
		fmt.Println(nameStyle.Render(line))
		time.Sleep(30 * time.Millisecond)
	}

	fmt.Println()
	fmt.Println(MutedStyle().Italic(true).Render("  " + tagline))

	waveRow1 := len(mascotLines)
	waveRow2 := len(mascotLines) + 1
	waveRow3 := len(mascotLines) + 2

	for i := 0; i < 10; i++ {
		var frame []string
		if i%2 == 0 {
			frame = waveFrame1
		} else {
			frame = waveFrame2
		}

		fmt.Printf("\033[%d;1H", waveRow1)
		fmt.Println(ghostStyle.Render(frame[0]))
		fmt.Printf("\033[%d;1H", waveRow2)
		fmt.Println(ghostStyle.Render(frame[1]))
		fmt.Printf("\033[%d;1H", waveRow3)
		fmt.Println(ghostStyle.Render(frame[2]))

		time.Sleep(250 * time.Millisecond)
	}

	fmt.Printf("\033[%d;1H", waveRow1)
	fmt.Println(ghostStyle.Render(waveFrame1[0]))
	fmt.Printf("\033[%d;1H", waveRow2)
	fmt.Println(ghostStyle.Render(waveFrame1[1]))
	fmt.Printf("\033[%d;1H", waveRow3)
	fmt.Println(ghostStyle.Render(waveFrame1[2]))

	time.Sleep(400 * time.Millisecond)
	fmt.Print("\033[2J\033[H")
}

// ─── Brand Banner ────────────────────────────────────────────────────────────

func ShowBrandBanner() string {
	var b strings.Builder
	nameStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	for _, line := range brandLines {
		b.WriteString(nameStyle.Render(line))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	b.WriteString(MutedStyle().Italic(true).Render("  " + tagline))
	b.WriteByte('\n')

	b.WriteString(InfoStyle().Render("  https://github.com/lakshaymaurya-felt/purewin"))
	b.WriteByte('\n')

	return b.String()
}

// ─── Completion Banner ───────────────────────────────────────────────────────

func ShowCompletionBanner(freed int64, freeSpace int64) {
	fmt.Println()

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true).
		Render(IconCheck + " Cleanup Complete!"))
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("%s  %s\n",
		lipgloss.NewStyle().Foreground(ColorText).Render("Space freed:"),
		FormatSize(freed)))
	content.WriteString(fmt.Sprintf("%s  %s",
		lipgloss.NewStyle().Foreground(ColorText).Render("Free space: "),
		FormatSize(freeSpace)))

	fmt.Println(CardStyle().Width(50).Render(content.String()))
	fmt.Println()
}

// ─── Mascot Art (Static) ─────────────────────────────────────────────────────

func MoleArt() string {
	ghostStyle := lipgloss.NewStyle().Foreground(ColorSecondary)

	var b strings.Builder
	for _, line := range mascotLines {
		b.WriteString(ghostStyle.Render(line))
		b.WriteByte('\n')
	}
	for _, line := range waveFrame1 {
		b.WriteString(ghostStyle.Render(line))
		b.WriteByte('\n')
	}
	return b.String()
}
