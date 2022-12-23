package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jim-at-jibba/telecharger/data"
	"github.com/muesli/reflow/wordwrap"
)

const useHighPerformanceRenderer = false

type status int

const (
	queued status = iota
	done
)

var widthDivisor = 2
var version = "0.0.1"

var (
	containerNugget = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1)
	containerStyle = containerNugget.Copy().
			Border(lipgloss.RoundedBorder(), true)
	helpContainerStyle = containerStyle.Copy()
	focusedStyle       = containerNugget.Copy().
				Border(lipgloss.RoundedBorder(), true).
				BorderForeground(lipgloss.Color("1"))
	detailsViewStyle = lipgloss.NewStyle().PaddingLeft(2).
				MarginRight(1)
	listViewStyle = lipgloss.NewStyle()
	// PaddingRight(1).
	// MarginRight(1)
	spinnerStyle = lipgloss.NewStyle().
			MarginLeft(1).
			MarginTop(1)
	statusNugget = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Padding(0, 1).MarginLeft(1)
	nameStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
		// PaddingLeft(5).
		Border(lipgloss.RoundedBorder(), true)
	// encodingStyle = statusNugget.Copy().Background(lipgloss.Color("#A550DF")).Align(lipgloss.Right)
	// statusText    = statusBarStyle.Copy()
	titleStyle = statusNugget.Copy().Background(lipgloss.Color("4"))
	listTitle  = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("4")).Padding(0, 1)
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

	lists   []list.Model
	focused status

	// queue            list.Model
	queueItemDetails QueueItem
	// done             list.Model
	doneItemDetails  QueueItem
	viewport         viewport.Model
	downloadOutput   string
	startingDownload bool
	spinner          spinner.Model
	quitting         bool
	err              error
	ready            bool
}

type downloadFinished struct {
	content string
}

func (m model) executeDownload() tea.Cmd {
	return func() tea.Msg {
		// command := `youtube-dl https://www.youtube.com/watch?v=J38Yq85ZoyY`
		cmd := exec.Command("youtube-dl", "-x", "https://www.youtube.com/watch?v=J38Yq85ZoyY") //nolint:gosec
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err := cmd.Run()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		m.downloadOutput = string(out)
		if err != nil {
			fmt.Println(err)
		}

		return downloadFinished{
			content: string(out),
		}
	}
}

// var quitKeys = key.NewBinding(
// 	key.WithKeys("q", "esc", "ctrl+c"),
// 	key.WithHelp("", "press q to quit"),
// )

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{spinner: s}
}

func (m *model) Next() {
	if m.focused == queued {
		m.focused = done
	} else {
		m.focused = queued
	}
}

func (m *model) Prev() {
	if m.focused == queued {
		m.focused = done
	} else {
		m.focused = queued
	}
}

func (m *model) initLists(width, height int) {
	queueItems, err := data.GetAllQueueItems("queued")
	if err != nil {
		fmt.Println(err.Error())
	}
	if err != nil {
		fmt.Println(err.Error())
	}
	doneItems, err := data.GetAllQueueItems("completed")
	if err != nil {
		fmt.Println(err.Error())
	}
	d := list.NewDefaultDelegate()

	// Change colors
	c := lipgloss.Color("6")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy() // reuse the title style here

	defaultList := list.New([]list.Item{}, d, width-10, height/5)
	defaultList.SetShowHelp(false)
	m.lists = []list.Model{defaultList, defaultList}

	queueItemsList := []list.Item{}
	for _, item := range queueItems {
		queueItemsList = append(queueItemsList, QueueItem{id: item.Id, videoId: item.VideoId, outputName: item.OutputName, embedThumbnail: item.EmbedThumbnail, audioOnly: item.AudioOnly, audioFormat: item.AudioFormat})
	}

	doneItemsList := []list.Item{}
	for _, item := range doneItems {
		doneItemsList = append(doneItemsList, QueueItem{id: item.Id, videoId: item.VideoId, outputName: item.OutputName, embedThumbnail: item.EmbedThumbnail, audioOnly: item.AudioOnly, audioFormat: item.AudioFormat})
	}
	m.lists[queued].Styles.Title = listTitle
	m.lists[queued].Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	m.lists[queued].Title = "Queued"
	m.lists[queued].SetItems(queueItemsList)

	m.lists[done].Styles.Title = listTitle
	m.lists[done].Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	m.lists[done].Title = "Done"
	m.lists[done].SetItems(doneItemsList)
}

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Quit     key.Binding
	Download key.Binding
	Enter    key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("→/l", "move right"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Download: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("↓/j", "start download"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "show more info"),
	),
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Left):
			m.Prev()
		case key.Matches(msg, DefaultKeyMap.Right):
			m.Next()
		case key.Matches(msg, DefaultKeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, DefaultKeyMap.Download):
			return m, m.executeDownload()
		case key.Matches(msg, DefaultKeyMap.Enter):
			selectedItem := m.lists[m.focused].SelectedItem()
			item := selectedItem.(QueueItem)
			switch m.focused {
			case queued:
				m.queueItemDetails = QueueItem{id: item.id, videoId: item.videoId, outputName: item.outputName, embedThumbnail: item.embedThumbnail, audioOnly: item.audioOnly, audioFormat: item.audioFormat}
			case done:
				m.doneItemDetails = QueueItem{id: item.id, videoId: item.videoId, outputName: item.outputName, embedThumbnail: item.embedThumbnail, audioOnly: item.audioOnly, audioFormat: item.audioFormat}
			}
		}
	case errMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.width, m.height = msg.Width, msg.Height
			containerStyle.Height(msg.Height / 5)
			containerStyle.Width(msg.Width - 10)
			focusedStyle.Height(msg.Height / 5)
			focusedStyle.Width(msg.Width - 10)
			m.initLists(msg.Width, msg.Height)
			m.viewport = viewport.New(msg.Width, msg.Height/7)
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.downloadOutput)
			m.ready = true

		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height / 7
		}

	case downloadFinished:
		m.downloadOutput = msg.content
		m.startingDownload = false

		m.viewport.SetContent(wordwrap.String(m.downloadOutput, m.width/widthDivisor-10))
	}
	if m.ready {
		currList, cmd := m.lists[m.focused].Update(msg)
		m.lists[m.focused] = currList
		cmds = append(cmds, cmd)
	}
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// VIEWS START
func (m model) nameView() string {
	oneWide := int(float64(m.width - 8))
	version := fmt.Sprintf("  Version: %s\n  Author: James Best", version)

	acsi := `
  _       _           _
 | |     | |         | |
 | |_ ___| | ___  ___| |__   __ _ _ __ __ _  ___ _ __
 | __/ _ \ |/ _ \/ __| '_ \ / _' | '__/ _' |/ _ \ '__|
 | ||  __/ |  __/ (__| | | | (_| | | | (_| |  __/ |
  \__\___|_|\___|\___|_| |_|\__,_|_|  \__, |\___|_|
                                       __/ |
                                      |___/
    `
	name := nameStyle.Width(oneWide).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			acsi,
			version,
		),
	)

	return name
}

