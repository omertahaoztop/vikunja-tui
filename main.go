package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/omertahaoztop/vikunja-tui/internal/config"
	"github.com/omertahaoztop/vikunja-tui/internal/tui"
	"github.com/omertahaoztop/vikunja-tui/internal/upgrade"
	"github.com/omertahaoztop/vikunja-tui/internal/vikunja"
)

var version = "dev"

func main() {
	args := os.Args[1:]

	for _, a := range args {
		switch a {
		case "--version", "-V":
			fmt.Printf("vikunja-tui %s\n", version)
			return
		case "--upgrade":
			if err := upgrade.SelfUpgrade(version); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client, err := vikunja.FromConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(client))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
