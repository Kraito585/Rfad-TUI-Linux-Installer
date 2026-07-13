package pages

import (
	"fmt"
	"os"
	"strings"
	"time"

	"rfad-installer/core"
	"rfad-installer/tui"
	"rfad-installer/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ChangePageMsg struct{ Page int }
type StartInstallMsg struct{}
type LogTickMsg time.Time // Сообщение для обновления логов по таймеру

const (
	PageInstallerPath = iota
	PageInstallPath
	PageOptions
	PageSummary
	PageInstalling
)

type ProgressMsg struct {
	Percent float64
	Message string
}

type DoneMsg struct{}
type ErrorMsg struct {
	Err error
}

type Index struct {
	Config     *tui.InstallConfig
	ActivePage int
	startChan  chan *tui.InstallConfig

	WindowWidth  int
	WindowHeight int

	Page1 InstallerPathPage
	Page2 InstallPathPage
	Page3 OptionsPage
	Page4 SummaryPage

	Status components.StatusBar
	Done   bool
	Err    error

	AsciiArt string
	ShowLogs bool
	LogLines string
	BoxWidth int
}

func NewIndex(startChan chan *tui.InstallConfig, ascii string) Index {
	cfg := tui.NewInstallConfig()

	// Динамически вычисляем ширину на основе ASCII-арта
	boxWidth := 70 // Минимальная ширина
	if ascii != "" {
		lines := strings.Split(ascii, "\n")
		for _, line := range lines {
			w := lipgloss.Width(line)
			if w > boxWidth {
				boxWidth = w
			}
		}
	}

	return Index{
		Config:     cfg,
		ActivePage: PageInstallerPath,
		startChan:  startChan,
		AsciiArt:   ascii,
		BoxWidth:   boxWidth, // Сохраняем вычисленную ширину

		Page1:  NewInstallerPathPage(cfg),
		Page2:  NewInstallPathPage(cfg),
		Page3:  NewOptionsPage(cfg),
		Page4:  NewSummaryPage(cfg),
		Status: components.NewStatusBar(),
	}
}

// Функция-помощник для запуска таймера обновления логов
func tickLogs() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return LogTickMsg(t)
	})
}

// Читаем последние N строк из файла логов
func tailLogs(path string, maxLines int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "Ожидание логов..."
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func (m Index) Init() tea.Cmd {
	return tea.Batch(m.Page1.Init(), m.Page2.Init(), m.Page3.Init())
}

func (m Index) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 1. Глобальные перехваты
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "i", "I", "ш", "Ш": // Добавили русскую раскладку на всякий случай
			m.ShowLogs = !m.ShowLogs
			if m.ShowLogs {
				m.LogLines = tailLogs(core.LogPath(), 10)
				return m, tickLogs() // Запускаем цикл обновления
			}
			return m, nil
		}
	}

	// 2. Системные сообщения
	switch msg := msg.(type) {
	case LogTickMsg:
		if m.ShowLogs {
			m.LogLines = tailLogs(core.LogPath(), 10)
			cmds = append(cmds, tickLogs()) // Планируем следующий тик
		}
		return m, tea.Batch(cmds...)

	case ChangePageMsg:
		m.ActivePage = msg.Page
		return m, nil

	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width
		m.WindowHeight = msg.Height
		m.Status.SetWidth(m.BoxWidth - 4)
		return m, nil

	case StartInstallMsg:
		m.ActivePage = PageInstalling
		go func() { m.startChan <- m.Config }()
		return m, nil

	case ErrorMsg:
		m.Err = msg.Err
		m.Status.Message = "Произошла критическая ошибка!"
		return m, nil

	case ProgressMsg:
		m.Status.Message = msg.Message
		cmd := m.Status.SetPercent(msg.Percent)
		if m.ShowLogs {
			m.LogLines = tailLogs(core.LogPath(), 10)
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case DoneMsg:
		m.Done = true
		m.Status.Message = "Установка полностью завершена!"
		m.Status.SetPercent(1.0)
		return m, nil
	}

	// 3. Маршрутизация нажатий
	var cmd tea.Cmd
	switch m.ActivePage {
	case PageInstallerPath:
		m.Page1, cmd = m.Page1.Update(msg)
	case PageInstallPath:
		m.Page2, cmd = m.Page2.Update(msg)
	case PageOptions:
		m.Page3, cmd = m.Page3.Update(msg)
	case PageSummary:
		m.Page4, cmd = m.Page4.Update(msg)
	case PageInstalling:
		m.Status, cmd = m.Status.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Index) View() string {
	if m.WindowWidth == 0 || m.WindowHeight == 0 {
		return "Инициализация интерфейса..."
	}

	// Используем нашу динамическую ширину
	boxWidth := m.BoxWidth

	// 1. ЗАГОЛОВОК (ASCII-арт или обычный текст)
	var header string
	if m.AsciiArt != "" {
		asciiBlock := lipgloss.NewStyle().Align(lipgloss.Left).Render(m.AsciiArt)
		header = lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, asciiBlock)
		header = lipgloss.NewStyle().MarginBottom(1).Foreground(lipgloss.Color("62")).Render(header)
	} else {
		header = lipgloss.NewStyle().
			Width(boxWidth).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("62")).
			Bold(true).
			MarginBottom(1).
			Render("=== TUI Установщик RFAD SE ===")
	}

	// 2. КОНТЕНТ ОСНОВНОГО ОКНА
	var rawBody string
	if m.Err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		rawBody = errorStyle.Render(fmt.Sprintf("ОШИБКА: %v", m.Err))
	} else {
		switch m.ActivePage {
		case PageInstallerPath:
			rawBody = m.Page1.View()
		case PageInstallPath:
			rawBody = m.Page2.View()
		case PageOptions:
			rawBody = m.Page3.View()
		case PageSummary:
			rawBody = m.Page4.View()
		case PageInstalling:
			rawBody = m.Status.View()
		}
	}

	bodyBlock := lipgloss.NewStyle().Align(lipgloss.Left).Render(rawBody)
	body := lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, bodyBlock)

	// 3. ПОДВАЛ
	footerText := "Нажмите 'i' для логов | 'ctrl+c' для выхода"
	if m.Done {
		footerText = "Установка завершена. Нажмите 'ctrl+c' для выхода."
	}
	footer := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Render(footerText)

	// 4. ДИСКЛЕЙМЕР (Разделительная черта и текст)
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")). // Очень темный серый для черты
		MarginTop(1).
		MarginBottom(1).
		Render(strings.Repeat("─", boxWidth))

	disclaimerText := "Не является Официальным продуктом: Requiem For A Dream by Immersive Chicken,\nвсе фиксы были найдены официальным Discord сообществом RFAD"
	disclaimer := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("239")). // Приглушенный серый для текста
		Render(disclaimerText)

	// Собираем главное окно
	ui := lipgloss.JoinVertical(lipgloss.Left, header, body, footer, divider, disclaimer)

	// ВЕРНУЛИ ОБЕРТКУ С РАМКОЙ
	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(ui)

	// 4. ОКНО ЛОГОВ (если включено)
	finalUI := dialogBox
	if m.ShowLogs {
		logStyle := lipgloss.NewStyle().
			Width(boxWidth+2). // Идеально выровнено по ширине с главным окном
			Height(12).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginTop(1).
			Foreground(lipgloss.Color("248"))

		logBox := logStyle.Render(m.LogLines)

		finalUI = lipgloss.JoinVertical(lipgloss.Center, dialogBox, logBox)
	}

	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		finalUI,
	)
}
