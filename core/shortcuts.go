package core

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetDesktopDir() string {
	home, _ := os.UserHomeDir()

	cmd := exec.Command("xdg-user-dir", "DESKTOP")
	out, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(out))
		if path != "" && path != home {
			return path
		}
	}
	return filepath.Join(home, "Desktop")
}

func CreateDesktopShortcuts(gamePath string, useSteamFix bool, assets embed.FS) error {
	LogInfo("CreateDesktopShortcuts: создание ярлыков, gamePath=%s, steamFix=%v", gamePath, useSteamFix)
	desktopDir := GetDesktopDir()
	home, _ := os.UserHomeDir()
	menuDir := filepath.Join(home, ".local", "share", "applications")
	os.MkdirAll(menuDir, 0755)

	iconGame := filepath.Join(gamePath, "rfad-tui-launcher.ico")
	if icoData, err := assets.ReadFile("src/rfad-tui-launcher.ico"); err == nil {
		os.WriteFile(iconGame, icoData, 0644)
		LogInfo("Иконка сохранена: %s", iconGame)
	} else {
		LogWarn("Не удалось извлечь иконку из ресурсов: %v", err)
	}

	iconMO2 := filepath.Join(gamePath, "mod-organizer.ico")
	if icoData, err := assets.ReadFile("src/mod-organizer.ico"); err == nil {
		os.WriteFile(iconMO2, icoData, 0644)
		LogInfo("Иконка сохранена: %s", iconMO2)
	} else {
		LogWarn("Не удалось извлечь иконку из ресурсов: %v", err)
	}

	var shortcuts []struct {
		name     string
		exec     string
		args     string
		icon     string
		filename string
	}

	// Базовый ярлык MO2 (нужен всегда, работает через PortProton)
	mo2Exec := filepath.Join(gamePath, "MO2", "ModOrganizer.exe")

	if useSteamFix {
		// ЛИЦЕНЗИЯ
		LogInfo("Лицензионная версия: ярлык запуска игры не создается, так как запуск должен идти строго через Steam.")

		// Оставляем только ярлык для чистого MO2 (для настройки модов)
		shortcuts = append(shortcuts, struct {
			name, exec, args, icon, filename string
		}{"RFAD-MO2", mo2Exec, "", iconMO2, "rfad-mo2.desktop"})

	} else {
		gameExec := filepath.Join(gamePath, "MO2", "ModOrganizerSKSE.exe")

		shortcuts = append(shortcuts, struct {
			name, exec, args, icon, filename string
		}{"RFAD-Game", gameExec, "moshortcut://:SKSE", iconGame, "rfad-game.desktop"})

		shortcuts = append(shortcuts, struct {
			name, exec, args, icon, filename string
		}{"RFAD-MO2", mo2Exec, "", iconMO2, "rfad-mo2.desktop"})
	}
	execPrefix := "/usr/bin/portproton"

	for _, s := range shortcuts {
		argsStr := ""
		if s.args != "" {
			argsStr = " " + s.args
		}

		content := fmt.Sprintf(`[Desktop Entry]
Name=%s
Exec=%s "%s"%s
Icon=%s
Type=Application
Categories=Game;
Terminal=false
`, s.name, execPrefix, s.exec, argsStr, s.icon)

		desktopPath := filepath.Join(desktopDir, s.filename)
		menuPath := filepath.Join(menuDir, s.filename)
		os.WriteFile(desktopPath, []byte(content), 0644)
		os.WriteFile(menuPath, []byte(content), 0644)
		LogInfo("Ярлык создан: %s и %s", desktopPath, menuPath)
	}

	return nil
}
