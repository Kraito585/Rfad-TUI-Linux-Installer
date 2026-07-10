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

	// Начинаем собирать bash-скрипт
	var sb strings.Builder

	// Обязательные заголовки PortProton
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("#Author: RFAD_Installer\n")
	sb.WriteString(fmt.Sprintf("#%s\n", targetExe))
	sb.WriteString("#Rating=1-5\n")

	// Базовые настройки PortProton
	sb.WriteString("export PW_WINE_USE=\"PROTON_LG\"\n")
	sb.WriteString("export PW_PREFIX_NAME=\"RFAD_SE\"\n")

	// DLL Overrides (критично для модов)
	sb.WriteString("export WINEDLLOVERRIDES=\"xaudio2_7=n,b;d3d11=n,b;d3dx9_42=n,b;d3dcompiler_47=n,b;dinput8=n,b;mscoree=n\"\n")

	if useSteamFix {
		sb.WriteString("export START_FROM_STEAM=\"1\"\n")
	}

	if useFSR {
		sb.WriteString("export WINE_FULLSCREEN_FSR=\"1\"\n")
		sb.WriteString("export WINE_FULLSCREEN_FSR_STRENGTH=\"2\"\n")
	}

	if useNVAPI {
		sb.WriteString("export PROTON_ENABLE_NVAPI=\"1\"\n")
		// Для верности добавляем DXVK флаг, который часто идёт в паре с NVAPI
		sb.WriteString("export DXVK_ENABLE_NVAPI=\"1\"\n")
	}

	if useGameMode {
		// Стандартный флаг активации Feral GameMode в PortProton
		sb.WriteString("export PW_USE_GAMEMODE=\"1\"\n")
	}

	// Для SKSE нам нужно передать аргумент запуска
	if strings.Contains(targetExe, "SKSE") {
		// PW_CUSTOM_ARGS (или PW_CMD_ARGS) передает параметры экзешнику
		sb.WriteString("export PW_CUSTOM_ARGS=\"moshortcut://:SKSE\"\n")
	}

	return os.WriteFile(configPath, []byte(sb.String()), 0755) // 0755 чтобы файл был исполняемым
}
