package tui

import "github.com/charmbracelet/lipgloss"

var (
	Subtle          = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	ContainerNugget = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1)
	ContainerStyleNoBorder = ContainerNugget.Copy()
	ContainerStyle         = ContainerNugget.Copy().
				Border(lipgloss.RoundedBorder(), true)
	HelpContainerStyle = ContainerStyle.Copy()
	FocusedStyle       = ContainerNugget.Copy().
				Border(lipgloss.RoundedBorder(), true).
				BorderForeground(lipgloss.Color("1"))
	DetailsViewStyle = lipgloss.NewStyle().PaddingLeft(2).
				MarginRight(1)
	ListViewStyle = lipgloss.NewStyle()
	SpinnerStyle  = lipgloss.NewStyle().
			MarginLeft(1).
			MarginTop(1)
	StatusNugget = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Padding(0, 1).MarginLeft(1)
	NameStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Border(lipgloss.RoundedBorder(), true)
	TitleStyle = StatusNugget.Copy().Background(lipgloss.Color("4")).MarginTop(1).MarginBottom(1)
	ListTitle  = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("4")).Padding(0, 1)
	FormStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingTop(1).
			MarginRight(1)
	InactiveStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ActiveStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	CheckboxCheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	OptionsViewStyle     = lipgloss.NewStyle()

	DialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	ButtonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	ActiveButtonStyle = ButtonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				Underline(true)

	ActiveRightButtonStyle = ActiveButtonStyle.Copy().
				MarginLeft(2)

	ActiveLeftButtonStyle = ActiveButtonStyle.Copy().
				MarginRight(2)
)
