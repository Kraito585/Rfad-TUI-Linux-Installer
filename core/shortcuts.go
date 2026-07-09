package core

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateDesktopShortcuts(gamePath string) error {
	home, _ := os.UserHomeDir()
	desktopDir := filepath.Join(home, "Desktop")
	menuDir := filepath.Join(home, ".local/share/applications")
	os.MkdirAll(menuDir, 0755)

	// Разные иконки для визуальной навигации
	iconGame := filepath.Join(gamePath, "skse64_loader.exe") // Берем иконку из самого SKSE
	iconMO2 := filepath.Join(gamePath, "rfad-tui-launcher.ico")

	shortcuts := []struct {
		name     string
		exec     string
		icon     string
		filename string
	}{
		{"RFAD-Game", filepath.Join(gamePath, "ModOrganizerSKSE.exe"), iconGame, "rfad-game.desktop"},
		{"RFAD-MO2", filepath.Join(gamePath, "ModOrganizer.exe"), iconMO2, "rfad-mo2.desktop"},
	}

	for _, s := range shortcuts {
		content := fmt.Sprintf(`[Desktop Entry]
Name=%s
Exec=portproton %s
Icon=%s
Type=Application
Categories=Game;
Terminal=false
`, s.name, s.exec, s.icon, s.filename)

		os.WriteFile(filepath.Join(desktopDir, s.filename), []byte(content), 0644)
		os.WriteFile(filepath.Join(menuDir, s.filename), []byte(content), 0644)
	}

	return nil
}
