package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"rfad-installer/tui"
	"rfad-installer/tui/components"
)

type OptionsPage struct {
	Config     *tui.InstallConfig
	focusIndex int
	inputs     []components.Input // 0: Ширина, 1: Высота
}

func NewOptionsPage(cfg *tui.InstallConfig) OptionsPage {
	widthInput := components.NewInput("Ширина", 8)
	widthInput.Model.SetValue(cfg.ResWidth)

	heightInput := components.NewInput("Высота", 8)
	heightInput.Model.SetValue(cfg.ResHeight)

	return OptionsPage{
		Config:     cfg,
		focusIndex: 0,
		inputs:     []components.Input{widthInput, heightInput},
	}
}

func (m OptionsPage) Init() tea.Cmd { return nil }

// getNextIndex помогает прыгать через скрытые элементы (инпуты)
func (m OptionsPage) getNextIndex(dir int) int {
	next := m.focusIndex + dir

	for {
		if next < 0 {
			next = 8
		} else if next > 8 {
			next = 0
		}

		// Пропускаем меню пресетов, если FSR выключен
		if !m.Config.UseFSR && (next == 2 || next == 3 || next == 4) {
			next += dir
			continue
		}

		// Пропускаем инпуты, если выбран не "Своё" (не Level 4)
		if m.Config.UseFSR && m.Config.FSRLevel != 4 && (next == 3 || next == 4) {
			next += dir
			continue
		}
		return next
	}
}

func (m OptionsPage) Update(msg tea.Msg) (OptionsPage, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "shift+tab":
			m.focusIndex = m.getNextIndex(-1)
		case "down", "tab":
			m.focusIndex = m.getNextIndex(1)

		// Переключение пресетов стрелками Влево/Вправо
		case "left":
			if m.focusIndex == 2 {
				// Логика переключения: 3 (25) -> 2 (50) -> 1 (75) -> 4 (Своё)
				switch m.Config.FSRLevel {
				case 3:
					m.Config.FSRLevel = 4
				case 2:
					m.Config.FSRLevel = 3
				case 1:
					m.Config.FSRLevel = 2
				case 4:
					m.Config.FSRLevel = 1
				}
			}
		case "right":
			if m.focusIndex == 2 {
				switch m.Config.FSRLevel {
				case 3:
					m.Config.FSRLevel = 2
				case 2:
					m.Config.FSRLevel = 1
				case 1:
					m.Config.FSRLevel = 4
				case 4:
					m.Config.FSRLevel = 3
				}
			}

		case "enter", " ":
			switch m.focusIndex {
			case 0:
				if m.Config.GraphicsMod == "ENB" {
					m.Config.GraphicsMod = "ReShade"
				} else {
					m.Config.GraphicsMod = "ENB"
				}
			case 1:
				m.Config.UseFSR = !m.Config.UseFSR
			case 2:
				// Enter на пресетах тоже переключает их вправо
				switch m.Config.FSRLevel {
				case 3:
					m.Config.FSRLevel = 2
				case 2:
					m.Config.FSRLevel = 1
				case 1:
					m.Config.FSRLevel = 4
				case 4:
					m.Config.FSRLevel = 3
				}
			case 5:
				m.Config.UseSteamFix = !m.Config.UseSteamFix
			case 6:
				m.Config.CreateShortcuts = !m.Config.CreateShortcuts
			case 7:
				return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallPath} } // Назад
			case 8:
				m.Config.ResWidth = m.inputs[0].Value()
				m.Config.ResHeight = m.inputs[1].Value()
				return m, func() tea.Msg { return ChangePageMsg{Page: PageSummary} } // Далее
			}
		}
	}

	if m.focusIndex == 3 {
		m.inputs[0].Focus()
		m.inputs[1].Blur()
		m.inputs[0].Model, cmd = m.inputs[0].Model.Update(msg)
	} else if m.focusIndex == 4 {
		m.inputs[0].Blur()
		m.inputs[1].Focus()
		m.inputs[1].Model, cmd = m.inputs[1].Model.Update(msg)
	} else {
		m.inputs[0].Blur()
		m.inputs[1].Blur()
	}

	return m, cmd
}

