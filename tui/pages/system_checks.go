package pages

import (
	"strings"

	"rfad-installer/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// === СТРАНИЦА 0: Сводка системных проверок ===
type SystemChecksPage struct {
	Checks tui.SystemChecks
}

func NewSystemChecksPage(checks tui.SystemChecks) SystemChecksPage {
	return SystemChecksPage{Checks: checks}
}

func (m SystemChecksPage) Init() tea.Cmd { return nil }

// Управление полностью передано в index.go
func (m SystemChecksPage) Update(msg tea.Msg) (SystemChecksPage, tea.Cmd) {
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

	// 1. Собираем результаты проверок
	var checks []string
	var errors []string
	var warnings []string

	if m.Checks.IsSudo {
		checks = append(checks, failMark+" root")
		errors = append(errors, "КРИТИЧЕСКАЯ ОШИБКА: Запуск от root запрещён. Перезапустите без sudo.")
	} else {
		checks = append(checks, okMark+" non root")
	}

	if m.Checks.HasWine {
		checks = append(checks, okMark+" PortProton")
	} else {
		checks = append(checks, failMark+" PortProton")
		errors = append(errors, "КРИТИЧЕСКАЯ ОШИБКА: Для установки необходим PortProton.")
	}

	if m.Checks.HasGameMode {
		checks = append(checks, okMark+" gamemoderun")
	} else {
		checks = append(checks, warnMark+" gamemoderun")
		warnings = append(warnings, "Предупреждение: Без gamemode производительность может быть ниже.")
	}

	if m.Checks.HasNVAPI {
		checks = append(checks, okMark+" NVAPI")
	} else {
		checks = append(checks, okMark+" NVAPI")
	}

	if m.Checks.IsSteamDeck {
		checks = append(checks, okMark+" Steam Deck "+m.Checks.DeckModel)
	} else {
		checks = append(checks, okMark+" PC Desktop")
	}

	// 2. Рендерим проверки
	var rows []string
	if len(checks) > 3 {
		rows = append(rows, strings.Join(checks[:3], "    "))
		rows = append(rows, strings.Join(checks[3:], "    "))
	} else {
		rows = append(rows, strings.Join(checks, "    "))
	}

	for _, row := range rows {
		s += lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, row) + "\n"
	}
	s += "\n"

	sep := dimStyle.Render(strings.Repeat("─", contentWidth))
	s += sep + "\n\n"

	// 3. Блок вывода ошибок/предупреждений (подсказки кнопок удалены)
	centerStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)

	if len(errors) > 0 {
		for _, err := range errors {
			s += centerStyle.Render(crossStyle.Render(err)) + "\n"
		}
		s += "\n"
	} else {
		if len(warnings) > 0 {
			for _, warn := range warnings {
				s += centerStyle.Render(warnStyle.Render(warn)) + "\n"
			}
			s += "\n"
		} else {
			s += "\n\n"
		}
	}

	return s
}
