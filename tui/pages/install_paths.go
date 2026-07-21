package pages

import (
	"rfad-installer/tui"
	"rfad-installer/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InstallPathsPage struct {
	Config     *tui.InstallConfig
	FocusIndex int // 0: Инсталлятор, 1: Путь установки, 2: Назад, 3: Далее
	Inputs     []components.Input
}

func NewInstallPathsPage(cfg *tui.InstallConfig) InstallPathsPage {
	// Поле 1: Путь к инсталлятору
	in1 := components.NewInput("/home/user/downloads/RFAD_6_2_Installer/RfaD SE 6.2.exe", 58)
	if cfg.InstallerPath != "" {
		in1.Model.SetValue(cfg.InstallerPath)
	}

	// Поле 2: Путь установки
	in2 := components.NewInput("/home/user/Games/RFAD_SE", 58)
	if cfg.InstallPath != "" {
		in2.Model.SetValue(cfg.InstallPath)
	}

	in1.Focus()

	return InstallPathsPage{
		Config:     cfg,
		FocusIndex: 0,
		Inputs:     []components.Input{in1, in2},
	}
}

func (m InstallPathsPage) Init() tea.Cmd {
	return tea.Batch(m.Inputs[0].Init(), m.Inputs[1].Init())
}

func (m InstallPathsPage) IsAtBottom() bool {
	// Возвращаем true, если фокус находится на последнем (втором) поле ввода
	return m.FocusIndex == 1
}

func (m InstallPathsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up":
			if m.FocusIndex > 0 {
				m.FocusIndex--
			}
		case "down":
			if m.FocusIndex < 1 {
				m.FocusIndex++
			}
		case "enter":
			// По Enter просто прыгаем на следующее поле ввода (для удобства)
			if m.FocusIndex == 0 {
				m.FocusIndex = 1
			}
		}
	}

	// Обновление состояния инпутов (курсор и ввод текста)
	if m.FocusIndex == 0 {
		m.Inputs[0].Focus()
		m.Inputs[1].Blur()
		m.Inputs[0].Model, cmd = m.Inputs[0].Model.Update(msg)
	} else if m.FocusIndex == 1 {
		m.Inputs[0].Blur()
		m.Inputs[1].Focus()
		m.Inputs[1].Model, cmd = m.Inputs[1].Model.Update(msg)
	}

	return m, cmd
}

func (m InstallPathsPage) View() string {
	boxWidth := 66

	// === БЛОК ПОДСКАЗОК ===
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	dangerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	hint := warnStyle.Render("ВНИМАНИЕ:") + textStyle.Render(" Путь должен указывать строго на файл\n инсталлятора ") +
		warnStyle.Render("(RfaD SE 6.2.exe)") + textStyle.Render(".\n ") +
		dangerStyle.Render("Кавычки (\"\") в пути не допускаются!")

	hintBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(boxWidth).
		Render(hint)

	// === ПОЛЯ ВВОДА ===
	cursor1 := "  "
	if m.FocusIndex == 0 {
		cursor1 = "> "
	}
	input1View := cursor1 + lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render("Путь к инсталлятору:") + "\n" + "  " + m.Inputs[0].View()

	cursor2 := "  "
	if m.FocusIndex == 1 {
		cursor2 = "> "
	}
	input2View := cursor2 + lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render("Путь для установки игры:") + "\n" + "  " + m.Inputs[1].View()

	inputsBox := lipgloss.NewStyle().Width(boxWidth).Render(input1View + "\n\n" + input2View)

	// Сборка всей страницы
	return lipgloss.JoinVertical(lipgloss.Left, hintBox, "\n", inputsBox)
}
