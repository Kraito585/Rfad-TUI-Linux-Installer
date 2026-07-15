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

	// Если в конфиге было пусто или старое значение, ставим "Без мода" по умолчанию
	if cfg.GraphicsMod == "" {
		cfg.GraphicsMod = "Без мода"
	}

	return OptionsPage{
		Config:     cfg,
		focusIndex: 0,
		inputs:     []components.Input{widthInput, heightInput},
	}
}

func (m OptionsPage) Init() tea.Cmd { return nil }

func (m OptionsPage) getNextIndex(dir int) int {
	next := m.focusIndex + dir

	for {
		if next < 0 {
			next = 8
		} else if next > 8 {
			next = 0
		}

		if !m.Config.UseFSR && (next == 2 || next == 3 || next == 4) {
			next += dir
			continue
		}

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

		case "left":
			// Логика переключения графики влево
			if m.focusIndex == 0 {
				switch m.Config.GraphicsMod {
				case "Без мода":
					m.Config.GraphicsMod = "ReShade"
				case "ENB":
					m.Config.GraphicsMod = "Без мода"
				case "ReShade":
					m.Config.GraphicsMod = "ENB"
				}
			} else if m.focusIndex == 2 {
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
			// Логика переключения графики вправо
			if m.focusIndex == 0 {
				switch m.Config.GraphicsMod {
				case "Без мода":
					m.Config.GraphicsMod = "ENB"
				case "ENB":
					m.Config.GraphicsMod = "ReShade"
				case "ReShade":
					m.Config.GraphicsMod = "Без мода"
				}
			} else if m.focusIndex == 2 {
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
				// Enter тоже циклично переключает графику
				switch m.Config.GraphicsMod {
				case "Без мода":
					m.Config.GraphicsMod = "ENB"
				case "ENB":
					m.Config.GraphicsMod = "ReShade"
				case "ReShade":
					m.Config.GraphicsMod = "Без мода"
				}
			case 1:
				m.Config.UseFSR = !m.Config.UseFSR
			case 2:
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
				return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallPath} }
			case 8:
				m.Config.ResWidth = m.inputs[0].Value()
				m.Config.ResHeight = m.inputs[1].Value()
				return m, func() tea.Msg { return ChangePageMsg{Page: PageSummary} }
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

	title := lipgloss.NewStyle().Width(46).Align(lipgloss.Center).Render("Настройка дополнительных параметров:")
	s += title + "\n\n"

	// --- ДИНАМИЧЕСКИЙ ОТСТУП СВЕРХУ ---
	// Если FSR выключен, добавляем 1 строку сверху, чтобы интерфейс "дышал"
	if !m.Config.UseFSR {
		s += "\n"
	}

	// === 0: ГРАФИЧЕСКИЙ МОД ===
	cMod := "  "
	if m.focusIndex == 0 {
		cMod = "> "
	}

	mods := []string{"Без мода", "ENB", "ReShade"}
	var renderedMods []string

	for _, mod := range mods {
		style := lipgloss.NewStyle().Padding(0, 1)
		if m.Config.GraphicsMod == mod {
			style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
		} else {
			style = style.Foreground(lipgloss.Color("240"))
		}
		renderedMods = append(renderedMods, style.Render(mod))
	}

	modRow := strings.Join(renderedMods, " ")
	styleMod := lipgloss.NewStyle()
	if m.focusIndex == 0 {
		styleMod = styleMod.Foreground(lipgloss.Color("62")).Bold(true)
	}

	s += cMod + styleMod.Render("Графический мод: ") + modRow + "\n\n"

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

	// === 1: FSR Вкл/Выкл ===
	s += renderCheckbox(1, "Включить upscale (FSR 3.0)", m.Config.UseFSR)

	// === 2: FSR Пресеты ===
	if m.Config.UseFSR {
		cFsr := "  "
		if m.focusIndex == 2 {
			cFsr = "> "
		}

		items := []string{"25%", "50%", "75%", "Своё"}
		levels := []int{3, 2, 1, 4}

		var renderedItems []string
		for i, lvl := range levels {
			style := lipgloss.NewStyle().Padding(0, 1)
			if m.Config.FSRLevel == lvl {
				style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
			} else {
				style = style.Foreground(lipgloss.Color("240"))
			}
			renderedItems = append(renderedItems, style.Render(items[i]))
		}

		presetRow := strings.Join(renderedItems, " ")
		styleFsr := lipgloss.NewStyle()
		if m.focusIndex == 2 {
			styleFsr = styleFsr.Foreground(lipgloss.Color("62")).Bold(true)
		}

		s += cFsr + styleFsr.Render("Уровень FSR:     ") + presetRow + "\n"

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

			s += "\n  " + wCursor + "Ширина: " + wView + "   " + hCursor + "Высота: " + hView + "\n"
		}
		// Убрали else с \n\n
	}
	// Убрали else с \n\n\n

	s += "\n"
	s += renderCheckbox(5, "Установить Steam Fix (достижения)", m.Config.UseSteamFix)
	s += renderCheckbox(6, "Создать ярлыки на рабочем столе", m.Config.CreateShortcuts)

	// --- ДИНАМИЧЕСКИЙ ОТСТУП СНИЗУ ---
	// Если мы не показываем инпуты ширины/высоты, нам нужно компенсировать 2 строки перед кнопками
	if !m.Config.UseFSR || m.Config.FSRLevel != 4 {
		s += "\n\n"
	}

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

	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Center, btnBack, "        ", btnNext)

	// ФИНАЛЬНАЯ СБОРКА БЕЗ СМЕЩЕНИЙ
	textBox := lipgloss.NewStyle().
		Width(46).
		Align(lipgloss.Left).
		Render(s)

	buttonBox := lipgloss.NewStyle().
		Width(46).
		Align(lipgloss.Center).
		Render(buttonsRow)

	return textBox + "\n" + buttonBox
}
