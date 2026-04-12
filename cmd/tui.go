package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

func runTUI() (CommandParams, bool) {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		fmt.Fprintln(os.Stderr, "Error: TUI requires an interactive terminal. Use 'revshell custom' for non-interactive use.")
		return CommandParams{}, false
	}

	m := newTUIModel()
	// Cell motion mode is more reliable than all-motion mode in tmux for wheel capture.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error running TUI:", err)
		return CommandParams{}, false
	}

	final := result.(tuiModel)
	if final.aborted || !final.confirmed {
		return CommandParams{}, false
	}

	params := CommandParams{
		Name:      final.selected.typ,
		Method:    final.selected.method,
		IPAddress: final.ipInput.Value(),
		Port:      final.portInput.Value(),
		Shell:     final.shellValue(),
		Encoding:  final.encodings[final.encodingIdx],
	}
	return params, true
}
