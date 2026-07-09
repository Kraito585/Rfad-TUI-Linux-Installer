package core

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"embed"
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

// ExtractPrefix распаковывает tar.gz архив с префиксом в директорию PortProton
func ExtractPrefix(bundledAssets embed.FS, progressCb func(string)) error {
	targetDir, err := GetPortProtonPrefixPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("ошибка доступа к директории префиксов: %v", err)
	}

	f, err := bundledAssets.Open("src/prefix.tar.gz")
	if err != nil {
		return fmt.Errorf("архив prefix.tar.gz не найден в ресурсах: %v", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("ошибка чтения gzip: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ошибка чтения tar: %v", err)
		}

		// Ключевое изменение:
		// Мы используем header.Name как есть, чтобы сохранить структуру папок из архива.
		// Если в архиве лежит "RFAD_SE/drive_c/...", то при Join с targetDir
		// мы получим ровно то, что нужно.
		targetPath := filepath.Join(targetDir, header.Name)

		// Защита от выхода за пределы
		if !filepath.HasPrefix(targetPath, filepath.Clean(targetDir)) {
			return fmt.Errorf("некорректный путь в архиве: %s", header.Name)
		}

		if progressCb != nil {
			progressCb(filepath.Base(header.Name))
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Создаем директорию, если её еще нет.
			// MkdirAll здесь безопасен: если RFAD_SE уже есть, он просто пройдет дальше.
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// Создаем родительскую директорию для файла (например, если файл лежит внутри папок)
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
			// Удаляем существующий путь, если это симлинк, чтобы перезаписать его
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			_ = os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		}
	}

	return nil
}
