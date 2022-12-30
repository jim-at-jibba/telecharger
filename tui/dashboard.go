package tui

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

/* MODEL MANAGMENT */
var Models []tea.Model

const (
	Info status = iota
	Form
)

var widthDivisor = 2
var version = "0.0.1"

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

	lists            []list.Model
	focused          status
	queueItemDetails QueueItem
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

func InitialModel() model {
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

	doneItems, err := data.GetAllQueueItems("completed")
	if err != nil {
		fmt.Println(err.Error())
	}

	d := list.NewDefaultDelegate()

	c := lipgloss.Color("6")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy() // reuse the title style here

	defaultList := list.New([]list.Item{}, d, width-10, height/5)
	defaultList.SetShowHelp(false)
	m.lists = []list.Model{defaultList, defaultList}

	queueItemsList := []list.Item{}
	for _, item := range queueItems {
		queueItemsList = append(queueItemsList,
			QueueItem{
				id:             item.Id,
				videoId:        item.VideoId,
				outputName:     item.OutputName,
				embedThumbnail: item.EmbedThumbnail,
				audioOnly:      item.AudioOnly,
				audioFormat:    item.AudioFormat,
			})
	}

	doneItemsList := []list.Item{}
	for _, item := range doneItems {
		doneItemsList = append(doneItemsList,
			QueueItem{
				id:             item.Id,
				videoId:        item.VideoId,
				outputName:     item.OutputName,
				embedThumbnail: item.EmbedThumbnail,
				audioOnly:      item.AudioOnly,
				audioFormat:    item.AudioFormat,
			})
	}
	m.lists[queued].Styles.Title = ListTitle
	m.lists[queued].Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	m.lists[queued].Title = "Queued"
	m.lists[queued].SetItems(queueItemsList)

	m.lists[done].Styles.Title = ListTitle
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
	Create   key.Binding
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
	Create: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "create new queued item"),
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
		case key.Matches(msg, DefaultKeyMap.Create):
			Models[Info] = m
			Models[Form] = NewForm()
			return Models[Form].Update(nil)
		case key.Matches(msg, DefaultKeyMap.Enter):
			selectedItem := m.lists[m.focused].SelectedItem()
			item := selectedItem.(QueueItem)
			switch m.focused {
			case queued:
				m.queueItemDetails = QueueItem{
					id:             item.id,
					videoId:        item.videoId,
					outputName:     item.outputName,
					embedThumbnail: item.embedThumbnail,
					audioOnly:      item.audioOnly,
					audioFormat:    item.audioFormat,
				}
			case done:
				m.doneItemDetails = QueueItem{
					id:             item.id,
					videoId:        item.videoId,
					outputName:     item.outputName,
					embedThumbnail: item.embedThumbnail,
					audioOnly:      item.audioOnly,
					audioFormat:    item.audioFormat,
				}
			}
		}
	case errMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.width, m.height = msg.Width, msg.Height
			ContainerStyle.Height(msg.Height / 5)
			ContainerStyle.Width(msg.Width - 10)
			FocusedStyle.Height(msg.Height / 5)
			FocusedStyle.Width(msg.Width - 10)
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
	name := NameStyle.Width(oneWide).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			acsi,
			version,
		),
	)

	return name
}

func (m model) queueView() string {
	return ListViewStyle.Render(m.lists[queued].View())
}

func (m model) doneView() string {
	return ListViewStyle.Render(m.lists[done].View())
}

func (m model) queueItemDetailsView() string {
	videoId := fmt.Sprintf("Video Id: %s", m.queueItemDetails.videoId)
	outputName := fmt.Sprintf("Outname: %s", m.queueItemDetails.outputName)
	audioFormat := fmt.Sprintf("AudioFormat: %s", m.queueItemDetails.audioFormat)
	audioOnly := fmt.Sprintf("AudioOnly: %s", strconv.FormatBool(m.queueItemDetails.audioOnly))
	return DetailsViewStyle.Render(
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
	return DetailsViewStyle.Render(
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
	info := DetailsViewStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
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

					FocusedStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.queueView(),
							TitleStyle.Render("Details"),
							m.queueItemDetailsView(),
						),
					),
					ContainerStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.doneView(),
							TitleStyle.Render("Details"),
							m.doneItemDetailsView(),
						),
					),
				),
				HelpContainerStyle.Width(oneWide).Render(
					m.helpView(),
				),
				ContainerStyle.Width(oneWide).Render(
					lipgloss.JoinVertical(lipgloss.Left,
						TitleStyle.Render("Download status"),
						m.viewportView(),
					),
				),
			)
		case done:
			return lipgloss.JoinVertical(lipgloss.Left,
				m.nameView(),
				lipgloss.JoinHorizontal(lipgloss.Left,

					ContainerStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.queueView(),
							TitleStyle.Render("Details"),
							m.queueItemDetailsView(),
						),
					),
					FocusedStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.doneView(),
							TitleStyle.Render("Details"),
							m.doneItemDetailsView(),
						),
					),
				),
				HelpContainerStyle.Width(oneWide).Render(
					m.helpView(),
				),
				ContainerStyle.Width(oneWide).Render(
					lipgloss.JoinVertical(lipgloss.Left,
						TitleStyle.Render("Download status"),
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
