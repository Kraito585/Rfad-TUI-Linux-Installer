package pages

import (
	"fmt"
	"rfad-installer/tui"
	"rfad-installer/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// === СТРАНИЦА 1: Путь к инсталлятору ===
type InstallerPathPage struct {
	Config *tui.InstallConfig
	Input  components.Input
}

// === СТРАНИЦА 1: Путь к инсталлятору ===
func NewInstallerPathPage(cfg *tui.InstallConfig) InstallerPathPage {
	// Меняем ширину 60 на 50
	input := components.NewInput("/home/user/download/RFAD_6_2_Installer/RfaD SE 6.2.exe", 50)
	input.Focus()
	return InstallerPathPage{Config: cfg, Input: input}
}

func (m InstallerPathPage) Init() tea.Cmd { return m.Input.Init() }

func (m InstallerPathPage) Update(msg tea.Msg) (InstallerPathPage, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
		m.Config.InstallerPath = m.Input.Value()
		return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallPath} }
	}
	var cmd tea.Cmd
	m.Input.Model, cmd = m.Input.Model.Update(msg)
	return m, cmd
}

func (m InstallerPathPage) View() string {
	// Создаем стиль: ширина 52 символа (как у инпута), отступ сверху 1 строка, текст по центру
	hintStyle := lipgloss.NewStyle().
		Width(52).
		Align(lipgloss.Center).
		MarginTop(1)

	hint := hintStyle.Render("[ Нажмите Enter для продолжения ]")

	return "Шаг 1: Укажите путь к установочному файлу игры:\n\n" +
		m.Input.View() + "\n" + hint
}

// === СТРАНИЦА 2: Путь установки ===
type InstallPathPage struct {
	Config *tui.InstallConfig
	Input  components.Input
}

// === СТРАНИЦА 2: Путь установки ===
func NewInstallPathPage(cfg *tui.InstallConfig) InstallPathPage {
	// Меняем ширину 60 на 50
	input := components.NewInput("/home/user/Games/RFAD_SE", 50)
	input.Focus()
	return InstallPathPage{Config: cfg, Input: input}
}

func (m InstallPathPage) Init() tea.Cmd { return m.Input.Init() }

func (m InstallPathPage) Update(msg tea.Msg) (InstallPathPage, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "enter" {
			m.Config.InstallPath = m.Input.Value()
			return m, func() tea.Msg { return ChangePageMsg{Page: PageOptions} }
		}
		if key.String() == "esc" {
			return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallerPath} }
		}
	}
	var cmd tea.Cmd
	m.Input.Model, cmd = m.Input.Model.Update(msg)
	return m, cmd
}

func (m InstallPathPage) View() string {
	hintStyle := lipgloss.NewStyle().
		Width(52).
		Align(lipgloss.Center).
		MarginTop(1)

	hint := hintStyle.Render("[ Enter - Далее | Esc - Назад ]")

	return "Шаг 2: Укажите директорию для установки игры:\n\n" +
		m.Input.View() + "\n" + hint
}

// === СТРАНИЦА 4: Сводка ===
type SummaryPage struct {
	Config *tui.InstallConfig
}

func NewSummaryPage(cfg *tui.InstallConfig) SummaryPage {
	return SummaryPage{Config: cfg}
}

func (m SummaryPage) Update(msg tea.Msg) (SummaryPage, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "enter" {
			// ЗАПУСК УСТАНОВКИ!
			return m, func() tea.Msg { return StartInstallMsg{} }
		}
		if key.String() == "esc" {
			return m, func() tea.Msg { return ChangePageMsg{Page: PageOptions} }
		}
	}
	return m, nil
}

func (m SummaryPage) View() string {
	s := "Шаг 4: Проверьте настройки перед установкой\n\n"

	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))

	s += fmt.Sprintf("Инсталлятор: %s\n", valStyle.Render(m.Config.InstallerPath))
	s += fmt.Sprintf("Куда ставить: %s\n", valStyle.Render(m.Config.InstallPath))

	// ДОБАВЛЯЕМ ВЫВОД ГРАФИКИ
	s += fmt.Sprintf("Графика: %s\n", valStyle.Render(m.Config.GraphicsMod))

	fsrStr := "Выкл"
	if m.Config.UseFSR {
		fsrStr = fmt.Sprintf("Вкл (%s x %s)", m.Config.ResWidth, m.Config.ResHeight)
	}
	s += fmt.Sprintf("FSR 3.0: %s\n", valStyle.Render(fsrStr))

	// s += fmt.Sprintf("Steam Fix: %v\n", m.Config.UseSteamFix)

	s += "\nВсё верно?\n[ Enter - Начать установку | Esc - Назад ]"

	return s
}
