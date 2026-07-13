package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"

	"rfad-installer/core"
	"rfad-installer/tui"
	"rfad-installer/tui/pages"
)

//go:embed src/*
var bundledAssets embed.FS

func RunSystemChecks() (bool, bool, bool) {
	core.LogInfo("=== Выполнение предполетных проверок ===")

	if os.Geteuid() == 0 {
		return false, false, false
	}
	if _, err := exec.LookPath("portproton"); err != nil {
		return false, false, false
	}

	_, errGM := exec.LookPath("gamemoderun")
	hasGameMode := errGM == nil

	hasNVAPI := false

	if _, err := os.Stat("/proc/driver/nvidia"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
		out, err := cmd.Output()
		if err == nil {
			gpuName := strings.ToUpper(string(out))
			if strings.Contains(gpuName, "RTX 20") ||
				strings.Contains(gpuName, "RTX 30") ||
				strings.Contains(gpuName, "RTX 40") ||
				strings.Contains(gpuName, "RTX A") {
				hasNVAPI = true
			}
		}
	}

	core.LogInfo("Системные проверки: root=%v, portproton=%v, GameMode=%v, NVAPI=%v", false, true, hasGameMode, hasNVAPI)
	return true, hasGameMode, hasNVAPI
}

// ensureTerminal проверяет наличие TTY и перезапускает программу в графическом терминале
func ensureTerminal() {
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsTerminal(os.Stderr.Fd()) {
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}

	terminals := []struct {
		exec string
		arg  string
	}{
		{"konsole", "-e"},        // KDE (Steam Deck в режиме рабочего стола)
		{"gnome-terminal", "--"}, // GNOME (Ubuntu, Fedora)
		{"xfce4-terminal", "-x"}, // XFCE (Linux Mint, Manjaro XFCE)
		{"alacritty", "-e"},      // Популярный у продвинутых юзеров
		{"kitty", "--"},          // Ещё один популярный GPU-терминал
		{"terminator", "-x"},     // Часто используется разработчиками
		{"mate-terminal", "-x"},  // MATE
		{"lxterminal", "-e"},     // LXDE
		{"xterm", "-e"},          // Универсальный фолбэк (есть почти везде)
	}

	launched := false
	for _, term := range terminals {
		if _, err := exec.LookPath(term.exec); err == nil {
			cmd := exec.Command(term.exec, term.arg, exePath)
			if err := cmd.Start(); err == nil {
				launched = true
				break
			}
		}
	}

	if !launched {
		msg := "Это консольный установщик. Пожалуйста, откройте терминал и запустите файл вручную:\n" + exePath
		if _, err := exec.LookPath("kdialog"); err == nil {
			exec.Command("kdialog", "--error", msg).Run()
		} else if _, err := exec.LookPath("zenity"); err == nil {
			exec.Command("zenity", "--error", "--text", msg).Run()
		}
	}

	os.Exit(0)
}

// openLogTerminal открывает отдельное окно терминала с tail -f файла лога.
func openLogTerminal(logPath string) {
	if logPath == "" {
		return
	}

	terminals := []struct {
		exec string
		arg  string
	}{
		{"konsole", "-e"},
		{"gnome-terminal", "--"},
		{"xfce4-terminal", "-x"},
		{"alacritty", "-e"},
		{"kitty", "--"},
		{"terminator", "-x"},
		{"mate-terminal", "-x"},
		{"lxterminal", "-e"},
		{"xterm", "-e"},
	}

	for _, term := range terminals {
		if _, err := exec.LookPath(term.exec); err == nil {
			cmd := exec.Command(term.exec, term.arg, "tail", "-f", logPath)
			cmd.Start()
			return
		}
	}
}

func getFSRPatches(enabled bool, resolution string) []core.ConfigPatch {
	if !enabled {
		return nil
	}
	return []core.ConfigPatch{
		{
			TargetFile: "MO2/mods/SSE Display Tweaks/SKSE/Plugins/SSEDisplayTweaks.ini",
			Replace: map[string]string{
				"Fullscreen=false": "Fullscreen=true",
				"Borderless=true":  "Borderless=false",
			},
			InsertAfter: map[string]string{
				"[Render]": fmt.Sprintf("Resolution=%s", resolution),
			},
		},
	}
}

