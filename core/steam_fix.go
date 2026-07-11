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

// =======================================================================
//                      Отправлена до лучших времён
//    когда вобще пойму как steam fix запустить на linux тогда и верну

// █   █  ███  █████     ████ █   █ ████   ███  ████  █████ █████ ████
// ██  █░█ ░░█  ░█░░░   █ ░░░░█░  █░█░░░█ █ ░░█ █░░░█  ░█░░░█░░░░░█░░░█
// █░█ █░█░ ░█░  █░░░░   ███░░█░░ █░████░░█░ ░█░████░░  █░░░████░░█░░░█░
// █░░██░█░░ █░░ █░░      ░░█ █░░ █░█░░░░ █░░ █░█░░█░ ░ █░░ █░░░░ █░░ █░░
// █░░ █░░███ ░░ █░░    ████░░ ███ ░█░░░░░ ███ ░█░░░█░  █░░ █████░████ ░░
//  ░░  ░░ ░░░ ░  ░░     ░░░░ ░ ░░░ ░░░     ░░░ ░░░  ░   ░░  ░░░░░ ░░░░ ░
//   ░   ░  ░░░    ░      ░░░░   ░░░  ░      ░░░  ░   ░   ░   ░░░░░ ░░░░

// =======================================================================
func ApplySteamFix(gamePath string) error {
	LogUnpacking("Начало Steam Fix: создание бэкапа оригиналов в %s", gamePath)

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
			LogError("Steam Fix: не найден файл для бэкапа: %s", name)
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
	LogUnpacking("Бэкап создан: disable_stiam_fix.tar.gz")

	LogUnpacking("Распаковка steam_fix.tar.gz в %s", gamePath)
	fixFile, err := os.Open("src/steam_fix.tar.gz")
	if err != nil {
		return fmt.Errorf("не найден архив steam_fix.tar.gz в /src: %v", err)
	}
	defer fixFile.Close()

	cmd := exec.Command("tar", "-xzf", "src/steam_fix.tar.gz", "-C", gamePath)
	if err := cmd.Run(); err != nil {
		LogError("Steam Fix: ошибка распаковки: %v", err)
		return fmt.Errorf("ошибка распаковки steam_fix: %v", err)
	}

	LogUnpacking("Steam Fix: распаковка завершена успешно")
	return nil
}
