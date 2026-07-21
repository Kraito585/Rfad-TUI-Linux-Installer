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
type LogTickMsg time.Time

const (
	PageSystemChecks = iota
	PageInstallerPath
	PageInstallPath
	PageOptions
	PageShaders
	PageSummary
	PageInstalling
	PageAskingSteamClose
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
	Config         *tui.InstallConfig
	SystemChecks   tui.SystemChecks
	ActivePage     int
	NavFocus       int
	startChan      chan *tui.InstallConfig
	steamReplyChan chan bool

	WindowWidth  int
	WindowHeight int

	Page0          SystemChecksPage
	Page1          InstallPathsPage
	Page3          OptionsPage
	PageShaders    ShadersPage
	Page4          SummaryPage
	PageSteamClose SteamClosePage

	Status components.StatusBar
	Done   bool
	Err    error

	AsciiArt string
	ShowLogs bool
	LogLines string
	BoxWidth int
}

func NewIndex(startChan chan *tui.InstallConfig, ascii string, isSudo, hasWine, hasGameMode, hasNVAPI bool, screenWidth, screenHeight int) Index {
	cfg := tui.NewInstallConfig()

	cfg.BaseWidth = screenWidth
	cfg.BaseHeight = screenHeight
	cfg.ResWidth = fmt.Sprintf("%d", screenWidth)
	cfg.ResHeight = fmt.Sprintf("%d", screenHeight)

	boxWidth := 70

	if ascii != "" {
		lines := strings.Split(ascii, "\n")
		for _, line := range lines {
			w := lipgloss.Width(line)
			if w > boxWidth {
				boxWidth = w
			}
		}
	}

	checks := tui.SystemChecks{
		IsSudo:       isSudo,
		HasWine:      hasWine,
		HasGameMode:  hasGameMode,
		HasNVAPI:     hasNVAPI,
		ScreenWidth:  screenWidth,
		ScreenHeight: screenHeight,
	}

	return Index{
		Config:       cfg,
		SystemChecks: checks,
		ActivePage:   PageSystemChecks,
		startChan:    startChan,
		AsciiArt:     ascii,
		BoxWidth:     boxWidth,

		Page0:          NewSystemChecksPage(checks),
		Page1:          NewInstallPathsPage(cfg),
		Page3:          NewOptionsPage(cfg),
		PageShaders:    NewShadersPage(cfg),
		Page4:          NewSummaryPage(cfg),
		PageSteamClose: NewSteamClosePage(),
		Status:         components.NewStatusBar(),
	}
}

func tickLogs() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return LogTickMsg(t)
	})
}

func tailLogs(path string, maxLines int) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return " Чтение логов..."
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func (m Index) Init() tea.Cmd {
	return tea.Batch(m.Page0.Init(), m.Page1.Init(), m.Page3.Init(), m.PageShaders.Init())
}

// === ЛОГИКА ОТКЛЮЧЕНИЯ ASCII АРТА ===
// Отключаем шапку для тяжелых страниц, чтобы уместить TUI в 40 строк (Steam Deck)
func (m Index) showAscii() bool {
	switch m.ActivePage {
	case PageShaders:
		return false
	default:
		return true
	}
}

