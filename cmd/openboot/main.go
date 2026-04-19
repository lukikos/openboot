package main

import (
	"os"

	"golang.org/x/term"

	"github.com/openbootdotdev/openboot/internal/cli"
)

func main() {
	// Save terminal state before any TUI (huh/bubbletea) runs.
	// Ensures the terminal is restored even if a TUI component crashes
	// or exits without proper cleanup (e.g., when invoked via curl|bash).
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) { //nolint:gosec // os.Stdin.Fd() returns a valid file descriptor; uintptr fits in int on all supported platforms
		if oldState, err := term.GetState(fd); err == nil {
			defer term.Restore(fd, oldState) //nolint:errcheck // best-effort terminal restore on exit
		}
	}

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
