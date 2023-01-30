package tui

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
	downloading
)

/* MODEL MANAGMENT */
var Models []tea.Model

const (
	Info status = iota
	Form
)

const (
	yes status = iota
	no
)

var widthDivisor = 2
var version = "0.0.2"

type errMsg error

type QueueItem struct {
	id             int
	videoId        string
	outputName     string
	embedThumbnail bool
	audioOnly      bool
	audioFormat    string
	extraCommands  string
}

func (i QueueItem) Title() string       { return i.outputName }
func (i QueueItem) Description() string { return i.videoId }
func (i QueueItem) FilterValue() string { return i.outputName }

type model struct {
	width, height int

	lists            []list.Model
	focused          status
	queueItemDetails QueueItem
	doneItemDetails  QueueItem
	downloadOutput   string
	viewport         viewport.Model
	startingDownload bool
	spinner          spinner.Model
	quitting         bool
	err              error
	ready            bool
	blockExit        bool
	dialogChoice     status
}

type downloadFinished struct {
	content string
}

type downloadingStatusUpdate struct {
	content string
}

func NewQueuedItem(videoId, outputName, audioFormat, extraCommands string, embedThumbnail, audioOnly bool) QueueItem {
	return QueueItem{
		videoId:        videoId,
		outputName:     outputName,
		embedThumbnail: embedThumbnail,
		audioOnly:      audioOnly,
		audioFormat:    audioFormat,
		extraCommands:  extraCommands,
	}
}

func (m model) executeDownload(item QueueItem) tea.Cmd {

	return func() tea.Msg {
		args := []string{}

		if item.audioOnly {
			args = append(args, "-x")
			args = append(args, "--audio-format")
			if len(item.audioFormat) > 0 {
				args = append(args, item.audioFormat)
			} else {
				args = append(args, "m4a")
			}
		}

		if len(item.extraCommands) > 0 {
			s := strings.Split(item.extraCommands, " ")
			args = append(args, s...)
		}

		if item.embedThumbnail {
			args = append(args, "--embed-thumbnail")
		}

		if len(item.outputName) > 0 {
			args = append(args, "-o")
			args = append(args, fmt.Sprintf("%s.%%(ext)s", item.outputName))
		}
		args = append(args, item.videoId)
		cmd := exec.Command("youtube-dl", args...) //nolint:gosec
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		_ = cmd.Start()

		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			t := scanner.Text()
			// fmt.Println(t)
			// m.downloadOutput = t
			P.Send(downloadingStatusUpdate{content: t})
		}
		// _ = cmd.Wait()
		// rescueStdout := os.Stdout
		// r, w, _ := os.Pipe()
		// os.Stdout = w
		//
		// cmd.Stdin = os.Stdin
		// cmd.Stdout = os.Stdout
		// // cmd.Stderr = os.Stderr
		// err := cmd.Run()
		// // if err != nil {
		// // 	data.UpdateQueueItemStatus(item.id, "error")
		// // 	fmt.Println(err)
		// // }
		//
		// w.Close()
		// out, _ := io.ReadAll(r)
		// os.Stdout = rescueStdout
		//
		// m.downloadOutput = string(out)
		// if err != nil {
		// 	data.UpdateQueueItemStatus(item.id, "error")
		// 	fmt.Println(err)
		// }

		data.UpdateQueueItemStatus(item.id, "completed")
		return downloadFinished{
			// content: string(out),
		}
	}
}

func InitialModel() *model {
	// s := spinner.New()
	// s.Spinner = spinner.Dot
	// s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &model{
		dialogChoice: 0,
	}
}

func (m *model) Next() {
	if m.focused == downloading {
		m.focused = queued
	} else {
		m.focused++
	}
}

func (m *model) Prev() {
	if m.focused == queued {
		m.focused = downloading
	} else {
		m.focused--
	}
}

func (m *model) PrevDialogChoice() {
	if m.dialogChoice == no {
		m.dialogChoice = yes
	} else {
		m.dialogChoice--
	}
}

