package banner

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// Print displays the startup banner.
func Print(version, commit, date string) {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	fmt.Println(style.Render("homerun2-scout"))
	fmt.Printf("  version: %s  commit: %s  date: %s\n\n", version, commit, date)
}
