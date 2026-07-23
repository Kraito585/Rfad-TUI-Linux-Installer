package pages

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"rfad-installer/core"
	"rfad-installer/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type KittyLoadedMsg struct {
	Escape string
	Rows   int
}

type TickImageMsg struct {
	ID int
}

func tickImage(id int) tea.Cmd {
	return tea.Tick(time.Second*4, func(t time.Time) tea.Msg {
		return TickImageMsg{ID: id}
	})
}

func loadKittyCmd(preset core.Preset, imgIdx int) tea.Cmd {
	return func() tea.Msg {
		imgFileName := fmt.Sprintf("%s_%d.png", preset.ID, imgIdx)
		cachedImagePath := filepath.Join("local_cache", "images", imgFileName)
		escape, _, rows := core.GetKittyEscape(cachedImagePath, 70, 15)
		return KittyLoadedMsg{Escape: escape, Rows: rows}
	}
}

func (m ShadersPage) GetKittyImage() string {
	if m.IsLoadingImg {
		return ""
	}
	return m.KittyEscape
}

type ShadersPage struct {
	Config            *tui.InstallConfig
	ActivePresetIndex int
	ActiveImageIndex  int
	ActiveTab         int // 0: Описание, 1: Тех. сводка, 2: Доп. моды
	FocusIndex        int // 0 = Пресеты, 1 = Вкладки, 2..N = Моды (если открыта вкладка 2)

	SelectedMods map[string]map[string]bool

	KittyEscape  string
	KittyRows    int
	IsLoadingImg bool
	TickID       int

	ActiveTabStyle   lipgloss.Style
	InactiveTabStyle lipgloss.Style
}

func NewShadersPage(cfg *tui.InstallConfig) ShadersPage {
	return ShadersPage{
		Config:            cfg,
		ActivePresetIndex: 0,
		ActiveImageIndex:  0,
		ActiveTab:         0,
		FocusIndex:        0,
		SelectedMods:      make(map[string]map[string]bool),
		KittyEscape:       "",
		KittyRows:         0,
		IsLoadingImg:      true,
		TickID:            1, // ИСПРАВЛЕНИЕ: Задаем 1 изначально, чтобы не увеличивать в Init()
		ActiveTabStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Border(lipgloss.NormalBorder()).Padding(0, 1),
		InactiveTabStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Border(lipgloss.NormalBorder()).Padding(0, 1),
	}
}

func (m *ShadersPage) initPresetMods(preset core.Preset) {
	if m.SelectedMods[preset.ID] == nil {
		m.SelectedMods[preset.ID] = make(map[string]bool)
		for _, mod := range preset.Metadata.OptionalMods {
			if mod.IsRequired {
				m.SelectedMods[preset.ID][mod.ID] = true
			} else {
				m.SelectedMods[preset.ID][mod.ID] = false
			}
		}
	}
	// Синхронизируем состояние с конфигом на лету
	m.Config.ShaderPresetID = preset.ID
	m.Config.ShaderMods = m.SelectedMods[preset.ID]
}

