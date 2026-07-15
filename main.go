package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func RunSystemChecks() (isSudo, hasWine, hasGameMode, hasNVAPI bool, screenWidth, screenHeight int) {
	core.LogInfo("=== Выполнение предполетных проверок ===")

	// 1. Запущен ли от root
	isSudo = os.Geteuid() == 0
	if isSudo {
		core.LogError("КРИТИЧЕСКАЯ ОШИБКА: Запуск от root запрещен!")
	} else {
		core.LogInfo("Запуск от обычного пользователя")
	}

	// 2. Проверка Wine/Proton (wine, wine64, proton)
	hasWine = checkBinary("proton")

	// Проверка PortProton (PATH, локальный путь, Flatpak)
	if !hasWine {
		if _, err := exec.LookPath("portproton"); err == nil {
			hasWine = true
			core.LogInfo("PortProton найден в системном PATH")
		} else {
			home, _ := os.UserHomeDir()
			localPP := filepath.Join(home, ".local", "bin", "portproton")
			if _, err := os.Stat(localPP); err == nil {
				hasWine = true
				core.LogInfo("PortProton найден по локальному пути: %s", localPP)
			} else {
				if _, err := exec.LookPath("flatpak"); err == nil {
					cmd := exec.Command("flatpak", "info", "ru.linux_gaming.PortProton")
					if err := cmd.Run(); err == nil {
						hasWine = true
						core.LogInfo("PortProton найден в реестре Flatpak")
					}
				}
			}
		}
	}

	if hasWine {
		core.LogInfo("PortProton обнаружен в системе")
	} else {
		core.LogError("КРИТИЧЕСКАЯ ОШИБКА: PortProton не найден в системе")
	}

	// 3. Проверка gamemoderun
	hasGameMode = checkBinary("gamemoderun") || checkBinary("gamemoded")
	if hasGameMode {
		core.LogInfo("gamemoderun найден в системе")
	} else {
		core.LogWarn("gamemoderun не найден, производительность может быть ниже ожидаемой")
	}

	// 4. Проверка NVAPI
	hasNVAPI = false
	if _, err := os.Stat("/proc/driver/nvidia"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
		out, err := cmd.Output()
		if err == nil {
			gpuName := strings.ToUpper(strings.TrimSpace(string(out)))
			if strings.Contains(gpuName, "RTX 20") ||
				strings.Contains(gpuName, "RTX 30") ||
				strings.Contains(gpuName, "RTX 40") ||
				strings.Contains(gpuName, "RTX A") {
				hasNVAPI = true
				core.LogInfo("Найдена видеокарта с поддержкой NVAPI: %s", gpuName)
			}
		}
	}
	if !hasNVAPI {
		core.LogInfo("NVAPI не требуется или видеокарта не NVIDIA")
	}

	screenWidth, screenHeight = 1920, 1080 // Значения по умолчанию
	out, err := exec.Command("sh", "-c", "xrandr | grep '\\*' | awk '{print $1}'").Output()
	if err == nil {
		parts := strings.Split(strings.TrimSpace(string(out)), "x")
		if len(parts) == 2 {
			if w, err := strconv.Atoi(parts[0]); err == nil {
				screenWidth = w
			}
			if h, err := strconv.Atoi(parts[1]); err == nil {
				screenHeight = h
			}
		}
	}
	core.LogInfo("Базовое разрешение экрана определено как: %dx%d", screenWidth, screenHeight)

	return
}

func checkBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
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
		{"konsole", "-e"},
		{"gnome-terminal", "--"},
		{"kgx", "-e"},
		{"ptyxis", "--"},
		{"xfce4-terminal", "-x"},
		{"alacritty", "-e"},
		{"wezterm", "-e"},
		{"kitty", "--"},
		{"foot", "-e"},
		{"terminator", "-x"},
		{"mate-terminal", "-x"},
		{"lxterminal", "-e"},
		{"xterm", "-e"},
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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func main() {
	ensureTerminal()

	showLogs := flag.Bool("show-logs", false, "Открыть окно терминала с логами в реальном времени")
	localTest := flag.Bool("local-test", false, "Использовать закешированные архивы для быстрого тестирования")
	flag.Parse()

	if err := core.InitLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Предупреждение: %v\n", err)
	}
	defer core.CloseLogger()

	if *showLogs {
		openLogTerminal(core.LogPath())
	}

	core.LogInfo("Запуск Rfad-TUI-Linux-Installer")

	isSudo, hasWine, hasGameMode, hasNVAPI, screenWidth, screenHeight := RunSystemChecks()
	core.LogInfo("Системные проверки пройдены: Sudo=%v, Wine=%v, GameMode=%v, NVAPI=%v, Screan=%v x %v", isSudo, hasWine, hasGameMode, hasNVAPI, screenWidth, screenHeight)

	startChan := make(chan *tui.InstallConfig)

	asciiBytes, err := bundledAssets.ReadFile("src/install.ascii")
	asciiArt := ""
	if err == nil {
		asciiArt = string(asciiBytes)
	} else {
		core.LogWarn("Не удалось загрузить install.ascii: %v", err)
	}

	page := pages.NewIndex(startChan, asciiArt, isSudo, hasWine, hasGameMode, hasNVAPI, screenWidth, screenHeight)
	p := tea.NewProgram(page, tea.WithInput(os.Stdin))

	go func() {
		cfg := <-startChan

		folderID := "1JUOctbsugh2IIEUCWcBkupXYVYoJMg4G"

		gamePath := cfg.InstallPath

		cacheDir := filepath.Join(".", "local_cache")
		os.MkdirAll(cacheDir, 0755)
		core.LogInfo("Временная папка загрузок установлена в: %s", cacheDir)

		// === ЭТАП 0.5: ПОДГОТОВКА СВОЕГО ДВИЖКА (PORTABLE WINE) ===
		core.LogInfo("=== ЭТАП 0.5: Подготовка Portable Wine ===")

		portableWineDir := filepath.Join(cacheDir, "wine")
		wineExePath := filepath.Join(portableWineDir, "bin", "wine")

		if !fileExists(wineExePath) {
			err := core.GetPortableWine(cacheDir, portableWineDir, func(percent float64, detail string) {
				p.Send(pages.ProgressMsg{
					Percent: percent,
					Message: detail,
				})
			})
			if err != nil {
				core.LogError("Ошибка подготовки Wine: %v", err)
				p.Send(pages.ErrorMsg{Err: fmt.Errorf("ошибка загрузки движка: %v", err)})
				return
			}
		}

		// === ЭТАП 1: УСТАНОВКА ЧИСТОЙ ИГРЫ ЧЕРЕЗ WINE ===
		core.LogInfo("=== ЭТАП 1: Установка базовой игры через Wine ===")

		components := "rfad_se"
		if cfg.GraphicsMod == "ENB" {
			components += ",enb"
		} else if cfg.GraphicsMod == "ReShade" {
			components += ",reshade"
		}

		winInstallPath := "Z:" + strings.ReplaceAll(cfg.InstallPath, "/", "\\")

		infContent := fmt.Sprintf(`[Setup]
Lang=english
Dir=%s
Group=RfaD SE
NoIcons=1
SetupType=custom
Components=%s
Tasks=`, winInstallPath, components)

		os.MkdirAll(cfg.InstallPath, 0755)
		infPath := filepath.Join(cfg.InstallPath, "rfad_install.inf")
		os.WriteFile(infPath, []byte(infContent), 0644)

		winInfPath := "Z:" + strings.ReplaceAll(infPath, "/", "\\")

		var isSuccess bool

		defer func() {
			if !isSuccess && !*localTest {
				core.LogWarn("Установка прервана или завершилась ошибкой. Очистка сломанной директории: %s", gamePath)
				p.Send(pages.ProgressMsg{
					Percent: -1,
					Message: "Очистка неудачной установки...",
				})
				os.RemoveAll(gamePath)
			}
		}()

		time.Sleep(500 * time.Millisecond)

		err = core.ExtractInstaller(
			wineExePath,
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

		// Удаляем временный inf-файл после установки
		os.Remove(infPath)

		if err != nil {
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		// === ЭТАП 2: ЗАГРУЗКА И УСТАНОВКА ОБНОВЛЕНИЯ ===
		core.LogInfo("=== ЭТАП 2: Загрузка обновления ===")

		creds, err := bundledAssets.ReadFile("src/credentials.json")
		if err != nil {
			p.Send(pages.ErrorMsg{Err: fmt.Errorf("не найден файл ключей: %v", err)})
			return
		}

		updateArchivePath := filepath.Join(cacheDir, "update.zip")

		if fileExists(updateArchivePath) {
			core.LogInfo("Найден кэшированный архив %s, загрузка пропущена", updateArchivePath)
			p.Send(pages.ProgressMsg{Percent: 1.0, Message: "Используется локальный архив обновления..."})
			time.Sleep(500 * time.Millisecond)
		} else {
			err = core.DownloadArchive(
				context.Background(),
				creds,
				folderID,
				updateArchivePath,
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
			core.LogInfo("Обновление загружено успешно: %s", updateArchivePath)
			time.Sleep(500 * time.Millisecond)
		}

		err = core.ProcessUpdate(
			gamePath,
			updateArchivePath,
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

		// === ЭТАП 3: ЗАГРУЗКА И РАСПАКОВКА ПРЕФИКСА WINE ===
		core.LogInfo("=== ЭТАП 3: Распаковка префикса Wine ===")

		prefixBase, err := core.GetPortProtonPrefixPath()
		if err == nil {
			oldPrefixPath := filepath.Join(prefixBase, "RFAD_SE")
			os.RemoveAll(oldPrefixPath)
		}

		prefixArchivePath := filepath.Join(cacheDir, "prefix.tar.gz")

		if fileExists(prefixArchivePath) {
			core.LogInfo("Найден кэшированный префикс %s, загрузка пропущена", prefixArchivePath)
			p.Send(pages.ProgressMsg{Percent: 1.0, Message: "Используется локальный архив префикса..."})
			time.Sleep(500 * time.Millisecond)
		} else {
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
		}

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

		// === ЭТАП 5: Генерация PPDB-файлов ===
		core.LogInfo("=== ЭТАП 5: Генерация PPDB-файлов ===")
		mo2Path := filepath.Join(cfg.InstallPath, "MO2")
		wineVersion := "GE-PROTON11-1"
		orig := filepath.Join(mo2Path, "ModOrganizer.exe")

		if cfg.UseSteamFix {
			// Внедряем Python-плагин для генерации маркера
			// Необходим для коректного запуска RFAD из steam в случае если mo2
			// был запущен раньше чем игра, поскольку в запуск игры вшит одноразовый старт mo2
			// для открытия контекстного окна portproton с дозогрузкой библиотек
			if err := core.CreateMO2PythonPlugin(cfg.InstallPath); err != nil {
				core.LogWarn("Не удалось создать Python-плагин: %v", err)
			}
		} else {
			// Генерация хардлинка для пиратки
			link := filepath.Join(mo2Path, "ModOrganizerSKSE.exe")
			os.Remove(link)
			if err = os.Link(orig, link); err != nil {
				p.Send(pages.ErrorMsg{Err: fmt.Errorf("ошибка создания дубликата MO2: %v", err)})
				return
			}
			core.GeneratePPDB(cfg.InstallPath, "ModOrganizerSKSE.exe", wineVersion, cfg.UseFSR, hasNVAPI, hasGameMode)
		}

		core.GeneratePPDB(cfg.InstallPath, "ModOrganizer.exe", wineVersion, cfg.UseFSR, hasNVAPI, hasGameMode)

		core.LogInfo("PPDB сгенерированы: wine=%s, FSR=%v, NVAPI=%v, GameMode=%v, SteamFix=%v", wineVersion, cfg.UseFSR, hasNVAPI, hasGameMode, cfg.UseSteamFix)

		// === ЭТАП 6: Steam Fix и интеграция со Steam ===
		if cfg.UseSteamFix {
			core.LogInfo("=== ЭТАП 6: Настройка ярлыка Steam ===")

			core.LogInfo("Ожидание подтверждения на закрытие Steam...")
			replyChan := make(chan bool)
			p.Send(pages.PromptSteamCloseMsg{
				ReplyChan: replyChan,
			})

			shouldContinue := <-replyChan

			if !shouldContinue {
				core.LogWarn("Пользователь отказался закрывать Steam. Интеграция пропущена.")
			} else {
				core.LogInfo("Пользователь дал согласие. Закрываем Steam...")

				core.ShutdownSteam()

				err = core.ApplySteamFix(cfg.InstallPath, bundledAssets)
				if err != nil {
					core.LogError("ВНИМАНИЕ: Не удалось применить Steam Fix: %v", err)
					return
				}

				err = core.InstallGEProton("GE-Proton11-1")
				if err != nil {
					core.LogError("ВНИМАНИЕ: Не удалось установить GE-Proton: %v", err)
				}

				prelaunchScript, err := core.CreateSteamPrelaunchScript(cfg.InstallPath)
				if err != nil {
					core.LogError("Ошибка создания prelaunch-скрипта: %v", err)
				}

				mo2ExePath := filepath.Join(cfg.InstallPath, "MO2", "ModOrganizer.exe")
				startDir := filepath.Join(cfg.InstallPath, "MO2")

				// ДИНАМИЧЕСКАЯ СБОРКА ФЛАГОВ ЗАПУСКА
				var launchParts []string
				launchParts = append(launchParts, "STEAM_APP_ID=489830")
				launchParts = append(launchParts, `WINEDLLOVERRIDES="xaudio2_7=n,b;d3d11=n,b;d3dx9_42=n,b;d3dcompiler_47=n,b;dinput8=n,b;mscoree=n"`)

				if cfg.UseFSR {
					launchParts = append(launchParts, "WINE_FULLSCREEN_FSR=1", "WINE_FULLSCREEN_FSR_STRENGTH=2")
				}

				if hasNVAPI {
					launchParts = append(launchParts, "PROTON_ENABLE_NVAPI=1")
				}

				if hasGameMode {
					launchParts = append(launchParts, "gamemoderun")
				}

				mainCmd := fmt.Sprintf(`bash "%s" %%command%% "moshortcut://:SKSE"`, prelaunchScript)
				launchParts = append(launchParts, mainCmd)

				launchOpts := strings.Join(launchParts, " ")

				err = core.AddToSteamShortcuts("RFAD Game (License)", mo2ExePath, startDir, launchOpts)
				if err != nil {
					core.LogError("Ошибка при добавлении лицензионного ярлыка в Steam: %v", err)
				} else {
					core.LogInfo("Лицензионный ярлык успешно добавлен.")
				}
				core.RestartSteam()
			}
		}

		// === Этап 7: Создание шорткатов ===
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

		isSuccess = true

		core.LogInfo("=== УСТАНОВКА ЗАВЕРШЕНА УСПЕШНО ===")

		if !*localTest {
			core.LogInfo("Очистка временных файлов загрузки: %s", cacheDir)
			os.RemoveAll(cacheDir)
		} else {
			core.LogInfo("ВНИМАНИЕ: Флаг --local-test активен. Кэш сохранен в: %s", cacheDir)
		}

		p.Send(pages.DoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		core.LogError("Ошибка TUI: %v", err)
		fmt.Fprintf(os.Stderr, "Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
