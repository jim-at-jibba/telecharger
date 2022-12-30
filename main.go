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
	"github.com/charmbracelet/bubbles/textinput"
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
var models []tea.Model

const (
	info status = iota
	form
)

var widthDivisor = 2
var version = "0.0.1"

var (
	subtle          = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
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
	spinnerStyle  = lipgloss.NewStyle().
			MarginLeft(1).
			MarginTop(1)
	statusNugget = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFDF5")).Padding(0, 1).MarginLeft(1)
	nameStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Border(lipgloss.RoundedBorder(), true)
	titleStyle = statusNugget.Copy().Background(lipgloss.Color("4")).MarginTop(1).MarginBottom(1)
	listTitle  = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("4")).Padding(0, 1)
	formStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingTop(1).
			MarginRight(1)
	inactiveStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	activeStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	checkboxCheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	optionsViewStyle     = lipgloss.NewStyle()
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
			models[info] = m
			models[form] = NewForm()
			return models[form].Update(nil)
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
func NewQueuedItem(videoId, outputName, audioFormat string, embedThumbnail, audioOnly bool) QueueItem {
	return QueueItem{
		videoId:        videoId,
		outputName:     outputName,
		embedThumbnail: embedThumbnail,
		audioOnly:      audioOnly,
		audioFormat:    audioFormat,
	}
}

/* FORM MODEL */
type FormKeyMap struct {
	Quit  key.Binding
	Enter key.Binding
	Back  key.Binding
	Up    key.Binding
	Down  key.Binding
	Tab   key.Binding
}

var DefaultFormKeyMap = FormKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "go to next field/submit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "go back"),
	),
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select option"),
	),
}

// boolChoices
// embedThumbnail
// audioOnly

type option int

const (
	embedThumbnail option = iota
	audioOnly
)

type Form struct {
	videoId         textinput.Model
	outputName      textinput.Model
	audioFormat     textinput.Model
	choosingOptions bool
	choice          option
	boolChoices     []option
}

func (m Form) CreateQueuedItem() tea.Msg {
	// TODO: Create a new task
	task := NewQueuedItem(
		m.videoId.Value(),
		m.outputName.Value(),
		m.audioFormat.Value(),
		true,
		true,
	)
	return task
}

func NewForm() *Form {
	form := &Form{}
	form.choosingOptions = false
	form.videoId = textinput.New()
	form.videoId.Placeholder = "Youtube video url"
	form.videoId.Focus()
	form.outputName = textinput.New()
	form.outputName.Placeholder = "New name"
	form.audioFormat = textinput.New()
	form.audioFormat.Placeholder = "Audio Format (mp3, m4a)"
	return form
}

func (m Form) Init() tea.Cmd {
	return nil
}

func contains(s []option, find int) (bool, int) {
	for i, v := range s {
		if int(v) == find {
			return true, i
		}
	}

	return false, 0
}

func (m Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {

		case key.Matches(msg, DefaultFormKeyMap.Tab):
			if m.videoId.Focused() {
				m.videoId.Blur()
				m.outputName.Focus()
				return m, textinput.Blink
			} else if m.outputName.Focused() {
				m.outputName.Blur()
				m.audioFormat.Focus()
				return m, textinput.Blink
			} else if m.audioFormat.Focused() {
				m.audioFormat.Blur()
				m.choosingOptions = true
				m.choice = embedThumbnail
			} else {
				models[form] = m
				return models[info], m.CreateQueuedItem
			}
		case key.Matches(msg, DefaultFormKeyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, DefaultFormKeyMap.Down):
			if !m.choosingOptions {
				return m, nil
			}
			m.choice++
			if m.choice > 1 {
				m.choice = 1
			}
		case key.Matches(msg, DefaultFormKeyMap.Up):
			if !m.choosingOptions {
				return m, nil
			}
			m.choice--
			if m.choice < 0 {
				m.choice = 0
			}
		case key.Matches(msg, DefaultFormKeyMap.Enter):
			match, i := contains(m.boolChoices, int(m.choice))

			if match {
				m.boolChoices = append(m.boolChoices[:i], m.boolChoices[i+1:]...)
			} else {
				m.boolChoices = append(m.boolChoices, m.choice)
			}
		case key.Matches(msg, DefaultFormKeyMap.Back):
			models[form] = m
			return models[info], nil
		}
	}
	if m.videoId.Focused() {
		m.videoId, cmd = m.videoId.Update(msg)
		return m, cmd
	} else if m.outputName.Focused() {
		m.outputName, cmd = m.outputName.Update(msg)
		return m, cmd
	} else if m.audioFormat.Focused() {
		m.audioFormat, cmd = m.audioFormat.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func checkbox(label string, checked, selected bool, optionsActive bool) string {
	if selected {
		return checkboxCheckedStyle.Render("[x] " + label)
	} else if checked && optionsActive {
		return activeStyle.Render("[ ] " + label)
	}
	return inactiveStyle.Render("[ ] " + label)
}

func (m Form) choicesView() string {
	c := m.choice
	s := m.boolChoices

	containsEmbed, _ := contains(s, 0)
	containsAudioOnly, _ := contains(s, 1)

	choices := fmt.Sprintf(
		"%s\n%s\n",
		checkbox("Embed Thumbnail", c == 0, containsEmbed, m.choosingOptions),
		checkbox("Audio Only", c == 1, containsAudioOnly, m.choosingOptions),
	)

	return choices
}

func (m Form) formHelpView() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n ↑/↓: navigate options • enter: select/deselect option • tab: move to next/complete • q: quit\n")
}

func (m Form) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,

		containerStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				titleStyle.Render("Create new download"),
				formStyle.Render(
					lipgloss.JoinVertical(lipgloss.Left,
						m.videoId.View(),
						m.outputName.View(),
						m.audioFormat.View(),
					),
				),
				titleStyle.Render("Youtube-dl options"),
				formStyle.Render(
					lipgloss.JoinVertical(lipgloss.Left,
						optionsViewStyle.Render(
							m.choicesView(),
						),
					),
				),
			),
		),
		helpContainerStyle.Render(
			m.formHelpView(),
		),
	)
}

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
	models = []tea.Model{initialModel(), NewForm()}
	m := models[info]
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
