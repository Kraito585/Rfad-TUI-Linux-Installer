package core

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

func CreateDesktopShortcuts(gamePath string, useSteamFix bool, assets embed.FS) error {
	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")
	menuDir := filepath.Join(home, ".local/share/applications")
	os.MkdirAll(menuDir, 0755)

	// 1. Извлекаем иконку для игры из бинарника и сохраняем рядом с игрой
	iconGame := filepath.Join(gamePath, "rfad-tui-launcher.ico")
	if icoData, err := assets.ReadFile("src/rfad-tui-launcher.ico"); err == nil {
		os.WriteFile(iconGame, icoData, 0644)
	}

	// 2. Иконка для MO2 (KDE Plasma сама достанет её из экзешника)
	iconMO2 := filepath.Join(gamePath, "MO2", "ModOrganizer.exe")

	// Добавляем поле args для передачи аргументов ВНЕ кавычек
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

		// Обрати внимание: %s "%s"%s -> аргумент подставляется после закрывающей кавычки
		content := fmt.Sprintf(`[Desktop Entry]
Name=%s
Exec=%s "%s"%s
Icon=%s
Type=Application
Categories=Game;
Terminal=false
`, s.name, execPrefix, s.exec, argsStr, s.icon)

		os.WriteFile(filepath.Join(desktopDir, s.filename), []byte(content), 0644)
		os.WriteFile(filepath.Join(menuDir, s.filename), []byte(content), 0644)
	}

	return nil
}
