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
	// Оставляем только буквы, цифры, точки, слеши и пробелы
	reg := regexp.MustCompile(`[^a-zA-Z0-9./\\ \-_]`)
	return reg.ReplaceAllString(s, "")
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Игнорируем файлы, к которым пока нет доступа
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// ExtractInstaller запускает оригинальный setup.exe через Wine в тихом режиме
func ExtractInstaller(installerPath, installPath string, infPath string, progressCb func(float64, string)) error {
	// ВАЖНО: 161 ГБ (в байтах) - примерный размер установленной сборки.
	// Отрегулируй это значение для большей точности ползунка.
	var expectedSize int64 = 161061273600

	// Запускаем инсталлятор через Wine (используем системный wine или portproton)
	// /VERYSILENT - без интерфейса, /SUPPRESSMSGBOXES - без окон с ошибками
	cmd := exec.Command("wine", installerPath, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", fmt.Sprintf("/LOADINF=%s", infPath))

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ошибка запуска setup.exe: %v", err)
	}

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
				return fmt.Errorf("сбой wine-установки: %v", err)
			}
			if progressCb != nil {
				progressCb(1.0, "Базовая установка завершена!")
			}
			return nil

		case <-ticker.C:
			if progressCb != nil {
				currentSize, _ := DirSize(installPath)
				percent := float64(currentSize) / float64(expectedSize)

				if percent > 0.99 {
					percent = 0.99 // Держим 99%, пока процесс не завершится
				}

				gbCurrent := float64(currentSize) / (1024 * 1024 * 1024)
				gbTotal := float64(expectedSize) / (1024 * 1024 * 1024)

				msg := fmt.Sprintf("Установка базовой игры: %.1f ГБ / %.1f ГБ", gbCurrent, gbTotal)
				progressCb(percent, msg)
			}
		}
	}
}
