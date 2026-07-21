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

// ДОБАВЛЕН МЕТОД ДЛЯ INDEX.GO
func (m OptionsPage) IsAtBottom() bool {
	return m.focusIndex == 6 // 6 - это индекс последнего чекбокса "Создать ярлыки"
}

func (m OptionsPage) getNextIndex(dir int) int {
	next := m.focusIndex + dir

	for {
		// Жестко ограничиваем индексы (без зацикливания), чтобы работал FocusGate
		if next < 0 {
			return 0
		} else if next > 6 {
			return 6
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

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			m.focusIndex = m.getNextIndex(-1)
		case "down":
			m.focusIndex = m.getNextIndex(1)

		// Удалили tab / shift+tab, они теперь живут только в index.go

		case "left":
			if m.focusIndex == 0 {
				switch m.Config.GraphicsMod {
				case "Без мода":
					m.Config.GraphicsMod = "Community Shaders"
				case "ENB":
					m.Config.GraphicsMod = "Без мода"
				case "ReShade":
					m.Config.GraphicsMod = "ENB"
				case "Community Shaders":
					m.Config.GraphicsMod = "ReShade"
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
			if m.focusIndex == 0 {
				switch m.Config.GraphicsMod {
				case "Без мода":
					m.Config.GraphicsMod = "ENB"
				case "ENB":
					m.Config.GraphicsMod = "ReShade"
				case "ReShade":
					m.Config.GraphicsMod = "Community Shaders"
				case "Community Shaders":
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
				switch m.Config.GraphicsMod {
				case "Без мода":
					m.Config.GraphicsMod = "ENB"
				case "ENB":
					m.Config.GraphicsMod = "ReShade"
				case "ReShade":
					m.Config.GraphicsMod = "Community Shaders"
				case "Community Shaders":
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
			}
		}
	}

	// Обработка инпутов (разрешения)
	if m.focusIndex == 3 {
		m.inputs[0].Focus()
		m.inputs[1].Blur()
		m.inputs[0].Model, cmd = m.inputs[0].Model.Update(msg)
		m.Config.ResWidth = m.inputs[0].Value() // Сохраняем на лету
	} else if m.focusIndex == 4 {
		m.inputs[0].Blur()
		m.inputs[1].Focus()
		m.inputs[1].Model, cmd = m.inputs[1].Model.Update(msg)
		m.Config.ResHeight = m.inputs[1].Value() // Сохраняем на лету
	} else {
		m.inputs[0].Blur()
		m.inputs[1].Blur()
	}

	return m, cmd
}

func (m OptionsPage) View() string {
	var s string
	boxWidth := 56

	title := lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).Render("Настройка дополнительных параметров:")
	s += title + "\n\n"

	if !m.Config.UseFSR {
		s += "\n"
	}

	cMod := "  "
	if m.focusIndex == 0 {
		cMod = "> "
	}

	mods := []string{"Без мода", "ENB", "ReShade", "Community Shaders"}
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

	s += renderCheckbox(1, "Включить upscale (FSR 3.0)", m.Config.UseFSR)

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
	}

	s += "\n"
	s += renderCheckbox(5, "Установить Steam Fix (достижения)", m.Config.UseSteamFix)
	s += renderCheckbox(6, "Создать ярлыки на рабочем столе", m.Config.CreateShortcuts)

	if !m.Config.UseFSR || m.Config.FSRLevel != 4 {
		s += "\n\n"
	}

	s += "\n" // Небольшой отступ перед кнопками в index.go

	return lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Left).
		Render(s)
}
