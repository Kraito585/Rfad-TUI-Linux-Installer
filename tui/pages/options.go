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
	inputs     []components.Input // 0: Кастомный масштаб (%)
}

func NewOptionsPage(cfg *tui.InstallConfig) OptionsPage {
	// Заменяем два инпута на один — для процентов FSR
	scaleInput := components.NewInput("Масштаб (%)", 4)

	// Если ResWidth пустой или там старое разрешение (типа 1920), ставим дефолтные 67%
	if cfg.ResWidth == "" || len(cfg.ResWidth) > 3 {
		scaleInput.Model.SetValue(cfg.CustomFSRScale)
	} else {
		scaleInput.Model.SetValue(cfg.ResWidth)
	}

	if cfg.GraphicsMod == "" {
		cfg.GraphicsMod = "Без мода"
	}

	// Дефолтный пресет для CS
	if cfg.ShaderPresetID == "" {
		cfg.ShaderPresetID = "Medium"
	}

	return OptionsPage{
		Config:     cfg,
		focusIndex: 0,
		inputs:     []components.Input{scaleInput},
	}
}

func (m OptionsPage) Init() tea.Cmd { return nil }

func (m OptionsPage) IsAtBottom() bool {
	return m.focusIndex == 6 // 6 - индекс последнего чекбокса
}

// Умный поиск следующего активного элемента
func (m OptionsPage) getNextIndex(dir int) int {
	next := m.focusIndex

	for {
		next += dir

		// Границы экрана
		if next < 0 {
			return 0
		} else if next > 6 {
			return 6
		}

		// Пропускаем строку пресетов CS, если выбран другой мод
		if next == 1 && m.Config.GraphicsMod != "Community Shaders" {
			continue
		}
		// Пропускаем настройки FSR, если он выключен
		if next == 3 && !m.Config.UseFSR {
			continue
		}
		// Пропускаем кастомный инпут FSR, если не выбран режим "Своё"
		if next == 4 && (!m.Config.UseFSR || m.Config.FSRLevel != 4) {
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

		case "left":
			switch m.focusIndex {
			case 0:
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
			case 1: // Пресеты CS
				presets := []string{"Low", "Medium", "High", "Ultra"}
				curr := 0
				for i, p := range presets {
					if p == m.Config.ShaderPresetID {
						curr = i
					}
				}
				curr--
				if curr < 0 {
					curr = len(presets) - 1
				}
				m.Config.ShaderPresetID = presets[curr]
			case 3: // Уровни FSR
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
			case 1: // Пресеты CS
				presets := []string{"Low", "Medium", "High", "Ultra"}
				curr := 0
				for i, p := range presets {
					if p == m.Config.ShaderPresetID {
						curr = i
					}
				}
				curr++
				if curr >= len(presets) {
					curr = 0
				}
				m.Config.ShaderPresetID = presets[curr]
			case 3: // Уровни FSR
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
				// Переключение мода по Enter (аналогично стрелке вправо)
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
			case 2: // Вкл/Выкл FSR
				m.Config.UseFSR = !m.Config.UseFSR
			case 5:
				m.Config.UseSteamFix = !m.Config.UseSteamFix
			case 6:
				m.Config.CreateShortcuts = !m.Config.CreateShortcuts
			}
		}
	}

	// Обработка ввода для кастомного скейла FSR (%)
	if m.focusIndex == 4 {
		m.inputs[0].Focus()
		m.inputs[0].Model, cmd = m.inputs[0].Model.Update(msg)
		m.Config.CustomFSRScale = m.inputs[0].Value()
		m.Config.ResWidth = m.inputs[0].Value()
	} else {
		m.inputs[0].Blur()
	}

	return m, cmd
}

func (m OptionsPage) View() string {
	var s string
	// Увеличили ширину, чтобы влезало длинное название "Community Shaders"
	boxWidth := 65

	title := lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).Render("Настройка дополнительных параметров:")
	s += title + "\n\n"

	// ================================
	// 1. Графический мод
	// ================================
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

	s += cMod + styleMod.Render("Графика:    ") + modRow + "\n"

	// ================================
	// 2. Пресеты Community Shaders
	// ================================
	if m.Config.GraphicsMod == "Community Shaders" {
		cPreset := "  "
		if m.focusIndex == 1 {
			cPreset = "> "
		}

		presets := []string{"Low", "Medium", "High", "Ultra"}
		var renderedPresets []string

		for _, p := range presets {
			style := lipgloss.NewStyle().Padding(0, 1)
			if m.Config.ShaderPresetID == p {
				style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
			} else {
				style = style.Foreground(lipgloss.Color("240"))
			}
			renderedPresets = append(renderedPresets, style.Render(p))
		}

		presetRow := strings.Join(renderedPresets, " ")
		stylePreset := lipgloss.NewStyle()
		if m.focusIndex == 1 {
			stylePreset = stylePreset.Foreground(lipgloss.Color("62")).Bold(true)
		}

		s += "\n" + cPreset + stylePreset.Render("Пресет CS:  ") + presetRow + "\n"
	} else {
		s += "\n" // Пустая строка для сохранения баланса интерфейса
	}
	s += "\n"

	// Хелпер для чекбоксов
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

	// ================================
	// 3. Настройки FSR
	// ================================
	s += renderCheckbox(2, "Включить upscale (FSR 3.0)", m.Config.UseFSR)

	if m.Config.UseFSR {
		cFsr := "  "
		if m.focusIndex == 3 {
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

		fsrRow := strings.Join(renderedItems, " ")
		styleFsr := lipgloss.NewStyle()
		if m.focusIndex == 3 {
			styleFsr = styleFsr.Foreground(lipgloss.Color("62")).Bold(true)
		}

		s += cFsr + styleFsr.Render("Масштаб:    ") + fsrRow + "\n"

		// Инпут для кастомного процента
		if m.Config.FSRLevel == 4 {
			scaleView := m.inputs[0].View()
			scaleCursor := "  "
			if m.focusIndex == 4 {
				scaleCursor = "> "
			}

			s += "\n  " + scaleCursor + "Кастомный масштаб: " + scaleView + " %\n"
		} else {
			s += "\n\n"
		}
	} else {
		s += "\n\n\n"
	}

	// ================================
	// 4. Глобальные опции
	// ================================
	s += renderCheckbox(5, "Установить Steam Fix (достижения)", m.Config.UseSteamFix)
	s += renderCheckbox(6, "Создать ярлыки на рабочем столе", m.Config.CreateShortcuts)

	return lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Left).
		Render(s)
}
