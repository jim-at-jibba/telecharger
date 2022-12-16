package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	listViewStyle = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1).
			Border(lipgloss.RoundedBorder(), true)
	spacerStyle = lipgloss.NewStyle().
			MarginTop(1)
	// statusNugget   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Padding(0, 1)
	nameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			PaddingLeft(5).
			Border(lipgloss.RoundedBorder(), true)
	// encodingStyle = statusNugget.Copy().Background(lipgloss.Color("#A550DF")).Align(lipgloss.Right)
	// statusText    = statusBarStyle.Copy()
	// datetimeStyle = statusNugget.Copy().Background(lipgloss.Color("#6124DF"))
)

type errMsg error

type model struct {
	width, height int

	queue    list.Model
	spinner  spinner.Model
	quitting bool
	err      error
}

var quitKeys = key.NewBinding(
	key.WithKeys("q", "esc", "ctrl+c"),
	key.WithHelp("", "press q to quit"),
)

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Redis Viewer"

	return model{spinner: s, queue: l}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if key.Matches(msg, quitKeys) {
			m.quitting = true
			return m, tea.Quit

		}
		return m, nil
	case errMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

// VIEWS START
func (m model) nameView() string {

	name := nameStyle.Width(m.width / 5).Render(`
  _       _           _
 | |     | |         | |
 | |_ ___| | ___  ___| |__   __ _ _ __ __ _  ___ _ __
 | __/ _ \ |/ _ \/ __| '_ \ / _' | '__/ _' |/ _ \ '__|
 | ||  __/ |  __/ (__| | | | (_| | | | (_| |  __/ |
  \__\___|_|\___|\___|_| |_|\__,_|_|  \__, |\___|_|
                                       __/ |
                                      |___/
    `)

	return name
}

func (m model) queueView() string {
	return listViewStyle.Width(m.width / 5).Render("A lovely list")
}

func (m model) queueItemDetailsView() string {
	return listViewStyle.Width(m.width / 2).Render("A lovely list item details")
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	if m.quitting {
		return "Exiting...\n"
	}
	return lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left,
			m.nameView(),
			m.queueView(),
		),
		lipgloss.JoinVertical(lipgloss.Left,
			m.queueItemDetailsView(),
		),
	)
}

// VIEWS END

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