func (m Index) handleGlobalButtons() (Index, tea.Cmd) {
	if m.NavFocus == 1 {
		switch m.ActivePage {
		case PageSystemChecks:
			return m, tea.Quit
		case PageInstallerPath:
			return m, func() tea.Msg { return ChangePageMsg{Page: PageSystemChecks} }
		case PageOptions:
			return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallerPath} }
		case PageShaders:
			return m, func() tea.Msg { return ChangePageMsg{Page: PageOptions} }
		case PageSummary:
			if m.Config.GraphicsMod == "Community Shaders" {
				return m, func() tea.Msg { return ChangePageMsg{Page: PageShaders} }
			}
			return m, func() tea.Msg { return ChangePageMsg{Page: PageOptions} }
		case PageAskingSteamClose:
			if m.PageSteamClose.ReplyChan != nil {
				m.PageSteamClose.ReplyChan <- false
				m.PageSteamClose.ReplyChan = nil
			}
			return m, func() tea.Msg { return ChangePageMsg{Page: PageInstalling} }
		}
	} else if m.NavFocus == 2 {
		switch m.ActivePage {
		case PageSystemChecks:
			if m.SystemChecks.IsSudo || !m.SystemChecks.HasWine {
				return m, nil
			}
			return m, func() tea.Msg { return ChangePageMsg{Page: PageInstallerPath} }
		case PageInstallerPath:
			m.Config.InstallerPath = m.Page1.Inputs[0].Value()
			m.Config.InstallPath = m.Page1.Inputs[1].Value()
			return m, func() tea.Msg { return ChangePageMsg{Page: PageOptions} }
		case PageOptions:
			if m.Config.GraphicsMod == "Community Shaders" {
				return m, func() tea.Msg { return ChangePageMsg{Page: PageShaders} }
			}
			return m, func() tea.Msg { return ChangePageMsg{Page: PageSummary} }
		case PageShaders:
			return m, func() tea.Msg { return ChangePageMsg{Page: PageSummary} }
		case PageSummary:
			return m, func() tea.Msg { return StartInstallMsg{} }
		case PageAskingSteamClose:
			if m.PageSteamClose.ReplyChan != nil {
				m.PageSteamClose.ReplyChan <- true
				m.PageSteamClose.ReplyChan = nil
			}
			return m, func() tea.Msg { return ChangePageMsg{Page: PageInstalling} }
		}
	}
	return m, nil
}

func (m Index) IsPageAtBottom() bool {
	switch m.ActivePage {
	case PageInstallerPath:
		return m.Page1.IsAtBottom()
	case PageOptions:
		return m.Page3.IsAtBottom()
	case PageShaders:
		return m.PageShaders.IsAtBottom()
	case PageSystemChecks, PageSummary, PageAskingSteamClose:
		// На простых экранах мы всегда "на дне"
		return true
	}
	return true
}

func (m Index) GetNextButtonText() string {
	if m.ActivePage == PageSummary {
		return "Установить"
	} else if m.ActivePage == PageAskingSteamClose {
		return "Перезапустить"
	}
	return "Далее"
}

func (m Index) GetBackButtonText() string {
	if m.ActivePage == PageAskingSteamClose {
		return "Пропустить"
	} else if m.ActivePage == PageSystemChecks {
		return "Выход"
	}
	return "Назад"
}

