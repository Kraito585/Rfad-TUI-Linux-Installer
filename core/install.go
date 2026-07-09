package core

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func sanitize(s string) string {
	// Оставляем только буквы, цифры, точки, слеши и пробелы
	reg := regexp.MustCompile(`[^a-zA-Z0-9./\\ \-_]`)
	return reg.ReplaceAllString(s, "")
}

// ExtractInstaller запускает innoextract и читает его консольный вывод для TUI
func ExtractInstaller(installerPath, installPath string, progressCb func(float64, string)) error {
	cmd := exec.Command("innoextract", "-e", "-d", installPath, installerPath)

	// СОЗДАЕМ БУФЕР ДЛЯ ПЕРЕХВАТА ОШИБОК (STDERR)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ошибка создания пайпа: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ошибка запуска innoextract: %v", err)
	}

	scanner := bufio.NewScanner(stdout)

	var count float64
	var maxFiles float64 = 60000

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Extracting") || strings.HasPrefix(line, " - ") {
			count++
			percent := count / maxFiles

			if percent > 0.99 {
				percent = 0.99
			}

			if progressCb != nil {
				cleanName := sanitize(strings.TrimPrefix(line, "Extracting \""))

				if len(cleanName) > 40 {
					cleanName = "..." + cleanName[len(cleanName)-37:]
				}

				progressCb(percent, fmt.Sprintf("Распаковка: %s", cleanName))
			}
		}
	}

	// ПРОВЕРЯЕМ ОШИБКУ И ВЫВОДИМ ТО, ЧТО INNOEXTRACT НАПИСАЛ В СВОЮ ЗАЩИТУ
	if err := cmd.Wait(); err != nil {
		// Читаем текст ошибки из буфера
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		// Возвращаем детализированную ошибку
		return fmt.Errorf("сбой innoextract: %s", errMsg)
	}

	entries, _ := os.ReadDir(installPath)
	for _, entry := range entries {
		if entry.IsDir() {
			subDirPath := filepath.Join(installPath, entry.Name())
			subEntries, _ := os.ReadDir(subDirPath)

			for _, subEntry := range subEntries {
				src := filepath.Join(subDirPath, subEntry.Name())
				dst := filepath.Join(installPath, subEntry.Name())
				os.Rename(src, dst)
			}
			// 2. Удаляем папку (будь она app, sd или любая другая пустая)
			os.Remove(subDirPath)
		}
	}

	if progressCb != nil {
		progressCb(1.0, "Распаковка базовой игры завершена!")
	}

	return nil
}
