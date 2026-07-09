package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GeneratePPDB(targetExe string, useFSR, useNVAPI, useGameMode bool) error {
	// Имя файла конфига должно совпадать с именем экзешника
	ppdbName := strings.TrimSuffix(targetExe, filepath.Ext(targetExe)) + ".ppdb"
	configPath := filepath.Join("MO2", ppdbName)

	envVars := []string{
		"WINEDLLOVERRIDES=\"xaudio2_7=n,b;d3d11=n,b;d3dx9_42=n,b;d3dcompiler_47=n,b;dinput8=n,b;mscoree=n\"",
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

	// Только для SKSE версии добавляем аргумент запуска
	args := ""
	if strings.Contains(targetExe, "SKSE") {
		args = "moshortcut://:SKSE"
	}

	content := fmt.Sprintf(`
PROTON_VERSION=proton_LG_10-28
PROTON_USE_ESYNC=1
PROTON_USE_FSYNC=1
ENV_VARS=%s
COMMAND_ARGS=%s %s
`, strings.Join(envVars, " "), args, command)

	return os.WriteFile(configPath, []byte(content), 0644)
}
