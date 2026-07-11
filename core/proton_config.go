package core

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func getFileHash(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func GeneratePPDB(gamePath string, targetExe string, wineVersion string, useFSR, useNVAPI, useGameMode, useSteamFix bool) error {
	LogInfo("GeneratePPDB: генерация конфига для %s (Wine: %s)", targetExe, wineVersion)

	ppdbName := targetExe + ".ppdb"
	exePath := filepath.Join(gamePath, "MO2", targetExe)
	configPath := filepath.Join(gamePath, "MO2", ppdbName)

	hash := getFileHash(exePath)

	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("#Author: RFAD_Installer\n")
	sb.WriteString(fmt.Sprintf("#%s\n", targetExe))
	sb.WriteString("#Rating=1-5\n")

	sb.WriteString(fmt.Sprintf("export PW_WINE_USE=\"%s\"\n", wineVersion))
	sb.WriteString("export PW_PREFIX_NAME=\"RFAD_SE\"\n")

	sb.WriteString("export PW_GUI_DISABLED_CS=\"1\"\n")

	sb.WriteString("export PW_VULKAN_USE=\"6\"\n")

	if hash != "" {
		sb.WriteString(fmt.Sprintf("export FILE_SHA256SUM=\"%s\"\n", hash))
	}

	if strings.Contains(targetExe, "SKSE") {
		sb.WriteString("export PORTPROTON_NAME=\"RFAD Game (SKSE)\"\n")
		sb.WriteString("export FILE_DESCRIPTION=\"Skyrim SE Launcher\"\n")
	} else {
		sb.WriteString("export PORTPROTON_NAME=\"Mod Organizer 2\"\n")
		sb.WriteString("export FILE_DESCRIPTION=\"Mod Organizer 2 GUI\"\n")
	}

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
		sb.WriteString("export DXVK_ENABLE_NVAPI=\"1\"\n")
	}

	if useGameMode {
		sb.WriteString("export PW_USE_GAMEMODE=\"1\"\n")
	}

	if strings.Contains(targetExe, "SKSE") {
		sb.WriteString("export PW_CUSTOM_ARGS=\"moshortcut://:SKSE\"\n")
	}

	if err := os.WriteFile(configPath, []byte(sb.String()), 0755); err != nil {
		LogError("GeneratePPDB: ошибка записи %s: %v", configPath, err)
		return err
	}

	LogInfo("GeneratePPDB: конфиг сохранён в %s", configPath)
	return nil
}
