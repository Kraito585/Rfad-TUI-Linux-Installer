package core

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
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

// progressReader для красивого статус-бара при скачивании
type progressReader struct {
	io.Reader
	Total   int64
	Current int64
	Report  func(float64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Total > 0 && pr.Report != nil {
		pr.Report(float64(pr.Current) / float64(pr.Total))
	}
	return n, err
}

// GetPortableWine скачивает и распаковывает изолированную версию Wine
func GetPortableWine(cacheDir, targetDir string, progressCb func(float64, string)) error {
	// Стабильная портативная сборка от Kron4ek
	wineURL := "https://github.com/Kron4ek/Wine-Builds/releases/download/11.13/wine-11.13-staging-x86.tar.xz"
	archivePath := filepath.Join(cacheDir, "wine-portable.tar.xz")

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		progressCb(0.0, "Скачивание Portable Wine (40 MB)...")

		resp, err := http.Get(wineURL)
		if err != nil {
			return fmt.Errorf("ошибка сети: %v", err)
		}
		defer resp.Body.Close()

		out, err := os.Create(archivePath)
		if err != nil {
			return err
		}
		defer out.Close()

		reader := &progressReader{
			Reader: resp.Body,
			Total:  resp.ContentLength,
			Report: func(p float64) {
				progressCb(p, fmt.Sprintf("Загрузка движка установки: %d%%", int(p*100)))
			},
		}

		if _, err := io.Copy(out, reader); err != nil {
			return fmt.Errorf("ошибка сохранения файла: %v", err)
		}
	}

	progressCb(-1, "Распаковка Portable Wine...")
	os.MkdirAll(targetDir, 0755)

	cmd := exec.Command("tar", "-xf", archivePath, "-C", targetDir, "--strip-components=1")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка tar распаковки: %v", err)
	}

	return nil
}

// ExtractInstaller запускает оригинальный setup.exe через Wine в тихом режиме
// Использование Innoextract невозможен.
// ps. если только авторы сборки не поделятся скриптами из установщика)))
func ExtractInstaller(wineExePath, installerPath, installPath string, infPath string, graphicsMod string, progressCb func(float64, string)) error {
	LogUnpacking("ExtractInstaller: запуск установки через Wine, setup=%s, target=%s", installerPath, installPath)

	// Да это фиксированые значения с Wine иначе не получится
	var expectedSize int64
	if graphicsMod == "ReShade" {
		// TODO: Замерить точный вес чистой установки с ReShade в байтах
		expectedSize = 85899345920 // Временное значение
	} else {
		// TODO: Замерить точный вес чистой установки с ENB в байтах
		expectedSize = 85899345920 // Временное значение
	}

	// ВЫЗЫВАЕМ НАШУ УМНУЮ ОБЕРТКУ
	cmd := exec.Command(
		wineExePath,
		installerPath,
		"/VERYSILENT",
		"/SUPPRESSMSGBOXES",
		"/NORESTART",
		fmt.Sprintf("/LOADINF=%s", infPath),
	)

	tempPrefix := filepath.Join(installPath, ".temp_wine_prefix")

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("WINEPREFIX=%s", tempPrefix),
		"WINEDLLOVERRIDES=winemenubuilder.exe=d;mscoree=d;mshtml=d",
		"WINE_DISABLE_MONO_PROMPT=1",
		"WINE_DISABLE_GECKO_PROMPT=1",
	)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	cmd.Stdout = &errBuf

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
				os.RemoveAll(tempPrefix)
				LogError("ExtractInstaller: сбой wine-установки: %v", err)
				LogError("ПРИЧИНА ПАДЕНИЯ WINE:\n%s", errBuf.String())
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

func CreateWineCommand(args ...string) *exec.Cmd {
	if _, err := exec.LookPath("wine"); err == nil {
		return exec.Command("wine", args...)
	}

	LogWarn("Системный wine не найден в PATH. Ищем скрытые сборки PortProton...")

	home, _ := os.UserHomeDir()

	// Оставлены только пути для нативной установки PortProton
	possibleRoots := []string{
		filepath.Join(home, ".local", "share", "PortProton", "data", "dist"),
		filepath.Join(home, "PortProton", "data", "dist"),
	}

	wineSuffixes := []string{
		filepath.Join("bin", "wine"),
		filepath.Join("wine", "bin", "wine"),
		filepath.Join("files", "bin", "wine"),
	}

	for _, root := range possibleRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			for _, suffix := range wineSuffixes {
				winePath := filepath.Join(root, entry.Name(), suffix)

				if _, err := os.Stat(winePath); err == nil {
					LogInfo("Найден скрытый Wine: %s", winePath)
					return exec.Command(winePath, args...)
				}
			}
		}
	}

	LogWarn("Скрытые версии Wine не найдены. Попытка стандартного вызова...")
	return exec.Command("wine", args...)
}
