package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"rfad-installer/core"
	"rfad-installer/tui"
	"rfad-installer/tui/pages"
)

//go:embed src/*
var bundledAssets embed.FS

func RunSystemChecks() (bool, bool, bool) {
	fmt.Println("=== Выполнение предполетных проверок ===")

	// 1. Проверка на sudo (Критическая)
	if os.Geteuid() == 0 {
		return false, false, false
	}

	// 2. Проверка PortProton
	if _, err := exec.LookPath("portproton"); err != nil {
		return false, false, false
	}

	// 3. Проверка GameMode
	_, errGM := exec.LookPath("gamemoderun")
	hasGameMode := errGM == nil
	if hasGameMode {
		fmt.Println(" [OK] Feral GameMode найден.")
	} else {
		fmt.Println(" [ПРЕДУПРЕЖДЕНИЕ] GameMode не найден.")
	}

	// 4. Проверка NVAPI (NVIDIA 20-й серии и старше)
	hasNVAPI := false
	cmd := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
	out, err := cmd.Output()
	if err == nil {
		gpuName := strings.ToUpper(string(out))
		// Список серий, поддерживающих полноценный NVAPI для Proton
		// RTX 20xx, 30xx, 40xx
		if strings.Contains(gpuName, "RTX 20") ||
			strings.Contains(gpuName, "RTX 30") ||
			strings.Contains(gpuName, "RTX 40") ||
			strings.Contains(gpuName, "RTX A") { // Профессиональные серии
			hasNVAPI = true
			fmt.Println(" [OK] NVIDIA GPU 20+ серии обнаружен, NVAPI будет включен.")
		}
	}

	return true, hasGameMode, hasNVAPI
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
	passed, useGameMode, useNVAPI := RunSystemChecks()
	if !passed {
		os.Exit(1)
	}

	// Сохраняем эти bool в объект конфигурации или используем для запуска
	fmt.Printf("Итого: GameMode=%v, NVAPI=%v\n", useGameMode, useNVAPI)

	// 1. Создаем канал для связи UI и фонового установщика
	startChan := make(chan *tui.InstallConfig)

	// 2. Передаем канал в роутер, чтобы Страница 4 смогла отправить в него конфиг
	page := pages.NewIndex(startChan)
	p := tea.NewProgram(page)

	// Наш фоновый процесс установки
	go func() {
		// КРИТИЧЕСКИЙ МОМЕНТ: Горутина "засыпает" на этой строчке и ждет,
		// пока пользователь не прокликает весь опросник и не нажмет Enter на последнем экране
		cfg := <-startChan

		folderID := "1JUOctbsugh2IIEUCWcBkupXYVYoJMg4G"
		archiveDest := "update.zip"

		// Теперь мы берем пути не из пустой переменной, а из реального выбора пользователя
		gamePath := cfg.InstallPath

		time.Sleep(500 * time.Millisecond)

		// === ЭТАП 0: РАСПАКОВКА БАЗОВОЙ ИГРЫ ЧЕРЕЗ INNOEXTRACT ===
		err := core.ExtractInstaller(
			cfg.InstallerPath,
			gamePath,
			func(percent float64, detail string) {
				p.Send(pages.ProgressMsg{
					Percent: percent,
					Message: detail,
				})
			},
		)
		if err != nil {
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		time.Sleep(500 * time.Millisecond)

		// Читаем ключи от Google Drive только после успешной распаковки
		creds, err := bundledAssets.ReadFile("src/credentials.json")
		if err != nil {
			p.Send(pages.ErrorMsg{Err: fmt.Errorf("не найден файл ключей: %v", err)})
			return
		}

		// === ЭТАП 1: ЗАГРУЗКА ОБНОВЛЕНИЯ ===
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
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		time.Sleep(500 * time.Millisecond)

		// === ЭТАП 2: РАСПАКОВКА АРХИВА ===
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
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		time.Sleep(500 * time.Millisecond)

		// === ЭТАП 3: РАСПАКОВКА ПРЕФИКСА WINE ===

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

		err = core.ExtractPrefix(prefixArchivePath, func(fileName string) {
			p.Send(pages.ProgressMsg{
				Percent: -1,
				Message: fmt.Sprintf("Настройка префикса: %s", fileName),
			})
		})
		if err != nil {
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		os.Remove(prefixArchivePath)

		// === ЭТАП 4: ПРИМЕНЕНИЕ ФИКСОВ КОНФИГУРАЦИИ ===
		err = core.ApplyPatches(cfg, func(percent float64, fileName string) {
			p.Send(pages.ProgressMsg{
				Percent: percent,
				Message: fmt.Sprintf("Патчинг: %s", fileName),
			})
		})

		if err != nil {
			p.Send(pages.ErrorMsg{Err: err})
			return
		}

		// === Этап 5: установка steam fix

		if cfg.UseSteamFix {
			err := core.ApplySteamFix(cfg.InstallPath)
			if err != nil {
				p.Send(pages.ErrorMsg{Err: err})
				return
			}
		}

		// === Этап 6: Генерация .ppdb
		mo2Path := filepath.Join(cfg.InstallPath, "MO2")
		orig := filepath.Join(mo2Path, "ModOrganizer.exe")
		link := filepath.Join(mo2Path, "ModOrganizerSKSE.exe")

		os.Remove(link)

		err = os.Link(orig, link)
		if err != nil {
			p.Send(pages.ErrorMsg{Err: fmt.Errorf("ошибка создания дубликата MO2: %v", err)})
			return
		}

		core.GeneratePPDB(cfg.InstallPath, "ModOrganizer.exe", cfg.UseFSR, useNVAPI, useGameMode, cfg.UseSteamFix)
		core.GeneratePPDB(cfg.InstallPath, "ModOrganizerSKSE.exe", cfg.UseFSR, useNVAPI, useGameMode, cfg.UseSteamFix)

		// === Этап 7: Создание шорткатов
		if cfg.CreateShortcuts {
			err := core.CreateDesktopShortcuts(cfg.InstallPath, cfg.UseSteamFix)
			if err != nil {
				p.Send(pages.ErrorMsg{Err: fmt.Errorf("ошибка создания ярлыков: %v", err)})
				return
			}
		}

		p.Send(pages.DoneMsg{})
	}()

	// Запускаем отрисовку интерфейса
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка TUI: %v\n", err)
		os.Exit(1)
	}
}
