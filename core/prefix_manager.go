package core

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GetPortProtonPrefixPath считывает конфиг PortProton и находит путь к префиксам
func GetPortProtonPrefixPath() (string, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config/PortProton/config")

	file, err := os.Open(configPath)
	if err != nil {
		// Если конфига нет, возвращаем стандарт, но лучше предупредить
		return filepath.Join(home, "PortProton/data/prefixes"), nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Ищем строку, где задан путь (обычно там формат KEY=VALUE)
		if strings.Contains(line, "PORTPROTON_HOME") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				// Путь к корню PortProton найден, возвращаем путь к префиксам
				return filepath.Join(parts[1], "data/prefixes"), nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ошибка при чтении конфига PortProton: %v", err)
	}

	return "", fmt.Errorf("не удалось найти путь в конфигурации PortProton")
}

// Извлечение префикса в дерикторию portproton
func ExtractPrefix(archivePath string, progressCb func(string)) error {
	LogUnpacking("ExtractPrefix: начало распаковки префикса из %s", archivePath)

	baseDir, err := GetPortProtonPrefixPath()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(baseDir, "RFAD_SE")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		LogError("ExtractPrefix: ошибка доступа к %s: %v", targetDir, err)
		return fmt.Errorf("ошибка доступа к директории префиксов: %v", err)
	}
	LogUnpacking("Целевая директория префикса: %s", targetDir)

	LogUnpacking("Открытие архива префикса: %s", archivePath)
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("архив %s не найден: %v", archivePath, err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("ошибка чтения gzip: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	LogUnpacking("Чтение tar-записей из архива префикса...")
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка чтения tar: %v", err)
		}

		parts := strings.Split(filepath.ToSlash(header.Name), "/")

		if len(parts) < 2 || parts[0] != "RFAD_SE" || parts[1] != "pfx" {
			continue
		}

		parts = parts[2:]

		if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
			continue
		}

		newName := strings.Join(parts, "/")
		targetPath := filepath.Join(targetDir, newName)

		if !filepath.HasPrefix(targetPath, filepath.Clean(targetDir)) {
			return fmt.Errorf("некорректный путь в архиве: %s", header.Name)
		}

		if progressCb != nil {
			progressCb(filepath.Base(newName))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tr)
			outFile.Close()
			if err != nil {
				return err
			}
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			_ = os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		}
	}

	LogUnpacking("Распаковка префикса завершена")
	return nil
}

// ExtractSteamPrefix распаковывает эталонную резервную копию всего префикса Steam
func ExtractSteamPrefix(archivePath string, installPath string, progressCb func(string)) error {
	targetDir := filepath.Join(installPath, "linux-fixes", "SteamPrefix")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", targetDir, err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		parts := strings.Split(filepath.ToSlash(header.Name), "/")

		if len(parts) > 0 && parts[0] == "RFAD_SE" {
			parts = parts[1:]
		}

		if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
			continue
		}

		targetPath := filepath.Join(targetDir, strings.Join(parts, "/"))

		if !filepath.HasPrefix(targetPath, filepath.Clean(targetDir)) {
			continue
		}

		if progressCb != nil {
			progressCb(filepath.Base(targetPath))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(targetPath, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			io.Copy(outFile, tr)
			outFile.Close()
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			os.Remove(targetPath)
			os.Symlink(header.Linkname, targetPath)
		}
	}

	return nil
}