func (m *ShadersPage) canEnable(preset core.Preset, modID string) bool {
	for _, mod := range preset.Metadata.OptionalMods {
		if mod.ID == modID {
			for _, dep := range mod.DependsOn {
				if !m.SelectedMods[preset.ID][dep] {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (m *ShadersPage) canDisable(preset core.Preset, modID string) bool {
	for _, mod := range preset.Metadata.OptionalMods {
		if mod.ID == modID && mod.IsRequired {
			return false
		}
	}
	for _, otherMod := range preset.Metadata.OptionalMods {
		if m.SelectedMods[preset.ID][otherMod.ID] {
			for _, dep := range otherMod.DependsOn {
				if dep == modID {
					return false
				}
			}
		}
	}
	return true
}

func (m ShadersPage) Init() tea.Cmd {
	if len(core.GlobalPresets) > 0 {
		m.initPresetMods(core.GlobalPresets[0])
		// ИСПРАВЛЕНИЕ: Используем жестко 1 для первого тика, так как Init работает с копией!
		return tea.Batch(loadKittyCmd(core.GlobalPresets[0], 0), tickImage(1))
	}
	return nil
}

// ДОБАВЛЕН МЕТОД ДЛЯ INDEX.GO
func (m ShadersPage) IsAtBottom() bool {
	if len(core.GlobalPresets) == 0 {
		return true
	}
	preset := core.GlobalPresets[m.ActivePresetIndex]
	maxFocus := 1
	if m.ActiveTab == 2 && len(preset.Metadata.OptionalMods) > 0 {
		maxFocus = 1 + len(preset.Metadata.OptionalMods)
	}
	return m.FocusIndex == maxFocus
}

func (m ShadersPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(core.GlobalPresets) == 0 {
		return m, nil
	}

	preset := core.GlobalPresets[m.ActivePresetIndex]
	m.initPresetMods(preset) // Убедимся, что конфиг актуален

	var cmd tea.Cmd

	// Определяем максимальный индекс фокуса
	maxFocus := 1
	if m.ActiveTab == 2 && len(preset.Metadata.OptionalMods) > 0 {
		maxFocus = 1 + len(preset.Metadata.OptionalMods)
	}

	switch msg := msg.(type) {
	case KittyLoadedMsg:
		if msg.Escape != "" {
			m.IsLoadingImg = false
			m.KittyEscape = msg.Escape
			m.KittyRows = msg.Rows
		} else {
			m.IsLoadingImg = true
		}
		return m, nil

	case TickImageMsg:
		if msg.ID != m.TickID {
			return m, nil
		}
		if len(preset.Images) > 0 {
			m.ActiveImageIndex = (m.ActiveImageIndex + 1) % len(preset.Images)
			m.IsLoadingImg = true
			m.TickID++
			return m, tea.Batch(loadKittyCmd(preset, m.ActiveImageIndex), tickImage(m.TickID))
		}
		m.TickID++
		return m, tickImage(m.TickID)

	case tea.KeyMsg:
		switch msg.String() {
		// Вверх/Вниз жестко ограничены границами страницы (без зацикливания)
		case "up", "w", "ц":
			if m.FocusIndex > 0 {
				m.FocusIndex--
			}

		case "down", "s", "ы":
			if m.FocusIndex < maxFocus {
				m.FocusIndex++
			}

		case "left", "a", "ф":
			if m.FocusIndex == 0 {
				m.ActivePresetIndex = (m.ActivePresetIndex - 1 + len(core.GlobalPresets)) % len(core.GlobalPresets)
				m.ActiveImageIndex = 0
				m.IsLoadingImg = true
				m.FocusIndex = 0 // Сброс фокуса при смене пресета
				m.ActiveTab = 0  // Сброс вкладки
				m.TickID++
				preset = core.GlobalPresets[m.ActivePresetIndex]
				m.initPresetMods(preset)
				return m, tea.Batch(loadKittyCmd(preset, m.ActiveImageIndex), tickImage(m.TickID))
			} else if m.FocusIndex == 1 {
				m.ActiveTab--
				if m.ActiveTab < 0 {
					m.ActiveTab = 2
				}
			}

		case "right", "d", "в":
			if m.FocusIndex == 0 {
				m.ActivePresetIndex = (m.ActivePresetIndex + 1) % len(core.GlobalPresets)
				m.ActiveImageIndex = 0
				m.IsLoadingImg = true
				m.FocusIndex = 0 // Сброс фокуса
				m.ActiveTab = 0  // Сброс вкладки
				m.TickID++
				preset = core.GlobalPresets[m.ActivePresetIndex]
				m.initPresetMods(preset)
				return m, tea.Batch(loadKittyCmd(preset, m.ActiveImageIndex), tickImage(m.TickID))
			} else if m.FocusIndex == 1 {
				m.ActiveTab = (m.ActiveTab + 1) % 3
			}

		case "q", "й", "e", "у":
			if len(preset.Images) == 0 {
				break
			}
			if msg.String() == "e" || msg.String() == "у" {
				m.ActiveImageIndex = (m.ActiveImageIndex + 1) % len(preset.Images)
			} else {
				m.ActiveImageIndex = (m.ActiveImageIndex - 1 + len(preset.Images)) % len(preset.Images)
			}
			m.IsLoadingImg = true
			m.TickID++
			return m, tea.Batch(loadKittyCmd(preset, m.ActiveImageIndex), tickImage(m.TickID))

		case "enter", " ":
			if m.FocusIndex == 1 {
				m.ActiveTab = (m.ActiveTab + 1) % 3
			} else if m.ActiveTab == 2 && m.FocusIndex >= 2 && m.FocusIndex <= maxFocus {
				modIdx := m.FocusIndex - 2
				mod := preset.Metadata.OptionalMods[modIdx]
				currentState := m.SelectedMods[preset.ID][mod.ID]

				if currentState {
					if m.canDisable(preset, mod.ID) {
						m.SelectedMods[preset.ID][mod.ID] = false
					}
				} else {
					if m.canEnable(preset, mod.ID) {
						m.SelectedMods[preset.ID][mod.ID] = true
					}
				}
				// Синхронизируем при каждом клике
				m.Config.ShaderMods = m.SelectedMods[preset.ID]
			}
		}
	}

	// Если сменили вкладку и фокус ушел за пределы, возвращаем на вкладки
	if m.ActiveTab != 2 && m.FocusIndex > 1 {
		m.FocusIndex = 1
	}

	return m, cmd
}

func (m ShadersPage) View() string {
	if len(core.GlobalPresets) == 0 {
		return "Загрузка пресетов..."
	}

	preset := core.GlobalPresets[m.ActivePresetIndex]
	width := 70

	var techDesc, mainDesc string
	parts := strings.Split(preset.Metadata.Description, "||")
	if len(parts) >= 2 {
		techDesc = strings.TrimSpace(parts[0])
		mainDesc = strings.TrimSpace(parts[1])
	} else {
		mainDesc = strings.TrimSpace(preset.Metadata.Description)
	}

	// === БЛОК 1: Колесо пресетов ===
	prevIdx := (m.ActivePresetIndex - 1 + len(core.GlobalPresets)) % len(core.GlobalPresets)
	nextIdx := (m.ActivePresetIndex + 1) % len(core.GlobalPresets)

	wheel := fmt.Sprintf("  <  %s  |  [%s]  |  %s  >  ",
		core.GlobalPresets[prevIdx].Metadata.AuthorNickname,
		preset.Metadata.AuthorNickname,
		core.GlobalPresets[nextIdx].Metadata.AuthorNickname,
	)

	style1 := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)
	if m.FocusIndex == 0 {
		style1 = style1.Foreground(lipgloss.Color("62")).Bold(true)
	}
	block1 := style1.Render(wheel)

	// === БЛОК 2: 3 Вкладки ===
	tabs := []string{"Описание", "Тех. сводка", "Доп. моды"}
	var renderedTabs []string

	for i, t := range tabs {
		style := m.InactiveTabStyle
		if m.FocusIndex == 1 {
			if m.ActiveTab == i {
				style = m.ActiveTabStyle.Copy().BorderForeground(lipgloss.Color("62"))
			}
		} else {
			if m.ActiveTab == i {
				style = m.ActiveTabStyle
			}
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	block2 := lipgloss.JoinHorizontal(lipgloss.Center, renderedTabs[0], "  ", renderedTabs[1], "  ", renderedTabs[2])
	block2 = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).MarginTop(1).Render(block2)

	// === БЛОК 3: Контент ===
	var block3 string
	contentHeight := 9 // Высота урезана с 14 до 9 для экономии места

	if m.ActiveTab == 0 { // Описание
		block3 = lipgloss.NewStyle().Width(width).Height(contentHeight).Padding(0, 2).Render(mainDesc)
	} else if m.ActiveTab == 1 { // Тех. сводка
		techInfo := ""
		if techDesc != "" {
			techInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(techDesc) + "\n\n"
		}
		techInfo += fmt.Sprintf("Влияние на FPS: %d\nАвтор: %s", preset.PerformanceImpact, preset.Metadata.AuthorNickname)
		block3 = lipgloss.NewStyle().Width(width).Height(contentHeight).Padding(0, 2).Render(techInfo)
	} else if m.ActiveTab == 2 { // Доп. моды
		if len(preset.Metadata.OptionalMods) == 0 {
			block3 = lipgloss.NewStyle().Width(width).Height(contentHeight).Padding(0, 2).Align(lipgloss.Center).Render("\n\nДля данного пресета нет дополнительных модов.")
		} else {
			var modsRender []string
			for i, mod := range preset.Metadata.OptionalMods {
				cursor := "  "
				if m.FocusIndex == i+2 {
					cursor = "> "
				}

				isEnabled := m.SelectedMods[preset.ID][mod.ID]
				canEnable := m.canEnable(preset, mod.ID)

				checkbox := "[ ]"
				if mod.IsRequired {
					checkbox = "[*]"
				} else if isEnabled {
					checkbox = "[x]"
				}

				style := lipgloss.NewStyle()
				if m.FocusIndex == i+2 {
					style = style.Foreground(lipgloss.Color("62")).Bold(true)
				} else if !canEnable && !isEnabled {
					style = style.Foreground(lipgloss.Color("240"))
				} else if mod.IsRequired {
					style = style.Foreground(lipgloss.Color("208"))
				}

				modStr := fmt.Sprintf("%s%s %s", cursor, checkbox, mod.Name)
				if mod.IsRequired {
					modStr += " (Обяз.)"
				} else if !canEnable && !isEnabled {
					modStr += " (Нет зависимости)"
				}

				modsRender = append(modsRender, style.Render(modStr))
			}
			modsList := strings.Join(modsRender, "\n")
			block3 = lipgloss.NewStyle().Width(width).Height(contentHeight).Padding(0, 2).Render(modsList)
		}
	}

	// === БЛОК 4: Изображение ===
	imageControls := fmt.Sprintf("[ Q ]  <  Скриншот %d из %d  >  [ E ]", m.ActiveImageIndex+1, len(preset.Images))
	imgBoxWidth := 70
	imgBoxHeight := 15

	var renderImage string
	if m.IsLoadingImg {
		renderImage = lipgloss.Place(imgBoxWidth, imgBoxHeight, lipgloss.Center, lipgloss.Center, "[ Загрузка изображения... ]")
	} else {
		// Просто пустая "дыра" из пробелов
		emptyRow := strings.Repeat(" ", imgBoxWidth)
		var hole []string
		for i := 0; i < imgBoxHeight; i++ {
			hole = append(hole, emptyRow)
		}
		renderImage = strings.Join(hole, "\n")
	}

	sixelBox := lipgloss.NewStyle().Width(imgBoxWidth).Height(imgBoxHeight).Align(lipgloss.Center).Render(renderImage)
	block4 := lipgloss.JoinVertical(lipgloss.Center, sixelBox, imageControls)
	block4 = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(block4)

	return lipgloss.JoinVertical(lipgloss.Center, block1, block2, block3, block4)
}
