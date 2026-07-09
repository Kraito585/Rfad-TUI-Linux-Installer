package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Input struct {
	Model textinput.Model
}

// Добавили width в аргументы
func NewInput(placeholder string, width int) Input {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256
	ti.Width = width // Динамическая ширина!

	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	return Input{Model: ti}
}

func (i Input) Init() tea.Cmd { return textinput.Blink }

func (i *Input) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	i.Model, cmd = i.Model.Update(msg)
	return cmd
}

func (i Input) View() string    { return i.Model.View() }
func (i *Input) Focus() tea.Cmd { return i.Model.Focus() }
func (i *Input) Blur()          { i.Model.Blur() }
func (i Input) Value() string   { return i.Model.Value() }
