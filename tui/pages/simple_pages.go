package pages

import (
	"fmt"
	"strings"

	"rfad-installer/tui"
	"rfad-installer/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// === Пауза в установке ===
type PromptSteamCloseMsg struct {
	ReplyChan chan bool
}

// === СТРАНИЦА 1: Путь к инсталлятору ===
type InstallerPathPage struct {
	Config *tui.InstallConfig
	Input  components.Input
}

// === СТРАНИЦА 1: Путь к инсталлятору ===
func NewInstallerPathPage(cfg *tui.InstallConfig) InstallerPathPage {
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
	contentWidth := 64

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

	// Собираем пункты настроек в отдельный блок (без пробелов в начале)
	var settings string
	settings += fmt.Sprintf("Инсталлятор: %s\n", valStyle.Render(m.Config.InstallerPath))
	settings += fmt.Sprintf("Куда ставить: %s\n", valStyle.Render(m.Config.InstallPath))
	settings += fmt.Sprintf("Графика: %s\n", valStyle.Render(m.Config.GraphicsMod))

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
	settings += fmt.Sprintf("FSR 3.0: %s", valStyle.Render(fsrStr)) // Убрали \n в конце

	// Сдвигаем весь блок настроек на 8 символов вправо (2 изначальных + 6 новых)
	s += lipgloss.NewStyle().MarginLeft(10).Render(settings) + "\n"

	s += "\n" + promptStyle.Render("Всё верно?") + "\n"
	s += promptStyle.Render("[ Enter - Начать установку | Esc - Назад ]")

	return s
}

// === СТРАНИЦА 0: Сводка системных проверок ===
type SystemChecksPage struct {
	Checks tui.SystemChecks
}

func NewSystemChecksPage(checks tui.SystemChecks) SystemChecksPage {
	return SystemChecksPage{Checks: checks}
}

func (m SystemChecksPage) Init() tea.Cmd { return nil }

func (m SystemChecksPage) Update(msg tea.Msg) (SystemChecksPage, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "enter" {
			return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallerPath} }
		}
		if key.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SystemChecksPage) View() string {
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // зелёный
	crossStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // красный
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // оранжевый
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	contentWidth := 64 // Жесткая ширина для центрирования

	headerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("62")).
		Bold(true)

	// Иконки
	brL := dimStyle.Render("[")
	brR := dimStyle.Render("]")
	okMark := brL + checkStyle.Render("✓") + brR
	failMark := brL + crossStyle.Render("✗") + brR
	warnMark := brL + warnStyle.Render("!") + brR

	s := headerStyle.Render("Системные проверки") + "\n\n"

	// 1. Собираем результаты проверок (только суть)
	var checks []string
	var errors []string
	var warnings []string

	// Проверка Sudo
	if m.Checks.IsSudo {
		checks = append(checks, failMark+" root")
		errors = append(errors, "КРИТИЧЕСКАЯ ОШИБКА: Запуск от root запрещён. Перезапустите без sudo.")
	} else {
		checks = append(checks, okMark+" non root")
	}

	// Проверка Proton
	if m.Checks.HasWine {
		checks = append(checks, okMark+" PortProton")
	} else {
		checks = append(checks, failMark+" PortProton")
		errors = append(errors, "КРИТИЧЕСКАЯ ОШИБКА: Для установки необходим PortProton.")
	}

	// Проверка GameMode
	if m.Checks.HasGameMode {
		checks = append(checks, okMark+" gamemoderun")
	} else {
		checks = append(checks, warnMark+" gamemoderun")
		warnings = append(warnings, "Предупреждение: Без gamemode производительность может быть ниже.")
	}

	// Проверка NVAPI
	if m.Checks.HasNVAPI {
		checks = append(checks, okMark+" NVAPI")
	} else {
		// Оставляем зеленую галочку, так как отсутствие NVAPI не является проблемой
		checks = append(checks, okMark+" NVAPI")
	}

	// 2. Рендерим проверки в ОДНУ СТРОКУ
	// Склеиваем элементы массива четырьмя пробелами
	checkRow := strings.Join(checks, "    ")

	// Выводим полученную строку строго по центру
	s += lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, checkRow) + "\n\n"

	// Разделитель
	sep := dimStyle.Render(strings.Repeat("─", contentWidth))
	s += sep + "\n\n"

	// 3. Блок вывода последствий (ошибки/предупреждения) и управления
	centerStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)

	if len(errors) > 0 {
		for _, err := range errors {
			s += centerStyle.Render(crossStyle.Render(err)) + "\n"
		}
		s += "\n"
		s += centerStyle.Render(dimStyle.Render("[ Нажмите Ctrl+C для выхода ]"))
	} else {
		if len(warnings) > 0 {
			for _, warn := range warnings {
				s += centerStyle.Render(warnStyle.Render(warn)) + "\n"
			}
			s += "\n"
		} else {
			s += "\n\n" // Резервируем место под предупреждения
		}
		s += centerStyle.Render("[ Enter - Продолжить установку ]")
	}

	return s
}