func (m OptionsPage) View() string {
	var s string
	s += "Настройка дополнительных параметров:\n\n"

	// 0: Графика
	cursor := "  "
	if m.focusIndex == 0 {
		cursor = "> "
	}
	styleMod := lipgloss.NewStyle()
	if m.focusIndex == 0 {
		styleMod = styleMod.Foreground(lipgloss.Color("62")).Bold(true)
	}
	s += cursor + styleMod.Render(fmt.Sprintf("Графический мод: [ %s ]", m.Config.GraphicsMod)) + "\n\n"

	// Вспомогательная функция для чекбоксов
	renderCheckbox := func(idx int, label string, isChecked bool) string {
		c := "  "
		if m.focusIndex == idx {
			c = "> "
		}
		check := "[ ]"
		if isChecked {
			check = "[x]"
		}
		style := lipgloss.NewStyle()
		if m.focusIndex == idx {
			style = style.Foreground(lipgloss.Color("62")).Bold(true)
		}
		return c + style.Render(fmt.Sprintf("%s %s", check, label)) + "\n"
	}

	// 1: FSR Вкл/Выкл
	s += renderCheckbox(1, "Включить upscale (FSR 3.0)", m.Config.UseFSR)

	// 2: FSR Пресеты (рендерим горизонтально!)
	if m.Config.UseFSR {
		c := "  "
		if m.focusIndex == 2 {
			c = "> "
		}

		items := []string{"25%", "50%", "75%", "Своё"}
		levels := []int{3, 2, 1, 4} // Твой порядок в коде!

		var renderedItems []string
		for i, lvl := range levels {
			style := lipgloss.NewStyle().Padding(0, 1)
			if m.Config.FSRLevel == lvl {
				// Активный пресет выделяем фоном
				style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
			} else {
				style = style.Foreground(lipgloss.Color("240"))
			}
			renderedItems = append(renderedItems, style.Render(items[i]))
		}

		presetRow := strings.Join(renderedItems, " ")
		style := lipgloss.NewStyle()
		if m.focusIndex == 2 {
			style = style.Foreground(lipgloss.Color("62")).Bold(true)
		}

		s += c + style.Render("Уровень FSR: ") + presetRow + "\n"

		// 3 и 4: Инпуты для "Своё"
		if m.Config.FSRLevel == 4 {
			wView := m.inputs[0].View()
			hView := m.inputs[1].View()

			wCursor := "  "
			if m.focusIndex == 3 {
				wCursor = "> "
			}
			hCursor := "  "
			if m.focusIndex == 4 {
				hCursor = "> "
			}

			// Занимает 2 строки (пустая строка + строка с инпутами)
			s += "\n  " + wCursor + "Ширина: " + wView + "   " + hCursor + "Высота: " + hView + "\n"
		} else {
			// Резервируем ровно столько же места (2 невидимые строки), чтобы интерфейс не прыгал
			s += "\n\n"
		}
	}

	s += "\n"
	s += renderCheckbox(5, "Установить Steam Fix (достижения)", m.Config.UseSteamFix)
	s += renderCheckbox(6, "Создать ярлыки на рабочем столе", m.Config.CreateShortcuts)
	s += "\n"

	// Кнопки Назад / Далее
	btnStyle := lipgloss.NewStyle().Padding(0, 3).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Foreground(lipgloss.Color("240"))
	activeBtnStyle := btnStyle.Copy().BorderForeground(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62"))

	btnBack := btnStyle.Render("Назад")
	if m.focusIndex == 7 {
		btnBack = activeBtnStyle.Render("Назад")
	}
	btnNext := btnStyle.Render("Далее")
	if m.focusIndex == 8 {
		btnNext = activeBtnStyle.Render("Далее")
	}

	s += lipgloss.NewStyle().MarginTop(1).MarginLeft(2).Render(lipgloss.JoinHorizontal(lipgloss.Center, btnBack, "        ", btnNext))

	return s
}