func (m model) queueView() string {
	return listViewStyle.Render(m.lists[queued].View())
}

func (m model) doneView() string {
	return listViewStyle.Render(m.lists[done].View())
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

func (m model) doneItemDetailsView() string {
	videoId := fmt.Sprintf("Video Id: %s", m.doneItemDetails.videoId)
	outputName := fmt.Sprintf("Outname: %s", m.doneItemDetails.outputName)
	audioFormat := fmt.Sprintf("AudioFormat: %s", m.doneItemDetails.audioFormat)
	audioOnly := fmt.Sprintf("AudioOnly: %s", strconv.FormatBool(m.doneItemDetails.audioOnly))
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

func (m model) footerView() string {
	info := detailsViewStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	return lipgloss.JoinHorizontal(lipgloss.Center, info)
}

func (m model) viewportView() string {
	var info string
	if m.startingDownload && m.downloadOutput == "" {
		info = fmt.Sprintf("\n\n   %s Downloading...\n\n", m.spinner.View())
	} else {
		info = lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(),
			m.footerView(),
		)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, info)
}

func (m model) helpView() string {
	// TODO: use the keymaps to populate the help string
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n ↑/↓: navigate • ←/→: swap lists • c: create entry • d: download entry • q: quit\n")
}

func (m model) View() string {
	twoWide := int(math.Floor(float64(m.width-10) / 2))
	oneWide := int(float64(m.width - 8))
	if m.err != nil {
		return m.err.Error()
	}
	if m.quitting {
		return "Exiting...\n"
	}

	if m.ready {
		switch m.focused {
		case queued:
			return lipgloss.JoinVertical(lipgloss.Left,
				m.nameView(),
				lipgloss.JoinHorizontal(lipgloss.Left,

					focusedStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.queueView(),
							titleStyle.Render("Details"),
							m.queueItemDetailsView(),
						),
					),
					containerStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.doneView(),
							titleStyle.Render("Details"),
							m.doneItemDetailsView(),
						),
					),
				),
				helpContainerStyle.Width(oneWide).Render(
					m.helpView(),
				),
				containerStyle.Width(oneWide).Render(
					lipgloss.JoinVertical(lipgloss.Left,
						titleStyle.Render("Download status"),
						m.viewportView(),
					),
				),
			)
		case done:
			return lipgloss.JoinVertical(lipgloss.Left,
				m.nameView(),
				lipgloss.JoinHorizontal(lipgloss.Left,

					containerStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.queueView(),
							titleStyle.Render("Details"),
							m.queueItemDetailsView(),
						),
					),
					focusedStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.doneView(),
							titleStyle.Render("Details"),
							m.doneItemDetailsView(),
						),
					),
				),
				helpContainerStyle.Width(oneWide).Render(
					m.helpView(),
				),
				containerStyle.Width(oneWide).Render(
					lipgloss.JoinVertical(lipgloss.Left,
						titleStyle.Render("Download status"),
						m.viewportView(),
					),
				),
			)
		}
	} else {
		return "Loading"
	}
	return "..."
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
