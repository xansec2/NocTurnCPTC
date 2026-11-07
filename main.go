// Command nocturncptc is an interactive terminal UI for assembling pentest
// tool command lines. Pick a tool, toggle and fill in flags, watch the command
// build live, then yank it to the clipboard (ctrl+y) and run it in a separate
// terminal.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	res, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// Print the last assembled command so it survives leaving the alt screen
	// (copy with ctrl+y inside the TUI, or print-and-quit with ctrl+o).
	if m, ok := res.(model); ok && strings.TrimSpace(m.finalCmd) != "" {
		fmt.Println(m.finalCmd)
	}
}