func main() {
	ensureTerminal()

	showLogs := flag.Bool("show-logs", false, "Открыть окно терминала с логами в реальном времени")
	flag.Parse()

	if err := core.InitLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: %v\n", err)
	}
	defer core.CloseLogger()

	if *showLogs {
		openLogTerminal(core.LogPath())
	}

	core.LogInfo("Запуск Rfad-TUI-Linux-Installer")

	passed, useGameMode, useNVAPI := RunSystemChecks()
	if !passed {
		core.LogError("Системные проверки не пройдены, выход")
		os.Exit(1)
	}
	core.LogInfo("Системные проверки пройдены: GameMode=%v, NVAPI=%v", useGameMode, useNVAPI)

	startChan := make(chan *tui.InstallConfig)

	// Читаем ASCII арт из встроенных ресурсов
	asciiBytes, err := bundledAssets.ReadFile("src/install.ascii")
	asciiArt := ""
	if err == nil {
		asciiArt = string(asciiBytes)
	} else {
		core.LogWarn("Не удалось загрузить install.ascii: %v", err)
	}

	// Передаем asciiArt в NewIndex
	page := pages.NewIndex(startChan, asciiArt)
	p := tea.NewProgram(page, tea.WithInput(os.Stdin))

	go func() {
		cfg := <-startChan

		folderID := "1JUOctbsugh2IIEUCWcBkupXYVYoJMg4G"
		archiveDest := "update.zip"

		gamePath := cfg.InstallPath

		time.Sleep(500 * time.Millisecond)

		// === ЭТАП 0: ЧЕСТНАЯ УСТАНОВКА ЧЕРЕЗ WINE ===
		core.LogInfo("=== ЭТАП 0: Установка базовой игры через Wine ===")

		components := "rfad_se,enb"
		if cfg.GraphicsMod == "ReShade" {
			components = "rfad_se,reshade"
		}

		winInstallPath := "Z:" + strings.ReplaceAll(cfg.InstallPath, "/", "\\")

		infContent := fmt.Sprintf(`[Setup]
			Lang=english
			Dir=%s
			Group=RfaD SE
			NoIcons=1
			SetupType=custom
			Components=%s
			Tasks=
		`, winInstallPath, components)

		infPath := filepath.Join(os.TempDir(), "rfad_install.inf")
		os.WriteFile(infPath, []byte(infContent), 0644)

		winInfPath := "Z:" + strings.ReplaceAll(infPath, "/", "\\")

		err := core.ExtractInstaller(
			cfg.InstallerPath,
			cfg.InstallPath,
			winInfPath,
			cfg.GraphicsMod,
			func(percent float64, detail string) {
				p.Send(pages.ProgressMsg{
					Percent: percent,
					Message: detail,
				})
			},
		)

		os.Remove(infPath)

		if err != nil {
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		// === ЭТАП 2: ЗАГРУЗКА ОБНОВЛЕНИЯ ===
		core.LogInfo("=== ЭТАП 1: Загрузка обновления ===")

		creds, err := bundledAssets.ReadFile("src/credentials.json")
		if err != nil {
			p.Send(pages.ErrorMsg{Err: fmt.Errorf("не найден файл ключей: %v", err)})
			return
		}

		err = core.DownloadArchive(
			context.Background(),
			creds,
			folderID,
			archiveDest,
			func(percent float64) {
				p.Send(pages.ProgressMsg{
					Percent: percent,
					Message: "Загрузка обновления...",
				})
			},
		)
		if err != nil {
			core.LogError("Ошибка загрузки обновления: %v", err)
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		core.LogInfo("Обновление загружено успешно: %s", archiveDest)
		time.Sleep(500 * time.Millisecond)

		err = core.ProcessUpdate(
			gamePath,
			archiveDest,
			bundledAssets,
			func(percent float64, detail string) {
				p.Send(pages.ProgressMsg{
					Percent: percent,
					Message: detail,
				})
			},
		)
		if err != nil {
			core.LogError("Ошибка распаковки обновления: %v", err)
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		err = core.EnablePlugin(gamePath, "Rfad_Runes.esp")
		if err != nil {
			core.LogWarn("Не удалось автоматически включить Rfad_Runes.esp: %v", err)
		} else {
			core.LogInfo("Rfad_Runes.esp успешно добавлен в plugins.txt")
		}

		core.LogInfo("Распаковка обновления завершена")
		time.Sleep(500 * time.Millisecond)

		// === ЭТАП 3: РАСПАКОВКА ПРЕФИКСА WINE ===
		core.LogInfo("=== ЭТАП 3: Распаковка префикса Wine ===")

		prefixBase, err := core.GetPortProtonPrefixPath()
		if err == nil {
			oldPrefixPath := filepath.Join(prefixBase, "RFAD_SE")
			os.RemoveAll(oldPrefixPath)
		}

		prefixArchivePath := filepath.Join(cfg.InstallPath, "prefix.tar.gz")

		err = core.DownloadPrefixDirectly(context.Background(), creds, prefixArchivePath, func(percent float64) {
			p.Send(pages.ProgressMsg{
				Percent: percent,
				Message: fmt.Sprintf("Загрузка префикса: %.1f%%", percent*100),
			})
		})
		if err != nil {
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		core.LogInfo("Префикс скачан: %s", prefixArchivePath)
		err = core.ExtractPrefix(prefixArchivePath, func(fileName string) {
			p.Send(pages.ProgressMsg{
				Percent: -1,
				Message: fmt.Sprintf("Настройка префикса: %s", fileName),
			})
		})
		if err != nil {
			core.LogError("Ошибка распаковки префикса: %v", err)
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		os.Remove(prefixArchivePath)
		core.LogInfo("Распаковка префикса завершена, архив удалён")

		// === ЭТАП 4: ПРИМЕНЕНИЕ ФИКСОВ КОНФИГУРАЦИИ ===
		core.LogInfo("=== ЭТАП 4: Патчинг конфигурации ===")
		err = core.ApplyPatches(cfg, func(percent float64, fileName string) {
			p.Send(pages.ProgressMsg{
				Percent: percent,
				Message: fmt.Sprintf("Патчинг: %s", fileName),
			})
		})

		if err != nil {
			core.LogError("Ошибка патчинга: %v", err)
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		core.LogInfo("Патчинг завершён")

		// === ЭТАП 5: Генерация PPDB-файлов (БЫВШИЙ ЭТАП 6) ===
		core.LogInfo("=== ЭТАП 5: Генерация PPDB-файлов ===")
		mo2Path := filepath.Join(cfg.InstallPath, "MO2")
		orig := filepath.Join(mo2Path, "ModOrganizer.exe")
		link := filepath.Join(mo2Path, "ModOrganizerSKSE.exe")

		os.Remove(link)

		err = os.Link(orig, link)
		if err != nil {
			p.Send(pages.ErrorMsg{Err: fmt.Errorf("ошибка создания дубликата MO2: %v", err)})
			return
		}

		wineVersion := "GE-PROTON11-1"

		core.GeneratePPDB(cfg.InstallPath, "ModOrganizer.exe", wineVersion, cfg.UseFSR, useNVAPI, useGameMode, cfg.UseSteamFix)
		core.GeneratePPDB(cfg.InstallPath, "ModOrganizerSKSE.exe", wineVersion, cfg.UseFSR, useNVAPI, useGameMode, cfg.UseSteamFix)
		core.LogInfo("PPDB сгенерированы: wine=%s, FSR=%v, NVAPI=%v, GameMode=%v, SteamFix=%v", wineVersion, cfg.UseFSR, useNVAPI, useGameMode, cfg.UseSteamFix)

		// === ЭТАП 6: Установка steam fix ===
		core.LogInfo("=== ЭТАП 6: Steam Fix ===")

		if cfg.UseSteamFix {
			core.LogInfo("Применение Steam Fix...")

			err = core.ApplySteamFix(cfg.InstallPath, bundledAssets)
			if err != nil {
				core.LogError("Ошибка применения Steam Fix: %v", err)
				p.Send(pages.ErrorMsg{Err: err})
				return
			}

			steamExePath := "/usr/bin/portproton"

			startDir := filepath.Join(cfg.InstallPath, "MO2")

			targetExe := filepath.Join(startDir, "ModOrganizerSKSE.exe")
			launchOptions := fmt.Sprintf("\"%s\"", targetExe)

			err = core.AddToSteamShortcuts("RFAD Game (SKSE)", steamExePath, startDir, launchOptions)
			if err != nil {
				core.LogError("Ошибка добавления в Steam: %v", err)
				p.Send(pages.ErrorMsg{Err: err})
				return
			}

			core.RestartSteam()
			core.LogInfo("Steam Fix применён успешно, Steam перезапущен")
		} else {
			core.LogInfo("Steam Fix пропущен (отключён пользователем)")
		}

		// === Этап 7: Создание шорткатов
		core.LogInfo("=== ЭТАП 7: Создание ярлыков ===")
		if cfg.CreateShortcuts {
			err := core.CreateDesktopShortcuts(cfg.InstallPath, cfg.UseSteamFix, bundledAssets)
			if err != nil {
				core.LogError("Ошибка создания ярлыков: %v", err)
				p.Send(pages.ErrorMsg{Err: fmt.Errorf("ошибка создания ярлыков: %v", err)})
				return
			}
			core.LogInfo("Ярлыки созданы успешно")
		} else {
			core.LogInfo("Создание ярлыков пропущено (отключено пользователем)")
		}

		core.LogInfo("=== УСТАНОВКА ЗАВЕРШЕНА УСПЕШНО ===")
		p.Send(pages.DoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		core.LogError("Ошибка TUI: %v", err)
		fmt.Fprintf(os.Stderr, "Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
