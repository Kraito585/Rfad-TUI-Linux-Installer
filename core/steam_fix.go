package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func GenerateSteamAppID(appName, exePath string) uint32 {
	input := appName + exePath
	hash := md5.Sum([]byte(input))
	seedHex := hex.EncodeToString(hash[:])[:8]
	seed, _ := strconv.ParseInt(seedHex, 16, 64)
	signedID := -(seed % 1000000000)
	return uint32(signedID)
}

func WriteVDFString(buf *bytes.Buffer, fieldType byte, key, value string) {
	buf.WriteByte(fieldType)
	buf.WriteString(key)
	buf.WriteByte(0x00)
	buf.WriteString(value)
	buf.WriteByte(0x00)
}

func WriteVDFInt32(buf *bytes.Buffer, fieldType byte, key string, value uint32) {
	buf.WriteByte(fieldType)
	buf.WriteString(key)
	buf.WriteByte(0x00)
	binary.Write(buf, binary.LittleEndian, value)
}

// AddToSteamShortcuts находит shortcuts.vdf для ВСЕХ пользователей и дописывает туда игру
func AddToSteamShortcuts(appName, exePath, startDir, launchOptions string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	userdataDir := filepath.Join(home, ".steam/steam/userdata")
	entries, err := os.ReadDir(userdataDir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("папка userdata Steam не найдена")
	}

	successCount := 0

	// Проходим по всем папкам в userdata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Игнорируем системные папки-заглушки
		if name == "0" || name == "ac" || name == "anonymous" {
			continue
		}

		vdfPath := filepath.Join(userdataDir, name, "config", "shortcuts.vdf")
		LogInfo("AddToSteamShortcuts: обработка профиля %s (%s)", name, vdfPath)

		var fileData []byte
		var shortcutCount int

		if _, err := os.Stat(vdfPath); err == nil {
			// Читаем существующий файл
			fileData, _ = os.ReadFile(vdfPath)
			// Отрезаем 2 последних байта (0x08 0x08), закрывающих файл, чтобы вписать новые данные
			if len(fileData) > 2 {
				fileData = fileData[:len(fileData)-2]
			}
			shortcutCount = bytes.Count(fileData, []byte("\x00AppName\x00"))
		} else {
			// Создаем новый файл, если его нет
			fileData = append(fileData, 0x00)
			fileData = append(fileData, []byte("shortcuts")...)
			fileData = append(fileData, 0x00)
			shortcutCount = 0
		}

		appID := GenerateSteamAppID(appName, exePath)

		buf := new(bytes.Buffer)
		buf.WriteByte(0x00)
		buf.WriteString(strconv.Itoa(shortcutCount))
		buf.WriteByte(0x00)

		WriteVDFInt32(buf, 0x02, "appid", appID)
		WriteVDFString(buf, 0x01, "AppName", appName)
		WriteVDFString(buf, 0x01, "Exe", fmt.Sprintf("\"%s\"", exePath))
		WriteVDFString(buf, 0x01, "StartDir", fmt.Sprintf("\"%s\"", startDir))
		WriteVDFString(buf, 0x01, "icon", "")
		WriteVDFString(buf, 0x01, "ShortcutPath", "")
		WriteVDFString(buf, 0x01, "LaunchOptions", launchOptions)

		WriteVDFInt32(buf, 0x02, "IsHidden", 0)
		WriteVDFInt32(buf, 0x02, "AllowDesktopConfig", 1)
		WriteVDFInt32(buf, 0x02, "AllowOverlay", 1)
		WriteVDFInt32(buf, 0x02, "OpenVR", 0)
		WriteVDFInt32(buf, 0x02, "Devkit", 0)
		WriteVDFString(buf, 0x01, "DevkitGameID", "")
		WriteVDFInt32(buf, 0x02, "DevkitOverrideAppID", 0)
		WriteVDFInt32(buf, 0x02, "LastPlayTime", 0)
		WriteVDFString(buf, 0x01, "FlatpakAppID", "")

		// ПРАВИЛЬНАЯ ЗАПИСЬ ТЕГОВ И ЗАКРЫТИЕ ФАЙЛА
		// 0x00 объявляет начало словаря "tags"
		buf.WriteByte(0x00)
		buf.WriteString("tags")
		buf.WriteByte(0x00)

		// Закрываем 4 словаря: tags -> текущий ярлык -> shortcuts -> весь файл
		buf.Write([]byte{0x08, 0x08, 0x08, 0x08})

		finalData := append(fileData, buf.Bytes()...)

		// Создаем директорию config, если её вдруг нет
		os.MkdirAll(filepath.Dir(vdfPath), 0755)

		// Записываем файл
		if err := os.WriteFile(vdfPath, finalData, 0644); err == nil {
			LogInfo("AddToSteamShortcuts: ярлык успешно добавлен для пользователя %s", name)
			successCount++
		} else {
			LogError("AddToSteamShortcuts: ошибка записи для пользователя %s: %v", name, err)
		}
	}

	if successCount == 0 {
		return fmt.Errorf("не удалось добавить ярлык ни одному реальному пользователю Steam")
	}

	return nil
}

