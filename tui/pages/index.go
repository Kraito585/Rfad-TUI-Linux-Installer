package pages

import (
	"fmt"
	"rfad-installer/tui"
	"rfad-installer/tui/components"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ChangePageMsg struct{ Page int }
type StartInstallMsg struct{}

const (
	PageInstallerPath = iota // 0
	PageInstallPath          // 1
	PageOptions              // 2
	PageSummary              // 3
	PageInstalling           // 4 (Экран прогресс-бара)
)

// Теперь ProgressMsg — это структура, которая динамически переносит текст и процент
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

	WindowWidth  int // Ширина окна
	WindowHeight int // Высота окна

	// Экземпляры страниц
	Page1 InstallerPathPage
	Page2 InstallPathPage
	Page3 OptionsPage
	Page4 SummaryPage

	// Состояние установки (от старого index.go)
	Status components.StatusBar
	Done   bool
	Err    error
}

func NewIndex(startChan chan *tui.InstallConfig) Index {
	cfg := tui.NewInstallConfig()

	return Index{
		Config:     cfg,
		ActivePage: PageInstallerPath, // Начинаем с первой страницы
		startChan:  startChan,

		Page1:  NewInstallerPathPage(cfg),
		Page2:  NewInstallPathPage(cfg),
		Page3:  NewOptionsPage(cfg),
		Page4:  NewSummaryPage(cfg),
		Status: components.NewStatusBar(),
	}
}

func (m Index) Init() tea.Cmd {
	return tea.Batch(m.Page1.Init(), m.Page2.Init(), m.Page3.Init())
}

func (m Index) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 1. Глобальные перехваты (Выход из программы)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// 2. Системные сообщения (Навигация и Прогресс-бар)
	switch msg := msg.(type) {
	case ChangePageMsg:
		m.ActivePage = msg.Page
		return m, nil

	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width
		m.WindowHeight = msg.Height

		// Наше главное окно будет шириной 70 символов.
		// Отдаем статус-бару 62 символа, чтобы он идеально влез внутрь рамки.
		m.Status.SetWidth(62)
		return m, nil

	case StartInstallMsg:
		m.ActivePage = PageInstalling
		// Отправляем конфиг в main.go через горутину, чтобы не заблокировать UI
		go func() { m.startChan <- m.Config }()
		return m, nil

	case ErrorMsg:
		m.Err = msg.Err
		m.Status.Message = "Произошла критическая ошибка!"
		return m, nil

	case ProgressMsg:
		m.Status.Message = msg.Message
		cmd := m.Status.SetPercent(msg.Percent)
		return m, cmd

	case DoneMsg:
		m.Done = true
		m.Status.Message = "Установка полностью завершена!"
		m.Status.SetPercent(1.0)
		return m, nil
	}

	// 3. Маршрутизация нажатий клавиш в активную страницу
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
	// Ждем, пока терминал отдаст свои размеры
	if m.WindowWidth == 0 || m.WindowHeight == 0 {
		return "Инициализация интерфейса..."
	}

	// Фиксируем ширину нашего "окна" установщика
	boxWidth := 70

	// 1. ЗАГОЛОВОК (Строго по центру рамки)
	header := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("62")).
		Bold(true).
		MarginBottom(1).
		Render("=== TUI Установщик RFAD SE 6.2 ===")

	// 2. КОНТЕНТ (По левому краю внутри рамки)
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

	// ХИТРОСТЬ ЦЕНТРИРОВАНИЯ:
	// Сначала выравниваем сам текст по левому краю (собираем его в "монолитный блок")
	bodyBlock := lipgloss.NewStyle().Align(lipgloss.Left).Render(rawBody)

	// Теперь берем этот ровный блок и помещаем его точно в центр наших 70 символов
	body := lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, bodyBlock)

	// 3. ПОДВАЛ (По центру, серый цвет)
	footerText := "Нажмите 'ctrl+c' для принудительного выхода."
	if m.Done {
		footerText = "Нажмите 'ctrl+c' или 'q' для выхода."
	}
	footer := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Render(footerText)

	// Собираем всё в единый блок
	ui := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	// МАГИЯ: Оборачиваем собранный блок в красивую рамку!
	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")). // Фиолетовая рамка
		Padding(1, 2).                          // Внутренние отступы от текста до рамки
		Render(ui)

	// Размещаем эту рамку по центру всего терминала
	return lipgloss.Place(
		m.WindowWidth, m.WindowHeight,
		lipgloss.Center, lipgloss.Center,
		dialogBox,
	)
}
