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
	"time"
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

// GetSteamCompatDataDir ищет папку compatdata для нужного AppID во всех возможных местах
func GetSteamCompatDataDir(appID string) string {
	home, _ := os.UserHomeDir()

	possiblePaths := []string{
		filepath.Join(home, ".steam", "steam", "steamapps", "compatdata", appID),
		filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata", appID),

		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam", "steamapps", "compatdata", appID),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".steam", "steam", "steamapps", "compatdata", appID),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	flatpakSteamRoot := filepath.Join(home, ".var", "app", "com.valvesoftware.Steam")
	if _, err := os.Stat(flatpakSteamRoot); err == nil {
		return possiblePaths[2]
	}

	return possiblePaths[0]
}

// AddToSteamShortcuts находит shortcuts.vdf для ВСЕХ пользователей и дописывает туда игру
func AddToSteamShortcuts(appName, exePath, startDir, launchOptions string) error {
	home, _ := os.UserHomeDir()

	possibleUserdataPaths := []string{
		filepath.Join(home, ".steam", "steam", "userdata"),
		filepath.Join(home, ".local", "share", "Steam", "userdata"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam", "userdata"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".steam", "steam", "userdata"),
	}

	var userdataDir string
	found := false

	for _, path := range possibleUserdataPaths {
		if _, err := os.Stat(path); err == nil {
			userdataDir = path
			found = true
			LogInfo("AddToSteamShortcuts: найдена папка userdata: %s", path)
			break
		}
	}

	if !found {
		return fmt.Errorf("папка userdata Steam не найдена ни в одном из стандартных путей")
	}
	entries, err := os.ReadDir(userdataDir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("папка userdata Steam не найдена")
	}

	successCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if name == "0" || name == "ac" || name == "anonymous" {
			continue
		}

		vdfPath := filepath.Join(userdataDir, name, "config", "shortcuts.vdf")
		LogInfo("AddToSteamShortcuts: обработка профиля %s (%s)", name, vdfPath)

		var fileData []byte
		var shortcutCount int

		if _, err := os.Stat(vdfPath); err == nil {
			fileData, _ = os.ReadFile(vdfPath)
			if len(fileData) > 2 {
				fileData = fileData[:len(fileData)-2]
			}
			shortcutCount = bytes.Count(fileData, []byte("\x00AppName\x00"))
		} else {
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

		buf.WriteByte(0x00)
		buf.WriteString("tags")
		buf.WriteByte(0x00)

		buf.Write([]byte{0x08, 0x08, 0x08, 0x08})

		finalData := append(fileData, buf.Bytes()...)

		os.MkdirAll(filepath.Dir(vdfPath), 0755)

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

		if err := os.Remove(filePath); err != nil {
			LogWarn("Steam Fix: не удалось удалить оригинальный файл %s: %v", name, err)
		} else {
			LogInfo("Steam Fix: удален старый файл %s", name)
		}
	}
	tw.Close()
	gw.Close()
	LogUnpacking("Бэкап создан: disable_stiam_fix.tar.gz")

	LogUnpacking("Извлечение steam_fix.tar.gz из бинарника...")

	fixData, err := assets.ReadFile("src/steam_fix.tar.gz")
	if err != nil {
		return fmt.Errorf("не удалось прочитать steam_fix.tar.gz из ресурсов: %v", err)
	}

	tmpArchivePath := filepath.Join(os.TempDir(), "rfad_steam_fix_temp.tar.gz")
	if err := os.WriteFile(tmpArchivePath, fixData, 0644); err != nil {
		return fmt.Errorf("ошибка записи временного файла фикса: %v", err)
	}
	defer os.Remove(tmpArchivePath)

	LogUnpacking("Распаковка steam_fix.tar.gz в %s", gamePath)

	cmd := exec.Command("tar", "-xzf", tmpArchivePath, "-C", gamePath)
	if err := cmd.Run(); err != nil {
		LogError("Steam Fix: ошибка распаковки: %v", err)
		return fmt.Errorf("ошибка распаковки steam_fix: %v", err)
	}

	LogUnpacking("Steam Fix: распаковка завершена успешно")
	return nil
}

// При первом запуске скрипт запускает MO2 через PortProton чтобы открыть диалговое
// окно догрузки пакетов, в последующих запусках патчит prefix из PortProton в Steam
// упрощает доработку префикса и уменьшает занимаемое место на диске 4+Gb
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

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err == nil {
		LogInfo("CreateLaunchScript: скрипт запуска успешно создан по пути %s", scriptPath)
	}
	return scriptPath, err
}

func CreateSteamPrelaunchScript(installPath string) (string, error) {
	scriptPath := filepath.Join(installPath, "steam_prelaunch.sh")

	scriptContent := fmt.Sprintf(`#!/usr/bin/env bash

INSTALL_DIR="%s"
MO2_EXE="$INSTALL_DIR/MO2/ModOrganizer.exe"

# Пути префиксов
PP_PREFIX="$HOME/PortProton/data/prefixes/RFAD_SE"
STEAM_COMPAT_DIR="$HOME/.steam/steam/steamapps/compatdata/489830"
STEAM_PFX="$STEAM_COMPAT_DIR/pfx"
MARKER_FILE="$INSTALL_DIR/.pp_initialized"

# 1. ПРОВЕРКА ПЕРВОГО ЗАПУСКА (Инициализация через PortProton)
if [ ! -f "$MARKER_FILE" ]; then
    echo "Первый запуск: инициализация префикса через PortProton..."
    
    # Запускаем MO2 через PortProton. 
    # Он скачает Mono/Gecko и закроется (из-за SteamStub или bwrap), что нам и нужно!
    /usr/bin/portproton "$MO2_EXE"
    
    # Ставим флаг, чтобы больше не вызывать PortProton
    touch "$MARKER_FILE"
    echo "Инициализация пакетов завершена."
fi

# 2. ПОДМЕНА ПРЕФИКСА НА ВРЕМЯ ИГРЫ
mkdir -p "$STEAM_COMPAT_DIR"

# Делаем бэкап оригинального префикса Steam (если он есть и это не наш симлинк)
if [ -d "$STEAM_PFX" ] && [ ! -L "$STEAM_PFX" ]; then
    mv "$STEAM_PFX" "${STEAM_PFX}_backup"
fi

# Прокидываем префикс PortProton в Steam
if [ ! -L "$STEAM_PFX" ]; then
    ln -s "$PP_PREFIX" "$STEAM_PFX"
fi

# 3. ЗАПУСК ИГРЫ ЧЕРЕЗ STEAM PROTON
# Символ $@ выполнит то, что передал Steam (proton run ModOrganizer.exe ...)
"$@"

# Указываем версию, которую скачала твоя функция InstallGEProton в Go
GE_VERSION="GE-Proton11-1"
CUSTOM_PROTON="$HOME/.steam/root/compatibilitytools.d/$GE_VERSION/proton"

# 4. ЗАПУСК ИГРЫ ЧЕРЕЗ СВОЙ PROTON
if [ -f "$CUSTOM_PROTON" ]; then
    echo "Запуск через кастомный Proton: $GE_VERSION"
    # Команда shift удаляет первый аргумент из $@ (путь к системному Proton, который передал Steam)
    shift
    
    # Запускаем наш GE-Proton, передавая ему оставшиеся аргументы (run ModOrganizer.exe ...)
    "$CUSTOM_PROTON" "$@"
else
    echo "GE-Proton не найден, откат на системный Proton"
    "$@"
fi

# 5. ОТКАТ ПРЕФИКСА ПОСЛЕ ЗАКРЫТИЯ ИГРЫ
rm -f "$STEAM_PFX"
if [ -d "${STEAM_PFX}_backup" ]; then
    mv "${STEAM_PFX}_backup" "$STEAM_PFX"
fi

`, installPath)

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755) // Делаем исполняемым
	if err == nil {
		LogInfo("CreateSteamPrelaunchScript: скрипт-обертка создан по пути %s", scriptPath)
	}
	return scriptPath, err
}