func (m Index) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 1. Глобальный перехват клавиш (FocusGate)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "ctrl+с":
			return m, tea.Quit
		case "ctrl+l", "ctrl+д":
			m.ShowLogs = !m.ShowLogs
			if m.ShowLogs {
				m.LogLines = tailLogs(core.LogPath(), 35)
				return m, tickLogs()
			}
			return m, nil

		// Навигация "Вниз"
		case "down", "tab":
			if m.NavFocus == 0 {
				if m.IsPageAtBottom() {
					m.NavFocus = 2 // Прыгаем на кнопку "Далее"
					return m, nil
				}
			}
		// Навигация "Вверх"
		case "up", "shift+tab":
			if m.NavFocus > 0 {
				m.NavFocus = 0 // Возвращаемся в контент страницы
				return m, nil
			}

		// Переключение между самими кнопками Назад <-> Далее
		case "left":
			if m.NavFocus == 2 {
				m.NavFocus = 1
				return m, nil
			}
		case "right":
			if m.NavFocus == 1 {
				m.NavFocus = 2
				return m, nil
			}

		// Нажатие на кнопку
		case "enter":
			if m.NavFocus > 0 {
				var cmd tea.Cmd
				m, cmd = m.handleGlobalButtons()
				return m, cmd
			}
		}
	}

	// 2. Обработка системных сообщений (без изменений)
	switch msg := msg.(type) {
	case LogTickMsg:
		if m.ShowLogs {
			m.LogLines = tailLogs(core.LogPath(), 35)
			cmds = append(cmds, tickLogs())
		}
		return m, tea.Batch(cmds...)
	case ChangePageMsg:
		m.ActivePage = msg.Page
		m.NavFocus = 0 // Сбрасываем фокус в контент при смене страницы
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
	case PromptSteamCloseMsg:
		m.PageSteamClose.ReplyChan = msg.ReplyChan
		m.ActivePage = PageAskingSteamClose
		return m, nil
	case ErrorMsg:
		m.Err = msg.Err
		m.Status.Message = " Произошла ошибка!"
		return m, nil
	case ProgressMsg:
		m.Status.Message = msg.Message
		cmd := m.Status.SetPercent(msg.Percent)
		if m.ShowLogs {
			m.LogLines = tailLogs(core.LogPath(), 35)
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case DoneMsg:
		m.Done = true
		m.Status.Message = " Установка успешно завершена!"
		m.Status.SetPercent(1.0)
		return m, tea.Batch(cmds...)
	}

	// 3. Передача управления странице (ТОЛЬКО если фокус в контенте)
	if m.NavFocus == 0 {
		var cmd tea.Cmd
		switch m.ActivePage {
		case PageSystemChecks:
			m.Page0, cmd = m.Page0.Update(msg)
		case PageInstallerPath: // Это наша InstallPathsPage
			var model tea.Model
			model, cmd = m.Page1.Update(msg)
			m.Page1 = model.(InstallPathsPage)
		case PageOptions:
			m.Page3, cmd = m.Page3.Update(msg)
		case PageShaders:
			var model tea.Model
			model, cmd = m.PageShaders.Update(msg)
			m.PageShaders = model.(ShadersPage)
		case PageSummary:
			m.Page4, cmd = m.Page4.Update(msg)
		case PageAskingSteamClose:
			m.PageSteamClose, cmd = m.PageSteamClose.Update(msg)
		case PageInstalling:
			m.Status, cmd = m.Status.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Index) View() string {
	if m.WindowWidth == 0 || m.WindowHeight == 0 {
		return " Инициализация терминала..."
	}

	boxWidth := m.BoxWidth

	// ==========================================
	// ШАПКА (ASCII-арт)
	// ==========================================
	var header string
	if m.showAscii() {
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
				Render("=== TUI УСТАНОВЩИК RFAD SE ===")
		}
	}

	// ==========================================
	// КОНТЕНТ СТРАНИЦЫ
	// ==========================================
	var rawBody string
	if m.Err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		rawBody = errorStyle.Render(fmt.Sprintf(" ОШИБКА: %v", m.Err))
	} else {
		switch m.ActivePage {
		case PageSystemChecks:
			rawBody = m.Page0.View()
		case PageInstallerPath:
			rawBody = m.Page1.View()
		case PageOptions:
			rawBody = m.Page3.View()
		case PageShaders:
			rawBody = m.PageShaders.View()
		case PageSummary:
			rawBody = m.Page4.View()
		case PageInstalling:
			rawBody = m.Status.View()
		case PageAskingSteamClose:
			warning := "\n  ВНИМАНИЕ!\n\n" +
				"  Для интеграции игры в библиотеку необходимо закрыть клиент Steam.\n" +
				"  Убедитесь, что у вас не скачиваются другие игры.\n\n" +
				"  Продолжить и закрыть Steam прямо сейчас?\n\n" +
				"  [Y / Enter] Да, закрыть Steam и продолжить\n" +
				"  [N / Esc]   Нет, пропустить интеграцию"
			rawBody = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render(warning)
		}
	}

	// Подстраиваем высоту: даем больше места шейдерам, если нужно
	bodyHeight := 16
	if m.ActivePage == PageShaders {
		bodyHeight = 35 // Можно сделать меньше, так как ASCII отключен
	}
	bodyBlock := lipgloss.PlaceVertical(bodyHeight, lipgloss.Center, lipgloss.NewStyle().Align(lipgloss.Left).Render(rawBody))
	body := lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, bodyBlock)

	// ==========================================
	// ГЛОБАЛЬНЫЕ КНОПКИ НАВИГАЦИИ (Shell)
	// ==========================================
	var globalNavigation string

	if m.ActivePage == PageSystemChecks || m.ActivePage == PageInstallerPath || m.ActivePage == PageOptions || m.ActivePage == PageSummary || m.ActivePage == PageAskingSteamClose || m.ActivePage == PageShaders {

		btnStyle := lipgloss.NewStyle().Padding(0, 3).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Foreground(lipgloss.Color("240"))
		activeBtnStyle := btnStyle.Copy().BorderForeground(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("62"))

		isBackActive := m.NavFocus == 1
		isNextActive := m.NavFocus == 2

		btnBackText := m.GetBackButtonText()
		btnBack := btnStyle.Render(btnBackText)
		if isBackActive {
			btnBack = activeBtnStyle.Render(btnBackText)
		}

		btnNextText := m.GetNextButtonText()
		btnNext := btnStyle.Render(btnNextText)
		if isNextActive {
			btnNext = activeBtnStyle.Render(btnNextText)
		}

		navRow := lipgloss.JoinHorizontal(lipgloss.Center, btnBack, "        ", btnNext)
		globalNavigation = lipgloss.NewStyle().Width(boxWidth).Align(lipgloss.Center).MarginTop(1).Render(navRow)
	}

	// ==========================================
	// ПОДВАЛ
	// ==========================================
	footerText := " [Ctrl+L] - Логи | [Ctrl+C] - Выход "
	if m.Done {
		footerText = " [Ctrl+C] - Выход "
	}
	footer := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Render(footerText)

	var mainUI string
	var divider, disclaimer string

	if m.ActivePage == PageShaders {
		// На странице шейдеров убираем разделитель и дисклеймер, экономя место для картинки
		mainUI = lipgloss.JoinVertical(lipgloss.Left, header, body, globalNavigation, footer)
	} else {
		// На остальных страницах всё по-старому
		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color("237")).
			MarginTop(1).
			MarginBottom(1).
			Render(strings.Repeat("─", boxWidth))

		disclaimerText := "Не является Официальным продуктом: Requiem For A Dream by Immersive Chicken\nВсе фиксы были найдены официальным Discord сообществом RFAD"
		disclaimer := lipgloss.NewStyle().
			Width(boxWidth).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("239")).
			Render(disclaimerText)

		mainUI = lipgloss.JoinVertical(lipgloss.Left, header, body, globalNavigation, footer, divider, disclaimer)
	}

	// ==========================================
	// СБОРКА ИНТЕРФЕЙСА
	// ==========================================
	mainUI = lipgloss.JoinVertical(lipgloss.Left, header, body, globalNavigation, footer, divider, disclaimer)

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(mainUI)

	finalUI := dialogBox
	logBoxWidth := 45 // Фиксированная ширина боковой панели

	// Добавление логов сбоку
	if m.ShowLogs {
		dialogHeight := lipgloss.Height(dialogBox)

		logStyle := lipgloss.NewStyle().
			Width(logBoxWidth).
			Height(dialogHeight-2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginLeft(2).
			Foreground(lipgloss.Color("248"))

		logBox := logStyle.Render(m.LogLines)
		finalUI = lipgloss.JoinHorizontal(lipgloss.Top, dialogBox, logBox)
	}

	// Сохраняем габариты ДО центрирования для математики координат
	uiWidth := lipgloss.Width(finalUI)
	uiHeight := lipgloss.Height(finalUI)
	dialogWidth := lipgloss.Width(dialogBox)

	// Центрирование на экране всего интерфейса
	finalUI = lipgloss.Place(
		m.WindowWidth,
		m.WindowHeight,
		lipgloss.Center,
		lipgloss.Center,
		finalUI,
	)

	// ==========================================
	// ВНЕДРЕНИЕ ГРАФИКИ (Абсолютное позиционирование)
	// ==========================================
	// Команда очистки ВСЕХ изображений Kitty
	clearCmd := "\x1b_Ga=d,d=a;\x1b\\"

	if m.ActivePage == PageShaders {
		kittyImage := m.PageShaders.GetKittyImage()
		if kittyImage != "" {
			startY := (m.WindowHeight - uiHeight) / 2
			startX := (m.WindowWidth - uiWidth) / 2

			if startY < 0 {
				startY = 0
			}
			if startX < 0 {
				startX = 0
			}

			imgOffsetY := 19
			if m.showAscii() && m.AsciiArt != "" {
				imgOffsetY += lipgloss.Height(header) - 2
			}

			imgY := startY + imgOffsetY + 1

			imgX := startX + (dialogWidth-70)/2 + 1 + 8

			drawCmd := fmt.Sprintf("%s\x1b[s\x1b[%d;%dH%s\x1b[u", clearCmd, imgY, imgX, kittyImage)

			return finalUI + drawCmd
		}
	}

	return finalUI + clearCmd
}
