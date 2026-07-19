package core

import (
	"bytes"
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

func ExtractInstaller(installerPath, installPath string, graphicsMod string, progressCb func(float64, string)) error {
	LogUnpacking("ExtractInstaller: нативная распаковка innoextract, setup=%s, target=%s", installerPath, installPath)

	cmd := exec.Command("innoextract", "-d", installPath, installerPath)

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
	cleanupGraphics(installPath, graphicsMod)

	if progressCb != nil {
		progressCb(1.0, "Базовая установка завершена!")
	}
	return nil
}

func cleanupGraphics(installPath string, graphicsMod string) {
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
	} else {
		LogInfo("Графические моды отключены. Удаление всех графических файлов...")
		deleteFiles(enbFiles)
		deleteFiles(reshadeFiles)
	}
}
