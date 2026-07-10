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
	// 1. Создаем архив бэкапа оригиналов (disable_stiam_fix.tar.gz)
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
		_, err = io.Copy(tw, f)
		f.Close()

		if err != nil {
			return fmt.Errorf("ошибка при записи бэкапа %s: %v", name, err)
		}
	}
	tw.Close()
	gw.Close()

	// 2. Распаковываем steam_fix.tar.gz (предполагаем, что он в /src/)
	fixFile, err := os.Open("src/steam_fix.tar.gz")
	if err != nil {
		return fmt.Errorf("не найден архив steam_fix.tar.gz в /src: %v", err)
	}
	defer fixFile.Close()

	// Распаковываем фикс поверх оригиналов
	cmd := exec.Command("tar", "-xzf", "src/steam_fix.tar.gz", "-C", gamePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка распаковки steam_fix: %v", err)
	}

	return nil
}
