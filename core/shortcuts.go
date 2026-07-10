package core

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateDesktopShortcuts(gamePath string, useSteamFix bool) error {
	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")
	menuDir := filepath.Join(home, ".local/share/applications")
	os.MkdirAll(menuDir, 0755)

	iconGame := filepath.Join(gamePath, "skse64_loader.exe")
	iconMO2 := filepath.Join(gamePath, "rfad-tui-launcher.ico")

	shortcuts := []struct {
		name     string
		exec     string
		icon     string
		filename string
	}{
		{"RFAD-Game", filepath.Join(gamePath, "MO2", "ModOrganizerSKSE.exe") + " moshortcut://:SKSE", iconGame, "rfad-game.desktop"},
		{"RFAD-MO2", filepath.Join(gamePath, "MO2", "ModOrganizer.exe"), iconMO2, "rfad-mo2.desktop"},
	}

	// Магия Steam Fix: если включено, добавляем переменную окружения в команду запуска
	execPrefix := "portproton"
	if useSteamFix {
		execPrefix = "env START_FROM_STEAM=1 portproton"
	}

	for _, s := range shortcuts {
		// Обрати внимание на %s "%s" — путь к exe теперь в кавычках
		content := fmt.Sprintf(`[Desktop Entry]
Name=%s
Exec=%s "%s"
Icon=%s
Type=Application
Categories=Game;
Terminal=false
`, s.name, execPrefix, s.exec, s.icon)

		os.WriteFile(filepath.Join(desktopDir, s.filename), []byte(content), 0644)
		os.WriteFile(filepath.Join(menuDir, s.filename), []byte(content), 0644)
	}

	return nil
}
