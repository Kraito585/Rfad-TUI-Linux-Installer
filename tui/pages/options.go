package pages

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"rfad-installer/tui"
	"rfad-installer/tui/components"
)

type OptionsPage struct {
	Config     *tui.InstallConfig
	focusIndex int
	inputs     []components.Input // Индексы 0: Ширина, 1: Высота
}

func NewOptionsPage(cfg *tui.InstallConfig) OptionsPage {
	widthInput := components.NewInput("Ширина (1920)", 10)
	widthInput.Model.SetValue(cfg.ResWidth)

	heightInput := components.NewInput("Высота (1080)", 10)
	heightInput.Model.SetValue(cfg.ResHeight)

	return OptionsPage{
		Config:     cfg,
		focusIndex: 0,
		inputs:     []components.Input{widthInput, heightInput},
	}
}

func (m OptionsPage) Init() tea.Cmd { return nil }

func (m OptionsPage) Update(msg tea.Msg) (OptionsPage, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "shift+tab":
			m.focusIndex--
			// Пропускаем инпуты FSR, перепрыгивая на чекбокс FSR (индекс 1)
			if !m.Config.UseFSR && (m.focusIndex == 2 || m.focusIndex == 3) {
				m.focusIndex = 1
			}
			if m.focusIndex < 0 {
				m.focusIndex = 6 // Зацикливаем на "Далее" (было 7)
			}

		case "down", "tab":
			m.focusIndex++
			// Пропускаем инпуты FSR, перепрыгивая на Ярлыки (индекс 4)
			if !m.Config.UseFSR && (m.focusIndex == 2 || m.focusIndex == 3) {
				m.focusIndex = 4
			}
			if m.focusIndex > 6 { // было 7
				m.focusIndex = 0
			}

		case "enter", " ":
			switch m.focusIndex {
			case 0:
				// Переключатель ENB / ReShade
				if m.Config.GraphicsMod == "ENB" {
					m.Config.GraphicsMod = "ReShade"
				} else {
					m.Config.GraphicsMod = "ENB"
				}
			case 1:
				m.Config.UseFSR = !m.Config.UseFSR
			// ИНДЕКС СТИМ-ФИКСА УДАЛЕН
			case 4:
				m.Config.CreateShortcuts = !m.Config.CreateShortcuts
			case 5:
				return m, func() tea.Msg { return ChangePageMsg{Page: 1} } // Назад
			case 6:
				m.Config.ResWidth = m.inputs[0].Value()
				m.Config.ResHeight = m.inputs[1].Value()
				return m, func() tea.Msg { return ChangePageMsg{Page: 3} } // Далее
			}
		}
	}

	// Управление фокусом текстовых полей
	if m.focusIndex == 2 {
		m.inputs[0].Focus()
		m.inputs[1].Blur()
		m.inputs[0].Model, cmd = m.inputs[0].Model.Update(msg)
	} else if m.focusIndex == 3 {
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

	// --- 0. ПЕРЕКЛЮЧАТЕЛЬ ENB / RESHADE ---
	cursorMod := "  "
	if m.focusIndex == 0 {
		cursorMod = "> "
	}
	styleMod := lipgloss.NewStyle()
	if m.focusIndex == 0 {
		styleMod = styleMod.Foreground(lipgloss.Color("62")).Bold(true)
	}
	s += cursorMod + styleMod.Render(fmt.Sprintf("Графический мод: [ %s ]", m.Config.GraphicsMod)) + "\n\n"

	// Функция-хелпер для обычных чекбоксов
	renderCheckbox := func(idx int, label string, isChecked bool) string {
		cursor := "  "
		if m.focusIndex == idx {
			cursor = "> "
		}
		check := "[ ]"
		if isChecked {
			check = "[x]"
		}
		style := lipgloss.NewStyle()
		if m.focusIndex == idx {
			style = style.Foreground(lipgloss.Color("62")).Bold(true)
		}
		return cursor + style.Render(fmt.Sprintf("%s %s", check, label)) + "\n"
	}

	// --- 1. FSR ---
	s += renderCheckbox(1, "Включить upscale (FSR)", m.Config.UseFSR)

	if m.Config.UseFSR {
		widthView := m.inputs[0].View()
		heightView := m.inputs[1].View()

		inputsRow := lipgloss.JoinHorizontal(lipgloss.Top, widthView, "  x  ", heightView)
		dropdownStyle := lipgloss.NewStyle().
			MarginLeft(6).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(2)

		s += dropdownStyle.Render("Базовое разрешение рендера:\n"+inputsRow) + "\n\n"
	} else {
		s += "\n"
	}

	// --- 2. STEAM FIX (ЗАКОММЕНТИРОВАН) & SHORTCUTS ---
	// s += renderCheckbox(4, "Установить Steam Fix (достижения)", m.Config.UseSteamFix)
	s += renderCheckbox(4, "Создать ярлыки на рабочем столе", m.Config.CreateShortcuts)
	s += "\n"

	// --- 3. КНОПКИ НАЗАД / ДАЛЕЕ (сдвинулись индексы) ---
	btnStyle := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Foreground(lipgloss.Color("240"))

	activeBtnStyle := btnStyle.Copy().
		BorderForeground(lipgloss.Color("62")).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62"))

	btnBack := btnStyle.Render("Назад")
	if m.focusIndex == 5 {
		btnBack = activeBtnStyle.Render("Назад")
	}

	btnNext := btnStyle.Render("Далее")
	if m.focusIndex == 6 {
		btnNext = activeBtnStyle.Render("Далее")
	}

	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Center, btnBack, "        ", btnNext)
	s += lipgloss.NewStyle().MarginTop(1).MarginLeft(2).Render(buttonsRow)

	return s
}
