package tui

// InstallConfig хранит все настройки, собранные TUI опросником
type InstallConfig struct {
	InstallerPath string // Страница 1
	InstallPath   string // Страница 2

	GraphicsMod     string // Страница 3
	UseFSR          bool   // Страница 3
	FSRLevel        int    // Страница 3
	BaseWidth       int
	BaseHeight      int
	ResWidth        string // Страница 3
	ResHeight       string // Страница 3
	UseSteamFix     bool   // Страница 3
	CreateShortcuts bool   // Страница 3
}

// SystemChecks хранит результаты предполётных проверок
type SystemChecks struct {
	IsSudo       bool
	HasWine      bool
	HasGameMode  bool
	HasNVAPI     bool
	ScreenWidth  int
	ScreenHeight int
}

// NewInstallConfig возвращает конфиг с дефолтными значениями
func NewInstallConfig() *InstallConfig {
	return &InstallConfig{
		GraphicsMod:     "",
		UseFSR:          false,
		FSRLevel:        1,
		BaseWidth:       1920,
		BaseHeight:      1080,
		ResWidth:        "1920",
		ResHeight:       "1080",
		UseSteamFix:     false,
		CreateShortcuts: true,
	}
}