func RestartSteam() {
	LogInfo("RestartSteam: отправка команды на перезапуск клиента Steam...")
	exec.Command("pkill", "steam").Run()
	exec.Command("sleep", "2").Run()
	exec.Command("nohup", "steam").Start()
}

func ApplySteamFix(gamePath string, assets embed.FS) error {
	LogUnpacking("Начало Steam Fix: создание бэкапа оригиналов в %s", gamePath)

	backupFile, err := os.Create(filepath.Join(gamePath, "disable_stiam_fix.tar.gz"))
	if err != nil {
		return err
	}
	defer backupFile.Close()

	gw := gzip.NewWriter(backupFile)
	tw := tar.NewWriter(gw)

	filesToBackup := []string{"SkyrimSE.exe", "steam_api64.dll", "steam_api64.cdx"}

	for _, name := range filesToBackup {
		filePath := filepath.Join(gamePath, name)
		f, err := os.Open(filePath)
		if err != nil {
			// steam_api64.cdx может и не быть в некоторых версиях, это нормально
			if name == "steam_api64.cdx" {
				continue
			}

			tw.Close()
			gw.Close()
			backupFile.Close()
			LogError("Steam Fix: не найден обязательный файл для бэкапа: %s", name)
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

		// Удаляем старый файл после бэкапа, чтобы очистить место для нашего фикса
		if err := os.Remove(filePath); err != nil {
			LogWarn("Steam Fix: не удалось удалить оригинальный файл %s: %v", name, err)
		} else {
			LogInfo("Steam Fix: удален старый файл %s", name)
		}
	}
	tw.Close()
	gw.Close()
	LogUnpacking("Бэкап создан: disable_stiam_fix.tar.gz")

	// === ИСПРАВЛЕНИЕ РАБОТЫ С EMBED.FS ===
	LogUnpacking("Извлечение steam_fix.tar.gz из бинарника...")

	// Читаем архив из памяти
	fixData, err := assets.ReadFile("src/steam_fix.tar.gz")
	if err != nil {
		return fmt.Errorf("не удалось прочитать steam_fix.tar.gz из ресурсов: %v", err)
	}

	// Создаем временный файл во временной папке ОС
	tmpArchivePath := filepath.Join(os.TempDir(), "rfad_steam_fix_temp.tar.gz")
	if err := os.WriteFile(tmpArchivePath, fixData, 0644); err != nil {
		return fmt.Errorf("ошибка записи временного файла фикса: %v", err)
	}
	// Обязательно удаляем его после распаковки, чтобы не мусорить на диске
	defer os.Remove(tmpArchivePath)

	LogUnpacking("Распаковка steam_fix.tar.gz в %s", gamePath)

	// Натравливаем tar на наш временный файл
	cmd := exec.Command("tar", "-xzf", tmpArchivePath, "-C", gamePath)
	if err := cmd.Run(); err != nil {
		LogError("Steam Fix: ошибка распаковки: %v", err)
		return fmt.Errorf("ошибка распаковки steam_fix: %v", err)
	}

	LogUnpacking("Steam Fix: распаковка завершена успешно")
	return nil
}

// CreateLaunchScript генерирует .sh скрипт для запуска игры через PortProton с сохранением Steam DRM
func CreateLaunchScript(installPath string) (string, error) {
	scriptPath := filepath.Join(installPath, "start_rfad.sh")
	mo2Exe := filepath.Join(installPath, "MO2", "ModOrganizerSKSE.exe")

	// Наш кастомный скрипт
	scriptContent := fmt.Sprintf(`#!/usr/bin/env bash

export STEAM_APP_ID="489830"
export SteamAppId="489830"

export START_FROM_STEAM=1
export STEAM_COMPAT_CLIENT_INSTALL_PATH="$HOME/.steam/steam"

/usr/bin/portproton "%s"
`, mo2Exe)

	// Флаг 0755 делает файл исполняемым (chmod +x)
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err == nil {
		LogInfo("CreateLaunchScript: скрипт запуска успешно создан по пути %s", scriptPath)
	}
	return scriptPath, err
}
