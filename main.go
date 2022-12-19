package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jim-at-jibba/telecharger/data"
)

var (
	containerNugget = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1)
	containerStyle = containerNugget.Copy().
			Border(lipgloss.RoundedBorder(), true)
	detailsViewStyle = lipgloss.NewStyle().PaddingLeft(2).
				MarginRight(1)
	listViewStyle = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1)
	spacerStyle = lipgloss.NewStyle().
			MarginTop(1)
	statusNugget = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Padding(0, 1)
	nameStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			PaddingLeft(5).
			Border(lipgloss.RoundedBorder(), true)
	// encodingStyle = statusNugget.Copy().Background(lipgloss.Color("#A550DF")).Align(lipgloss.Right)
	// statusText    = statusBarStyle.Copy()
	titleStyle = statusNugget.Copy().Background(lipgloss.Color("#6124DF"))
)

type errMsg error

type QueueItem struct {
	id             int
	videoId        string
	outputName     string
	embedThumbnail bool
	audioOnly      bool
	audioFormat    string
}

func (i QueueItem) Title() string       { return i.outputName }
func (i QueueItem) Description() string { return i.outputName }
func (i QueueItem) FilterValue() string { return i.outputName }

type model struct {
	width, height int

	queue            list.Model
	queueItemDetails QueueItem
	spinner          spinner.Model
	quitting         bool
	err              error
}

var quitKeys = key.NewBinding(
	key.WithKeys("q", "esc", "ctrl+c"),
	key.WithHelp("", "press q to quit"),
)

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	queueItems, err := data.GetAllQueueItems()
	if err != nil {
		fmt.Printf(err.Error())
	}

	items := []list.Item{}
	for _, item := range queueItems {
		items = append(items, QueueItem{id: item.Id, videoId: item.VideoId, outputName: item.OutputName, embedThumbnail: item.EmbedThumbnail, audioOnly: item.AudioOnly, audioFormat: item.AudioFormat})
	}

	l := list.New(items, list.NewDefaultDelegate(), 100, 500)
	l.Title = "Queued"

	return model{spinner: s, queue: l, queueItemDetails: QueueItem{id: queueItems[0].Id, videoId: queueItems[0].VideoId, outputName: queueItems[0].OutputName, embedThumbnail: queueItems[0].EmbedThumbnail, audioOnly: queueItems[0].AudioOnly, audioFormat: queueItems[0].AudioFormat}}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
		case tea.KeyCtrlC, tea.KeyEsc:
			cmd = tea.Quit
			return m, cmd
		case tea.KeyEnter:
			selectedItem := m.queue.SelectedItem()
			item := selectedItem.(QueueItem)
			m.queueItemDetails = QueueItem{id: item.id, videoId: item.videoId, outputName: item.outputName, embedThumbnail: item.embedThumbnail, audioOnly: item.audioOnly, audioFormat: item.audioFormat}
		}
	case errMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.queue.SetSize(msg.Width/3, msg.Height/3)
		return m, nil
	}
	m.queue, _ = m.queue.Update(msg)
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// VIEWS START
func (m model) nameView() string {
	h, _ := nameStyle.GetFrameSize()

	name := nameStyle.Width(m.width - h).Render(`
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
	return listViewStyle.Render(m.queue.View())
}

func (m model) queueItemDetailsView() string {
	videoId := fmt.Sprintf("Video Id: %s", m.queueItemDetails.videoId)
	outputName := fmt.Sprintf("Outname: %s", m.queueItemDetails.outputName)
	audioFormat := fmt.Sprintf("AudioFormat: %s", m.queueItemDetails.audioFormat)
	audioOnly := fmt.Sprintf("AudioOnly: %s", strconv.FormatBool(m.queueItemDetails.audioOnly))
	return detailsViewStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			outputName,
			videoId,
			audioFormat,
			audioOnly,
		),
	)
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	if m.quitting {
		return "Exiting...\n"
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.nameView(),
		lipgloss.JoinHorizontal(lipgloss.Left,
			containerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					m.queueView(),
					titleStyle.Margin(1, 2).Render("Details"),
					m.queueItemDetailsView(),
				),
			),
			containerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					m.queue.View(),
					titleStyle.Margin(1, 2).Render("Details"),
					m.queueItemDetailsView(),
				),
			),
		),
	)
}

// VIEWS END

func main() {
	data.OpenDatabase()
	data.CreateQueueTable()
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
