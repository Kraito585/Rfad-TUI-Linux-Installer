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

	s += fmt.Sprintf("  Инсталлятор: %s\n", valStyle.Render(m.Config.InstallerPath))
	s += fmt.Sprintf("  Куда ставить: %s\n", valStyle.Render(m.Config.InstallPath))
	s += fmt.Sprintf("  Графика: %s\n", valStyle.Render(m.Config.GraphicsMod))

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
	s += fmt.Sprintf("  FSR 3.0: %s\n", valStyle.Render(fsrStr))

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
	boldStyle := lipgloss.NewStyle().Bold(true)

	contentWidth := 64 // Единая ширина для идеального баланса

	headerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("62")).
		Bold(true)

	titleStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("241"))

	// Стиль для центрирования обычного текста
	promptStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center)

	// Красивые иконки в квадратных скобках [ ✓ ]
	brL := dimStyle.Render("[")
	brR := dimStyle.Render("]")
	okMark := brL + checkStyle.Render("✓") + brR
	failMark := brL + crossStyle.Render("✗") + brR
	starMark := brL + checkStyle.Render("★") + brR
	zapMark := brL + warnStyle.Render("⚡") + brR

	s := headerStyle.Render("Системные проверки") + "\n\n"

	// 1. Проверка на root (Sudo)
	s += titleStyle.Render("──── Проверка безопасности ────") + "\n"
	if m.Checks.IsSudo {
		s += promptStyle.Render(failMark+" "+boldStyle.Render("Запуск от root")) + "\n"
		s += promptStyle.Render(warnStyle.Render("КРИТИЧЕСКАЯ ОШИБКА: ")+"Запуск от root запрещён.") + "\n"
		s += promptStyle.Render("Завершите установку и перезапустите без sudo.") + "\n\n"
		s += promptStyle.Render(warnStyle.Render("Установщик НЕ МОЖЕТ быть запущен от имени root.")) + "\n"
		s += promptStyle.Render(warnStyle.Render("Нажмите Ctrl+C для выхода.")) + "\n\n"
		return s
	} else {
		s += promptStyle.Render(okMark+" Запуск от обычного пользователя") + "\n\n"
	}

	// 2. Проверка Wine/Proton
	s += titleStyle.Render("──── Проверка Proton ────") + "\n"
	if m.Checks.HasWine {
		s += promptStyle.Render(okMark+" Proton обнаружен") + "\n\n"
	} else {
		s += promptStyle.Render(failMark+" "+boldStyle.Render("Proton не найден")) + "\n"
		s += promptStyle.Render(warnStyle.Render("Proton отсутствует в системе.")) + "\n"
		s += promptStyle.Render("Установите PortProton и повторите попытку.") + "\n\n"
		s += promptStyle.Render(warnStyle.Render("Для продолжения необходим PortProton.")) + "\n"
		s += promptStyle.Render(warnStyle.Render("Нажмите Ctrl+C для выхода.")) + "\n\n"
		return s
	}

	// 3. Проверка gamemoderun
	s += titleStyle.Render("──── Проверка GameMode ────") + "\n"
	if m.Checks.HasGameMode {
		s += promptStyle.Render(okMark+" gamemoderun обнаружен") + "\n\n"
	} else {
		s += promptStyle.Render(failMark+" "+boldStyle.Render("gamemoderun не найден")) + "\n"
		s += promptStyle.Render(warnStyle.Render("Производительность может быть ниже ожидаемой.")) + "\n"
		s += promptStyle.Render("Рекомендуется установить пакет gamemode.") + "\n\n"
	}

	// 4. Проверка NVAPI
	s += titleStyle.Render("──── Проверка NVAPI ────") + "\n"
	if m.Checks.HasNVAPI {
		s += promptStyle.Render(okMark+" NVAPI поддерживается вашей видеокартой") + "\n\n"
	} else {
		s += promptStyle.Render(starMark+" NVAPI не требуется (видеокарта без поддержки или не NVIDIA)") + "\n"
		s += promptStyle.Render(dimStyle.Render("Проблем не обнаружено.")) + "\n\n"
	}

	// Разделитель
	sep := strings.Repeat("─", contentWidth)
	s += dimStyle.Render(sep) + "\n\n"

	// 1. Заголовок "Сводка" центрируем относительно всего окна
	s += lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("62")).
		Bold(true).
		Render("Сводка") + "\n"

	// 2. СВОДКА: Собираем пункты в отдельный массив (БЕЗ заголовка)
	var summaryLines []string

	if !m.Checks.IsSudo {
		summaryLines = append(summaryLines, okMark+" Безопасность: OK")
	}
	if m.Checks.HasWine {
		summaryLines = append(summaryLines, okMark+" Proton: OK")
	}
	if m.Checks.HasGameMode {
		summaryLines = append(summaryLines, okMark+" GameMode: OK")
	} else {
		summaryLines = append(summaryLines, zapMark+" GameMode: отсутствует (не критично)")
	}
	if !m.Checks.HasNVAPI {
		summaryLines = append(summaryLines, okMark+" NVAPI: проблем нет")
	} else {
		summaryLines = append(summaryLines, okMark+" NVAPI: доступен")
	}

	// 3. Склеиваем строки сводки и центрируем получившийся блок с чекбоксами
	sumBlock := lipgloss.JoinVertical(lipgloss.Left, summaryLines...)
	s += lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, sumBlock) + "\n\n"

	s += dimStyle.Render(sep) + "\n\n"

	// Кнопка продолжения
	s += promptStyle.Render("[ Enter - Продолжить установку ]")

	return s
}
