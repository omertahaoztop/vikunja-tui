package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Palette struct {
	Name       string
	BgPrimary  color.Color
	BgSecond   color.Color
	BgTertiary color.Color
	FgPrimary  color.Color
	FgSecond   color.Color
	Accent     color.Color
	AccentAlt  color.Color
	Green      color.Color
	Yellow     color.Color
	Red        color.Color
	Border     color.Color
	BorderFoc  color.Color
}

var Palettes = []Palette{
	{
		Name:       "GitHub Dark",
		BgPrimary:  lipgloss.Color("#0d1117"),
		BgSecond:   lipgloss.Color("#161b22"),
		BgTertiary: lipgloss.Color("#21262d"),
		FgPrimary:  lipgloss.Color("#e6edf3"),
		FgSecond:   lipgloss.Color("#8b949e"),
		Accent:     lipgloss.Color("#58a6ff"),
		AccentAlt:  lipgloss.Color("#388bfd"),
		Green:      lipgloss.Color("#3fb950"),
		Yellow:     lipgloss.Color("#d29922"),
		Red:        lipgloss.Color("#f85149"),
		Border:     lipgloss.Color("#30363d"),
		BorderFoc:  lipgloss.Color("#388bfd"),
	},
	{
		Name:       "Dracula",
		BgPrimary:  lipgloss.Color("#282a36"),
		BgSecond:   lipgloss.Color("#21222c"),
		BgTertiary: lipgloss.Color("#343746"),
		FgPrimary:  lipgloss.Color("#f8f8f2"),
		FgSecond:   lipgloss.Color("#6272a4"),
		Accent:     lipgloss.Color("#bd93f9"),
		AccentAlt:  lipgloss.Color("#ff79c6"),
		Green:      lipgloss.Color("#50fa7b"),
		Yellow:     lipgloss.Color("#f1fa8c"),
		Red:        lipgloss.Color("#ff5555"),
		Border:     lipgloss.Color("#44475a"),
		BorderFoc:  lipgloss.Color("#bd93f9"),
	},
	{
		Name:       "Nord",
		BgPrimary:  lipgloss.Color("#2e3440"),
		BgSecond:   lipgloss.Color("#3b4252"),
		BgTertiary: lipgloss.Color("#434c5e"),
		FgPrimary:  lipgloss.Color("#eceff4"),
		FgSecond:   lipgloss.Color("#81a1c1"),
		Accent:     lipgloss.Color("#88c0d0"),
		AccentAlt:  lipgloss.Color("#5e81ac"),
		Green:      lipgloss.Color("#a3be8c"),
		Yellow:     lipgloss.Color("#ebcb8b"),
		Red:        lipgloss.Color("#bf616a"),
		Border:     lipgloss.Color("#4c566a"),
		BorderFoc:  lipgloss.Color("#88c0d0"),
	},
	{
		Name:       "Gruvbox",
		BgPrimary:  lipgloss.Color("#282828"),
		BgSecond:   lipgloss.Color("#3c3836"),
		BgTertiary: lipgloss.Color("#504945"),
		FgPrimary:  lipgloss.Color("#ebdbb2"),
		FgSecond:   lipgloss.Color("#a89984"),
		Accent:     lipgloss.Color("#83a598"),
		AccentAlt:  lipgloss.Color("#d3869b"),
		Green:      lipgloss.Color("#b8bb26"),
		Yellow:     lipgloss.Color("#fabd2f"),
		Red:        lipgloss.Color("#fb4934"),
		Border:     lipgloss.Color("#665c54"),
		BorderFoc:  lipgloss.Color("#83a598"),
	},
}

// Theme bundles a palette with pre-built lipgloss styles.
type Theme struct {
	P Palette

	Header     lipgloss.Style
	HeaderLogo lipgloss.Style
	HeaderSub  lipgloss.Style
	StatusOK   lipgloss.Style
	StatusSync lipgloss.Style
	Footer     lipgloss.Style
	KeyHint    lipgloss.Style

	Logo         lipgloss.Style
	TreeItem     lipgloss.Style
	TreeSelected lipgloss.Style

	Column      lipgloss.Style
	ColumnFoc   lipgloss.Style
	ColHeader   lipgloss.Style
	ColHeaderF  lipgloss.Style
	Card        lipgloss.Style
	CardFocused lipgloss.Style
	CardDone    lipgloss.Style

	BadgePrio  lipgloss.Style
	BadgeDue   lipgloss.Style
	BadgeLabel lipgloss.Style

	Modal      lipgloss.Style
	ModalTitle lipgloss.Style
	ModalBody  lipgloss.Style

	NotifInfo  lipgloss.Style
	NotifWarn  lipgloss.Style
	NotifError lipgloss.Style
}

func NewTheme(p Palette) Theme {
	return Theme{
		P: p,
		Header: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.FgPrimary),
		HeaderLogo: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.Accent).Bold(true),
		HeaderSub: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.FgSecond),
		StatusOK: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.Green),
		StatusSync: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.Yellow),
		Footer: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.FgSecond),
		KeyHint: lipgloss.NewStyle().
			Background(p.BgSecond).Foreground(p.Accent),

		Logo: lipgloss.NewStyle().
			Foreground(p.Accent).Bold(true).Align(lipgloss.Center),
		TreeItem: lipgloss.NewStyle().Foreground(p.FgSecond),
		TreeSelected: lipgloss.NewStyle().
			Foreground(p.FgPrimary).Background(p.AccentAlt).Bold(true),

		Column: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(p.Border).
			Padding(0, 1),
		ColumnFoc: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(p.BorderFoc).
			Padding(0, 1),
		ColHeader: lipgloss.NewStyle().
			Foreground(p.FgPrimary).Background(p.BgTertiary).Bold(true).
			Padding(0, 1),
		ColHeaderF: lipgloss.NewStyle().
			Foreground(p.Accent).Background(p.BgTertiary).Bold(true).
			Padding(0, 1),
		Card: lipgloss.NewStyle().Foreground(p.FgSecond),
		CardFocused: lipgloss.NewStyle().
			Foreground(p.FgPrimary).Background(p.BgTertiary).Bold(true),
		CardDone: lipgloss.NewStyle().
			Foreground(p.FgSecond).Strikethrough(true),

		BadgePrio: lipgloss.NewStyle().
			Foreground(p.BgPrimary).Background(p.Red).Bold(true).Padding(0, 1),
		BadgeDue: lipgloss.NewStyle().
			Foreground(p.Yellow),
		BadgeLabel: lipgloss.NewStyle().
			Foreground(p.AccentAlt),

		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(p.BorderFoc).
			Background(p.BgSecond).Padding(1, 2),
		ModalTitle: lipgloss.NewStyle().
			Foreground(p.FgPrimary).Bold(true).Align(lipgloss.Center),
		ModalBody: lipgloss.NewStyle().Foreground(p.FgSecond),

		NotifInfo: lipgloss.NewStyle().
			Foreground(p.Green).Background(p.BgTertiary).Padding(0, 1),
		NotifWarn: lipgloss.NewStyle().
			Foreground(p.Yellow).Background(p.BgTertiary).Padding(0, 1),
		NotifError: lipgloss.NewStyle().
			Foreground(p.Red).Background(p.BgTertiary).Padding(0, 1),
	}
}
