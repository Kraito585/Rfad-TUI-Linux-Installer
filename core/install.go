package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// ExtractInstaller запускает оригинальный setup.exe через Wine в тихом режиме
func ExtractInstaller(installerPath, installPath string, infPath string, graphicsMod string, progressCb func(float64, string)) error {
	LogUnpacking("ExtractInstaller: запуск установки через Wine, setup=%s, target=%s", installerPath, installPath)

	var expectedSize int64
	if graphicsMod == "ReShade" {
		// TODO: Замерить точный вес чистой установки с ReShade в байтах
		expectedSize = 85899345920 // Временное значение
	} else {
		// TODO: Замерить точный вес чистой установки с ENB в байтах
		expectedSize = 85899345920 // Временное значение
	}

	cmd := exec.Command("wine", installerPath, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", fmt.Sprintf("/LOADINF=%s", infPath))
	cmd.Env = append(os.Environ(), "WINEDLLOVERRIDES=winemenubuilder.exe=d")

	if err := cmd.Start(); err != nil {
		LogError("ExtractInstaller: ошибка запуска setup.exe: %v", err)
		return fmt.Errorf("ошибка запуска setup.exe: %v", err)
	}
	LogUnpacking("Wine-установщик запущен, PID=%d", cmd.Process.Pid)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			if err != nil {
				LogError("ExtractInstaller: сбой wine-установки: %v", err)
				return fmt.Errorf("сбой wine-установки: %v", err)
			}
			LogUnpacking("Wine-установка завершена успешно")
			if progressCb != nil {
				progressCb(1.0, "Базовая установка завершена!")
			}
			return nil

		case <-ticker.C:
			if progressCb != nil {
				currentSize, _ := DirSize(installPath)
				percent := float64(currentSize) / float64(expectedSize)

				if percent > 0.99 {
					percent = 0.99
				}

				gbCurrent := float64(currentSize) / (1024 * 1024 * 1024)
				gbTotal := float64(expectedSize) / (1024 * 1024 * 1024)

				msg := fmt.Sprintf("Установка базовой игры: %.1f ГБ / %.1f ГБ", gbCurrent, gbTotal)
				progressCb(percent, msg)
			}
		}
	}
}
