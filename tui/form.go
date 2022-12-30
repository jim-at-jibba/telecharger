package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

type FormModel struct {
	videoId         textinput.Model
	outputName      textinput.Model
	audioFormat     textinput.Model
	choosingOptions bool
	choice          option
	boolChoices     []option
}

func NewQueuedItem(videoId, outputName, audioFormat string, embedThumbnail, audioOnly bool) QueueItem {
	return QueueItem{
		videoId:        videoId,
		outputName:     outputName,
		embedThumbnail: embedThumbnail,
		audioOnly:      audioOnly,
		audioFormat:    audioFormat,
	}
}

func (m FormModel) CreateQueuedItem() tea.Msg {
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

func NewForm() *FormModel {
	form := &FormModel{}
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

func (m FormModel) Init() tea.Cmd {
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

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				Models[Form] = m
				return Models[Info], m.CreateQueuedItem
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
			Models[Form] = m
			return Models[Info], nil
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
		return CheckboxCheckedStyle.Render("[x] " + label)
	} else if checked && optionsActive {
		return ActiveStyle.Render("[ ] " + label)
	}
	return InactiveStyle.Render("[ ] " + label)
}

func (m FormModel) choicesView() string {
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

func (m FormModel) formHelpView() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("\n ↑/↓: navigate options • enter: select/deselect option • tab: move to next/complete • q: quit\n")
}

func (m FormModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,

		ContainerStyle.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				TitleStyle.Render("Create new download"),
				FormStyle.Render(
					lipgloss.JoinVertical(lipgloss.Left,
						m.videoId.View(),
						m.outputName.View(),
						m.audioFormat.View(),
					),
				),
				TitleStyle.Render("Youtube-dl options"),
				FormStyle.Render(
					lipgloss.JoinVertical(lipgloss.Left,
						OptionsViewStyle.Render(
							m.choicesView(),
						),
					),
				),
			),
		),
		HelpContainerStyle.Render(
			m.formHelpView(),
		),
	)
}
