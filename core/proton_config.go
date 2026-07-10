package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GeneratePPDB(gamePath string, targetExe string, useFSR, useNVAPI, useGameMode, useSteamFix bool) error {
	cleanName := strings.Replace(targetExe, ".exe", "", 1)
	ppdbName := cleanName + ".ppdb"

	configPath := filepath.Join(gamePath, "MO2", ppdbName)

	envVars := []string{
		"WINEDLLOVERRIDES=\"xaudio2_7=n,b;d3d11=n,b;d3dx9_42=n,b;d3dcompiler_47=n,b;dinput8=n,b;mscoree=n\"",
	}

	// Если включен Steam Fix, прописываем переменную в конфиг PortProton
	if useSteamFix {
		envVars = append(envVars, "START_FROM_STEAM=1")
	}

	if useFSR {
		envVars = append(envVars, "WINE_FULLSCREEN_FSR=1", "WINE_FULLSCREEN_FSR_STRENGTH=2")
	}

	if useFSR {
		envVars = append(envVars, "WINE_FULLSCREEN_FSR=1", "WINE_FULLSCREEN_FSR_STRENGTH=2")
	}
	if useNVAPI {
		envVars = append(envVars, "PROTON_ENABLE_NVAPI=1")
	}

	command := "%command%"
	if useGameMode {
		command = "gamemoderun %command%"
	}

	args := ""
	if strings.Contains(targetExe, "SKSE") {
		args = "moshortcut://:SKSE"
	}

	// Явно указываем PREFIX_NAME=RFAD_SE, чтобы не запускался DEFAULT
	content := fmt.Sprintf(`
PROTON_VERSION=proton_LG_10-28
PREFIX_NAME=RFAD_SE
PROTON_USE_ESYNC=1
PROTON_USE_FSYNC=1
ENV_VARS=%s
COMMAND_ARGS=%s %s
`, strings.Join(envVars, " "), args, command)

	return os.WriteFile(configPath, []byte(content), 0644)
}