func (m *model) NextDialogChoice() {
	if m.dialogChoice == yes {
		m.dialogChoice = no
	} else {
		m.dialogChoice++
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

	downloadingItems, err := data.GetAllQueueItems("downloading")
	if err != nil {
		fmt.Println(err.Error())
	}

	d := list.NewDefaultDelegate()

	c := lipgloss.Color("6")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy() // reuse the title style here

	defaultList := list.New([]list.Item{}, d, width-10, height/5)
	defaultList.SetShowHelp(false)
	m.lists = []list.Model{defaultList, defaultList, defaultList}

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
				extraCommands:  item.ExtraCommands,
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
				extraCommands:  item.ExtraCommands,
			})
	}

	downloadingItemsList := []list.Item{}
	for _, item := range downloadingItems {
		var outputSymbol string
		if item.Status == "downloading" {
			outputSymbol = "📀"
		} else if item.Status == "error" {
			outputSymbol = "❌"
		}
		downloadingItemsList = append(downloadingItemsList,
			QueueItem{
				id:             item.Id,
				videoId:        item.VideoId,
				outputName:     fmt.Sprintf("%s %s", outputSymbol, item.OutputName),
				embedThumbnail: item.EmbedThumbnail,
				audioOnly:      item.AudioOnly,
				audioFormat:    item.AudioFormat,
				extraCommands:  item.ExtraCommands,
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

	m.lists[downloading].Styles.Title = ListTitle
	m.lists[downloading].Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	m.lists[downloading].Title = "Download status"
	m.lists[downloading].SetItems(downloadingItemsList)
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
		key.WithHelp("d", "start download"),
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

func (m *model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeyMap.Left):
			if m.blockExit {
				m.PrevDialogChoice()
			} else {
				m.Prev()
			}
		case key.Matches(msg, DefaultKeyMap.Right):
			if m.blockExit {
				m.NextDialogChoice()
			} else {
				m.Next()
			}
		case key.Matches(msg, DefaultKeyMap.Quit):
			fmt.Println(len(m.lists[downloading].Items()))
			if len(m.lists[downloading].Items()) > 0 {
				m.blockExit = true
				return m, nil
			} else {
				return m, tea.Quit

			}
		case key.Matches(msg, DefaultKeyMap.Download):
			selectedItem := m.lists[m.focused].SelectedItem()
			item := selectedItem.(QueueItem)
			data.UpdateQueueItemStatus(item.id, "downloading")
			m.initLists(m.width, m.height)
			return m, m.executeDownload(item)
		case key.Matches(msg, DefaultKeyMap.Create):
			Models[Info] = m
			Models[Form] = NewForm()
			return Models[Form].Update(nil)
		case key.Matches(msg, DefaultKeyMap.Enter):
			if m.blockExit {
				if m.dialogChoice == yes {
					downloadingItems, err := data.GetAllQueueItems("downloading")
					if err != nil {
						fmt.Println(err.Error())
					}
					for _, item := range downloadingItems {
						data.UpdateQueueItemStatus(item.Id, "queued")
					}
					files, err := filepath.Glob("*.part")
					if err != nil {
						fmt.Println("Error finding part downloaded files")
					}
					for _, f := range files {
						if err := os.Remove(f); err != nil {
							fmt.Println("Error removing part downloaded files")
						}
					}
					return m, tea.Quit
				} else {
					m.blockExit = false
					return m, nil
				}

			} else {
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
						extraCommands:  item.extraCommands,
					}
				case done:
					m.doneItemDetails = QueueItem{
						id:             item.id,
						videoId:        item.videoId,
						outputName:     item.outputName,
						embedThumbnail: item.embedThumbnail,
						audioOnly:      item.audioOnly,
						audioFormat:    item.audioFormat,
						extraCommands:  item.extraCommands,
					}
				}
			}
		}
	case errMsg:
		m.err = msg
		return m, nil

	case QueueItem:
		m.initLists(m.width, m.height)

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

	case downloadingStatusUpdate:
		m.downloadOutput = msg.content
		m.viewport.SetContent(wordwrap.String(m.downloadOutput, m.width/widthDivisor-10))

	case downloadFinished:
		m.downloadOutput = msg.content
		// fmt.Println(m.downloadOutput)
		// m.viewport.SetContent(wordwrap.String(m.downloadOutput, m.width/widthDivisor-10))
		m.startingDownload = false
		m.initLists(m.width, m.height)
	}

	if m.ready {
		currList, cmd := m.lists[m.focused].Update(msg)
		m.lists[m.focused] = currList
		cmds = append(cmds, cmd)
	}
	m.spinner, cmd = m.spinner.Update(msg)
	// m.setViewportContent()
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

func (m model) downloadingView() string {
	return ListViewStyle.Render(m.lists[downloading].View())
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

func (m model) helpView() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n ↑/↓: navigate • ←/→: swap lists • c: create entry • d: download entry • q: quit\n 📀: downloading • ❌ error\n")
}

func (m model) dialogView() string {
	var (
		okButton, cancelButton string
	)

	if m.dialogChoice == yes {
		okButton = ActiveLeftButtonStyle.Render("Yes")
		cancelButton = ButtonStyle.Render("No")
	} else {
		okButton = ButtonStyle.Render("Yes")
		cancelButton = ActiveRightButtonStyle.Render("No")

	}

	question := lipgloss.NewStyle().Width(50).Align(lipgloss.Center).Render("You are currently downloading items.\nAre you sure you want to exit?")
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, okButton, cancelButton)
	ui := lipgloss.JoinVertical(lipgloss.Center, question, buttons)

	dialog := lipgloss.Place(m.width, 9,
		lipgloss.Center, lipgloss.Center,
		DialogBoxStyle.Render(ui),
	)
	return dialog
}

func (m model) footerView() string {
	info := DetailsViewStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	return lipgloss.JoinHorizontal(lipgloss.Center, info)
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

	if m.blockExit {
		return ContainerStyleNoBorder.Width(oneWide).Render(
			m.dialogView(),
		)
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
				ContainerStyle.Width(oneWide).Render(
					m.downloadingView(),
				),
				ContainerStyle.Width(oneWide).Render(
					TitleStyle.Render(m.downloadOutput),
				),
				HelpContainerStyle.Width(oneWide).Render(
					m.helpView(),
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
				ContainerStyle.Width(oneWide).Render(
					TitleStyle.Render(m.downloadOutput),
				),
				ContainerStyle.Width(oneWide).Render(
					m.downloadingView(),
				),
				HelpContainerStyle.Width(oneWide).Render(
					m.helpView(),
				),
			)
		case downloading:
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
					ContainerStyle.Width(twoWide).Render(
						lipgloss.JoinVertical(lipgloss.Left,
							m.doneView(),
							TitleStyle.Render("Details"),
							m.doneItemDetailsView(),
						),
					),
				),
				FocusedStyle.Width(oneWide).Render(
					m.downloadingView(),
				),
				ContainerStyle.Width(oneWide).Render(
					TitleStyle.Render(m.downloadOutput),
				),
				HelpContainerStyle.Width(oneWide).Render(
					m.helpView(),
				),
			)
		}
	} else {
		return "Loading"
	}
	return "..."
}

// VIEWS END
