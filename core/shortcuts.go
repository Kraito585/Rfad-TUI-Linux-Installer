package core

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

func CreateDesktopShortcuts(gamePath string, useSteamFix bool, assets embed.FS) error {
	LogInfo("CreateDesktopShortcuts: создание ярлыков, gamePath=%s, steamFix=%v", gamePath, useSteamFix)
	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")
	menuDir := filepath.Join(home, ".local/share/applications")
	os.MkdirAll(menuDir, 0755)

	// 1. Извлекаем иконку для игры
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

	shortcuts := []struct {
		name     string
		exec     string
		args     string
		icon     string
		filename string
	}{
		{"RFAD-Game", filepath.Join(gamePath, "MO2", "ModOrganizerSKSE.exe"), "moshortcut://:SKSE", iconGame, "rfad-game.desktop"},
		{"RFAD-MO2", filepath.Join(gamePath, "MO2", "ModOrganizer.exe"), "", iconMO2, "rfad-mo2.desktop"},
	}

	execPrefix := "portproton"
	if useSteamFix {
		execPrefix = "env START_FROM_STEAM=1 portproton"
	}

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
