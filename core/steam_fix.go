package core

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func ApplySteamFix(gamePath string) error {
	// 1. ПРОВЕРКА ПУТИ (Подняли в начало)
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("не удалось найти домашнюю директорию: %v", err)
	}
	steamCommon := filepath.Join(home, ".local/share/Steam/steamapps/common")

	if _, err := os.Stat(steamCommon); os.IsNotExist(err) {
		return fmt.Errorf("Steam не найден по пути: %s. Убедитесь, что Steam установлен", steamCommon)
	}

	// 2. Создаем архив бэкапа оригиналов (disable_stiam_fix.tar.gz)
	backupFile, err := os.Create(filepath.Join(gamePath, "disable_stiam_fix.tar.gz"))
	if err != nil {
		return err
	}
	defer backupFile.Close()

	gw := gzip.NewWriter(backupFile)
	tw := tar.NewWriter(gw)

	filesToBackup := []string{"SkyrimSE.exe", "steam_api64.dll"}
	for _, name := range filesToBackup {
		f, err := os.Open(filepath.Join(gamePath, name))
		if err != nil {
			tw.Close()
			gw.Close()
			backupFile.Close()
			return fmt.Errorf("не найден файл для бэкапа: %s", name)
		}

		stat, _ := f.Stat()
		header, _ := tar.FileInfoHeader(stat, "")
		header.Name = name
		tw.WriteHeader(header)
		io.Copy(tw, f)
		f.Close()
	}
	tw.Close()
	gw.Close()

	// 3. Распаковываем steam_fix.tar.gz (предполагаем, что он в /src/)
	// Используем os.Open для доступа к файлу фикса
	fixFile, err := os.Open("src/steam_fix.tar.gz")
	if err != nil {
		return fmt.Errorf("не найден архив steam_fix.tar.gz в /src: %v", err)
	}
	defer fixFile.Close()

	// (Здесь логика распаковки tar.gz поверх gamePath)
	// Для простоты реализации можно использовать os/exec команду "tar -xzf ..."
	// Это стандарт для Linux-систем и максимально надежно
	cmd := exec.Command("tar", "-xzf", "src/steam_fix.tar.gz", "-C", gamePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка распаковки steam_fix: %v", err)
	}

	// 4. Создаем симлинк
	targetLink := filepath.Join(steamCommon, "RFAD_SE")
	os.Remove(targetLink) // Удаляем старую ссылку если есть
	return os.Symlink(gamePath, targetLink)
}
