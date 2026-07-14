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
	startChan      chan *tui.InstallConfig
	steamReplyChan chan bool

	WindowWidth  int
	WindowHeight int

	Page0 SystemChecksPage
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

func NewIndex(startChan chan *tui.InstallConfig, ascii string, isSudo, hasWine, hasGameMode, hasNVAPI bool) Index {
	cfg := tui.NewInstallConfig()
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
		IsSudo:      isSudo,
		HasWine:     hasWine,
		HasGameMode: hasGameMode,
		HasNVAPI:    hasNVAPI,
	}

	return Index{
		Config:       cfg,
		SystemChecks: checks,
		ActivePage:   PageSystemChecks,
		startChan:    startChan,
		AsciiArt:     ascii,
		BoxWidth:     boxWidth,

		Page0:  NewSystemChecksPage(checks),
		Page1:  NewInstallerPathPage(cfg),
		Page2:  NewInstallPathPage(cfg),
		Page3:  NewOptionsPage(cfg),
		Page4:  NewSummaryPage(cfg),
		Status: components.NewStatusBar(),
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
	return tea.Batch(m.Page0.Init(), m.Page1.Init(), m.Page2.Init(), m.Page3.Init())
}

func (m Index) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 1. Обработка глобальных нажатий клавиш
	if keyMsg, ok := msg.(tea.KeyMsg); ok {

		// Блокировка Enter на странице системных проверок при критических ошибках
		if m.ActivePage == PageSystemChecks {
			// Если sudo — Enter не работает, только Ctrl+C
			if m.SystemChecks.IsSudo {
				if keyMsg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				return m, nil
			}
			// Если нет wine — Enter не работает, только Ctrl+C
			if !m.SystemChecks.HasWine {
				if keyMsg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				return m, nil
			}
			// Иначе Enter разрешён — обрабатывается в Update страницы
		}

		// Если мы находимся на экране вопроса о Steam, перехватываем клавиши
		if m.ActivePage == PageAskingSteamClose {
			switch keyMsg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "y", "д", "enter":
				if m.steamReplyChan != nil {
					m.steamReplyChan <- true
					m.steamReplyChan = nil
				}
				m.ActivePage = PageInstalling
				return m, nil
			case "n", "н", "esc":
				if m.steamReplyChan != nil {
					m.steamReplyChan <- false
					m.steamReplyChan = nil
				}
				m.ActivePage = PageInstalling
				return m, nil
			}
			return m, nil
		}

		// Стандартная обработка для остальных экранов
		switch keyMsg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "i", "I", "ш", "Ш":
			m.ShowLogs = !m.ShowLogs
			if m.ShowLogs {
				m.LogLines = tailLogs(core.LogPath(), 10)
				return m, tickLogs()
			}
			return m, nil
		}
	}

	// 2. Обработка системных сообщений
	switch msg := msg.(type) {
	case LogTickMsg:
		if m.ShowLogs {
			m.LogLines = tailLogs(core.LogPath(), 10)
			cmds = append(cmds, tickLogs())
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
		go func() {
			m.startChan <- m.Config
		}()
		return m, nil

	case PromptSteamCloseMsg:
		m.steamReplyChan = msg.ReplyChan
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
			m.LogLines = tailLogs(core.LogPath(), 10)
		}
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case DoneMsg:
		m.Done = true
		m.Status.Message = " Установка успешно завершена!"
		m.Status.SetPercent(1.0)
		return m, nil
	}

	// 3. Маршрутизация обновлений по страницам
	var cmd tea.Cmd
	switch m.ActivePage {
	case PageSystemChecks:
		m.Page0, cmd = m.Page0.Update(msg)
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
		return " Инициализация терминала..."
	}

	boxWidth := m.BoxWidth

	// 1. Шапка (ASCII-арт)
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
			Render("=== TUI УСТАНОВЩИК RFAD SE ===")
	}

	// 2. Тело (Текущий экран)
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
		case PageInstallPath:
			rawBody = m.Page2.View()
		case PageOptions:
			rawBody = m.Page3.View()
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

	bodyBlock := lipgloss.NewStyle().Align(lipgloss.Left).Render(rawBody)
	body := lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, bodyBlock)

	// 3. Подвал
	footerText := " Нажмите 'i' для логов | 'ctrl+c' для выхода "
	if m.Done {
		footerText = " Нажмите 'ctrl+c' для выхода "
	}
	footer := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Render(footerText)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		MarginTop(1).
		MarginBottom(1).
		Render(strings.Repeat("─", boxWidth))

	disclaimerText := " Не является Официальным продуктом: Requiem For A Dream by Immersive Chicken,\n все фиксы были найдены официальным Discord сообществом RFAD"
	disclaimer := lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("239")).
		Render(disclaimerText)

	ui := lipgloss.JoinVertical(lipgloss.Left, header, body, footer, divider, disclaimer)

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Render(ui)

	// 4. Панель логов
	finalUI := dialogBox
	if m.ShowLogs {
		logStyle := lipgloss.NewStyle().
			Width(boxWidth+2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			MarginTop(1).
			Foreground(lipgloss.Color("248"))
		logBox := logStyle.Render(m.LogLines)
		finalUI = lipgloss.JoinVertical(lipgloss.Center, dialogBox, logBox)
	}

	return lipgloss.Place(
		m.WindowWidth,
		m.WindowHeight,
		lipgloss.Center,
		lipgloss.Center,
		finalUI,
	)
}
