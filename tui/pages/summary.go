package pages

import (
	"fmt"

	"rfad-installer/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SummaryPage struct {
	Config *tui.InstallConfig
}

func NewSummaryPage(cfg *tui.InstallConfig) SummaryPage {
	return SummaryPage{Config: cfg}
}

func (m SummaryPage) Update(msg tea.Msg) (SummaryPage, tea.Cmd) {
	return m, nil
}

func (m SummaryPage) View() string {
	contentWidth := 70

	headerStyle := lipgloss.NewStyle().
	Width(contentWidth).
	Align(lipgloss.Center).
	Foreground(lipgloss.Color("62")).
	Bold(true)

	promptStyle := lipgloss.NewStyle().
	Width(contentWidth).
	Align(lipgloss.Center)

	s := headerStyle.Render("Шаг 4: Проверьте настройки перед установкой") + "\n\n"

	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	var settings string
	settings += fmt.Sprintf("Инсталлятор: %s\n", valStyle.Render(m.Config.InstallerPath))
	settings += fmt.Sprintf("Куда ставить: %s\n", valStyle.Render(m.Config.InstallPath))
	settings += fmt.Sprintf("Графика: %s\n", valStyle.Render(m.Config.GraphicsMod))

	resStr := fmt.Sprintf("%dx%d", m.Config.BaseWidth, m.Config.BaseHeight)
	settings += fmt.Sprintf("Разрешение экрана: %s\n", valStyle.Render(resStr))

	fsrStr := "Выкл"
	if m.Config.UseFSR {
		switch m.Config.FSRLevel {
			case 3:
				fsrStr = fmt.Sprintf("Вкл (25%% от %dx%d)", m.Config.BaseWidth, m.Config.BaseHeight)
			case 2:
				fsrStr = fmt.Sprintf("Вкл (50%% от %dx%d)", m.Config.BaseWidth, m.Config.BaseHeight)
			case 1:
				fsrStr = fmt.Sprintf("Вкл (75%% от %dx%d)", m.Config.BaseWidth, m.Config.BaseHeight)
			case 4:
				fsrStr = fmt.Sprintf("Вкл (Своё: %s x %s)", m.Config.ResWidth, m.Config.ResHeight)
		}
	}
	settings += fmt.Sprintf("FSR 3.0: %s", valStyle.Render(fsrStr))

	s += lipgloss.NewStyle().MarginLeft(10).Render(settings) + "\n"

	s += "\n" + promptStyle.Render("Всё верно?") + "\n"

	return s
}
