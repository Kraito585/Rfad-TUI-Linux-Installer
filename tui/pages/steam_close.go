package pages

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// === Сообщение от main.go ===
type PromptSteamCloseMsg struct {
	ReplyChan chan bool
}

// === СТРАНИЦА: Подтверждение закрытия Steam ===
type SteamClosePage struct {
	ReplyChan chan bool
}

func NewSteamClosePage() SteamClosePage {
	return SteamClosePage{}
}

func (m SteamClosePage) Init() tea.Cmd { return nil }

// Update теперь практически пустой, так как кнопки навигации глобальные
func (m SteamClosePage) Update(msg tea.Msg) (SteamClosePage, tea.Cmd) {
	return m, nil
}

func (m SteamClosePage) View() string {
	warning := "\n  ВНИМАНИЕ!\n\n" +
		"  Для интеграции игры в библиотеку необходимо закрыть клиент Steam.\n" +
		"  Убедитесь, что у вас не скачиваются другие игры.\n\n" +
		"  Закрыть Steam и продолжить интеграцию прямо сейчас?"

	return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render(warning)
}
