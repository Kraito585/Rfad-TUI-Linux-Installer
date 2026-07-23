package core

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"rfad-installer/tui"
	"strings"
	"time"
)

func sanitize(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9./\\ \-_]`)
	return reg.ReplaceAllString(s, "")
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// Добавлен аргумент embeddedFiles типа embed.FS
func ExtractInstaller(installerPath, installPath string, cfg *tui.InstallConfig, graphicsMod string, cacheDir string, embeddedFiles embed.FS, progressCb func(float64, string)) error {
	installerPath = strings.Trim(installerPath, "\"' ")
	installPath = strings.Trim(installPath, "\"' ")

	LogUnpacking("ExtractInstaller: подготовка автономного innoextract...")

	// 1. Извлекаем innoextract из вшитых ассетов
	innoBinPath := filepath.Join(cacheDir, "innoextract")
	innoData, err := embeddedFiles.ReadFile("src/innoextract")
	if err != nil {
		LogError("ExtractInstaller: не удалось найти innoextract в ресурсах: %v", err)
		return fmt.Errorf("внутренняя ошибка: innoextract не найден: %v", err)
	}

	// 2. Сохраняем на диск и ОБЯЗАТЕЛЬНО даем права на исполнение (0755)
	if err := os.WriteFile(innoBinPath, innoData, 0755); err != nil {
		LogError("ExtractInstaller: ошибка сохранения innoextract: %v", err)
		return fmt.Errorf("не удалось сохранить распаковщик: %v", err)
	}

	LogUnpacking("ExtractInstaller: нативная распаковка автономным innoextract, setup=%s, target=%s", installerPath, installPath)

	// 3. Запускаем именно наш извлеченный бинарник
	cmd := exec.Command(innoBinPath, "-d", installPath, installerPath)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		LogError("ExtractInstaller: ошибка запуска innoextract: %v", err)
		return fmt.Errorf("innoextract не запущен: %v", err)
	}
	LogUnpacking("innoextract запущен, PID=%d", cmd.Process.Pid)

	expectedSize := int64(77737979510)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

Loop:
	for {
		select {
		case err := <-done:
			if err != nil {
				LogError("ExtractInstaller: сбой распаковки innoextract: %v", err)
				LogError("ОШИБКА INNOEXTRACT:\n%s", errBuf.String())
				return fmt.Errorf("сбой распаковки innoextract: %v", err)
			}
			break Loop

		case <-ticker.C:
			if progressCb != nil {
				currentSize, _ := DirSize(installPath)
				percent := float64(currentSize) / float64(expectedSize)

				if percent > 0.99 {
					percent = 0.99
				}

				gbCurrent := float64(currentSize) / (1024 * 1024 * 1024)
				gbTotal := float64(expectedSize) / (1024 * 1024 * 1024)

				msg := fmt.Sprintf("Распаковка архивов: %.1f ГБ / %.1f ГБ", gbCurrent, gbTotal)
				progressCb(percent, msg)
			}
		}
	}

	if progressCb != nil {
		progressCb(0.99, "Перенос файлов и очистка мусора...")
	}

	appDir := filepath.Join(installPath, "app")
	entries, err := os.ReadDir(appDir)
	if err == nil {
		for _, entry := range entries {
			oldPath := filepath.Join(appDir, entry.Name())
			newPath := filepath.Join(installPath, entry.Name())
			os.Rename(oldPath, newPath)
		}
	}

	os.RemoveAll(appDir)
	os.RemoveAll(filepath.Join(installPath, "tmp"))

	LogInfo("Очистка файлов неиспользуемых графических модов (Текущий выбор: '%s')", graphicsMod)

	// Пробрасываем embeddedFiles дальше в cleanupGraphics
	if err := cleanupGraphics(installPath, cfg, graphicsMod, cacheDir, embeddedFiles, progressCb); err != nil {
		return err
	}

	if progressCb != nil {
		progressCb(1.0, "Базовая установка завершена!")
	}
	return nil
}

// Добавлен аргумент embeddedFiles типа embed.FS
func cleanupGraphics(installPath string, cfg *tui.InstallConfig, graphicsMod string, cacheDir string, embeddedFiles embed.FS, progressCb func(float64, string)) error {
	enbFiles := []string{
		"d3d11.dll",
		"enblocal.ini",
		"enbseries",
		"_weatherlist.ini",
		"_locationweather.ini",
	}

	reshadeFiles := []string{
		"dxgi.dll",
		"ReShade.ini",
		"reshade-shaders",
	}

	deleteFiles := func(files []string) {
		for _, f := range files {
			target := filepath.Join(installPath, f)
			if err := os.RemoveAll(target); err == nil {
				LogInfo("Удален: %s", f)
			}
		}
	}

	if graphicsMod == "ENB" {
		LogInfo("Оставлен ENB. Удаление файлов ReShade...")
		deleteFiles(reshadeFiles)
	} else if graphicsMod == "ReShade" {
		LogInfo("Оставлен ReShade. Удаление файлов ENB...")
		deleteFiles(enbFiles)
	} else if graphicsMod == "Community Shaders" {
		LogInfo("Выбран Community Shaders. Удаление файлов ENB и ReShade...")
		deleteFiles(enbFiles)
		deleteFiles(reshadeFiles)

		LogInfo("Начало скачивания и установки базы CS...")

		err := InstallCSBaseMods(installPath, cacheDir, progressCb)
		if err != nil {
			LogError("Критическая ошибка установки базы CS: %v", err)
			return fmt.Errorf("ошибка установки Community Shaders: %v", err)
		}

		LogInfo("Установка графического пресета Community Shaders...")
		if err := InstallCSPresetMods(installPath, cfg, cacheDir, progressCb); err != nil {
			LogError("Критическая ошибка установки пресета CS: %v", err)
			return fmt.Errorf("ошибка установки пресета CS: %v", err)
		}

		if progressCb != nil {
			progressCb(-1, "Создание базовых настроек Community Shaders...")
		}

		csSettingsDir := filepath.Join(installPath, "MO2", "overwrite", "SKSE", "Plugins", "CommunityShaders")
		if err := os.MkdirAll(csSettingsDir, 0755); err != nil {
			return fmt.Errorf("не удалось создать директорию для CS Settings: %v", err)
		}

		csSettingsDest := filepath.Join(csSettingsDir, "SettingsUser.json")

		// Используем проброшенный embeddedFiles
		settingsData, err := embeddedFiles.ReadFile("src/SettingsUser.json")
		if err == nil {
			if err := os.WriteFile(csSettingsDest, settingsData, 0644); err != nil {
				LogError("Ошибка сохранения SettingsUser.json: %v", err)
			} else {
				LogInfo("Базовый SettingsUser.json успешно извлечен.")
			}
		} else {
			LogError("Не удалось прочитать SettingsUser.json из embed: %v", err)
		}

	} else {
		LogInfo("Графические моды отключены. Удаление всех графических файлов...")
		deleteFiles(enbFiles)
		deleteFiles(reshadeFiles)
	}

	return nil
}