// CreateMO2PythonPlugin создает плагин для MO2, который генерирует маркер при запуске
func CreateMO2PythonPlugin(installPath string) error {
	pluginsDir := filepath.Join(installPath, "MO2", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}

	pluginPath := filepath.Join(pluginsDir, "rfad_linux_init.py")

	pyCode := `import mobase
import os

class RFADLinuxInit(mobase.IPlugin):
    def init(self, organizer: mobase.IOrganizer) -> bool:
        try:
            # __file__ указывает на MO2/plugins/rfad_linux_init.py
            # Поднимаемся на 2 уровня выше, чтобы попасть в корень игры
            marker_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".rfad_initialized"))
            
            # Создаем пустой файл-маркер (или обновляем дату изменения, если он есть)
            with open(marker_path, 'a'):
                pass
        except Exception:
            pass
        return True

    def name(self) -> str: return "RFAD Linux Init Marker"
    def author(self) -> str: return "RFAD Installer"
    def description(self) -> str: return "Generates initialization marker for Linux PortProton integration"
    def version(self) -> mobase.VersionInfo: return mobase.VersionInfo(1,0,0,mobase.ReleaseType.FINAL)
    def isActive(self) -> bool: return True
    def settings(self): return []

def createPlugin() -> mobase.IPlugin:
    return RFADLinuxInit()
`

	err := os.WriteFile(pluginPath, []byte(pyCode), 0644)
	if err == nil {
		LogInfo("Python-плагин для MO2 успешно внедрен: %s", pluginPath)
	} else {
		LogError("Ошибка создания Python-плагина: %v", err)
	}
	return err
}

func ShutdownSteam() {
	LogInfo("AddToSteamShortcuts: отправка команды завершения работы Steam...")

	exec.Command("steam", "-shutdown").Run()

	for i := 0; i < 5; i++ {
		if _, err := exec.Command("pidof", "steam").Output(); err != nil {
			LogInfo("AddToSteamShortcuts: Steam успешно закрыт.")
			return
		}
		time.Sleep(1 * time.Second)
	}
	LogWarn("AddToSteamShortcuts: Steam не закрылся вовремя, работаем на свой страх и риск.")
}
