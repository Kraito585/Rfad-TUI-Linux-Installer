package components

import (
	"github.com/charmbracelet/lipgloss"
)

type Button struct {
	Label     string
	IsFocused bool
}

func NewButton(label string) Button {
	return Button{
		Label:     label,
		IsFocused: false,
	}
}

// Update здесь не нужен, так как кнопка не обрабатывает ввод текста,
// фокусом будет управлять родительская модель (Index).

func (b Button) View() string {
	// Базовый стиль рамки
	style := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder())

	if b.IsFocused {
		// Состояние "В фокусе": яркие цвета, инверсия
		style = style.
			BorderForeground(lipgloss.Color("62")).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("62"))
	} else {
		// Состояние "Не в фокусе": тусклые цвета
		style = style.
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("240"))
	}

	return style.Render(b.Label)
}
