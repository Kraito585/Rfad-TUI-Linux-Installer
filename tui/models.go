package tui

// InstallConfig хранит все настройки, собранные TUI опросником
type InstallConfig struct {
	InstallerPath string // Страница 1
	InstallPath   string // Страница 2

	GraphicsMod     string // Страница 3
	UseFSR          bool   // Страница 3
	ResWidth        string // Страница 3 (например, "2560")
	ResHeight       string // Страница 3 (например, "1440")
	UseSteamFix     bool   // Страница 3
	CreateShortcuts bool   // Страница 3
}

// NewInstallConfig возвращает конфиг с дефолтными значениями
func NewInstallConfig() *InstallConfig {
	return &InstallConfig{
		GraphicsMod:     "ENB",
		UseFSR:          false,
		ResWidth:        "1920",
		ResHeight:       "1080",
		UseSteamFix:     false,
		CreateShortcuts: true,
	}
}
