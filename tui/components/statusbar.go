package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBar — это наш "React-компонент"
type StatusBar struct {
	progBar  progress.Model
	Message  string
	IsActive bool
	width    int
}

// NewStatusBar — аналог конструктора компонента
func NewStatusBar() StatusBar {
	return StatusBar{
		progBar:  progress.New(progress.WithDefaultGradient()),
		Message:  "Инициализация...",
		IsActive: true,
		width:    60, // Базовая ширина
	}
}

func (s *StatusBar) SetWidth(w int) {
	s.width = w
	s.progBar.Width = w - 4 // Оставляем место для рамки и отступов
}

// Init не делает ничего на старте
func (s StatusBar) Init() tea.Cmd {
	return nil
}

// Update принимает глобальные сообщения и реагирует ТОЛЬКО на те, которые касаются прогресса
func (s StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	switch msg := msg.(type) {

	// Если это сообщение для анимации кадров ползунка
	case progress.FrameMsg:
		progressModel, cmd := s.progBar.Update(msg)
		s.progBar = progressModel.(progress.Model)
		return s, cmd
	}

	return s, nil
}

// Устанавливаем процент (вызывается из родителя)
func (s *StatusBar) SetPercent(percent float64) tea.Cmd {
	s.IsActive = true
	return s.progBar.SetPercent(percent)

}

func (s StatusBar) View() string {
	if !s.IsActive {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingTop(0).
		PaddingBottom(1).
		PaddingLeft(1).
		PaddingRight(1).
		BorderForeground(lipgloss.Color("62")).
		Width(s.width)

	msgStyle := lipgloss.NewStyle().
		Height(3).
		Width(s.width - 4) // Вычитаем ширину рамок и внутренних отступов

	msgBox := msgStyle.Render(s.Message)

	// Убираем двойной \n\n, так как msgBox теперь сам держит нужную дистанцию
	content := fmt.Sprintf("%s\n%s", msgBox, s.progBar.View())

	return style.Render(content)
}
