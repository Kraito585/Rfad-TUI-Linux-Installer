package core

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"rfad-installer/tui"
	"strings"
)

type ConfigPatch struct {
	TargetFile  string
	Replace     map[string]string
	InsertAfter map[string]string
}

// ApplyPatches теперь сам решает, какие патчи нужны, на основе конфига
func ApplyPatches(cfg *tui.InstallConfig, progressCallback func(percent float64, fileName string)) error {
	// 1. Внутренняя генерация списка патчей
	patches := generatePatchList(cfg)

	totalPatches := len(patches)
	for i, patch := range patches {
		fullPath := filepath.Join(cfg.InstallPath, patch.TargetFile)

		if progressCallback != nil {
			progressCallback(float64(i+1)/float64(totalPatches), patch.TargetFile)
		}

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("файл не найден: %s", fullPath)
		}

		if err := applySinglePatch(fullPath, patch); err != nil {
			return fmt.Errorf("ошибка при патчинге %s: %v", patch.TargetFile, err)
		}
	}
	return nil
}

// Внутренняя функция, скрытая от main.go
func generatePatchList(cfg *tui.InstallConfig) []ConfigPatch {
	patches := []ConfigPatch{
		{
			TargetFile: "MO2/mods/SSE Display Tweaks/SKSE/Plugins/SSEDisplayTweaks.ini",
			Replace:    map[string]string{"FramerateLimit = 60": "FramerateLimit = 0"},
		},
	}

	if cfg.UseFSR {
		patches = append(patches, ConfigPatch{
			TargetFile: "MO2/profiles/RFAD_SE/SkyrimPrefs.ini",
			Replace: map[string]string{
				"iSize W=1920": fmt.Sprintf("iSize W=%s", cfg.ResWidth),
				"iSize H=1080": fmt.Sprintf("iSize H=%s", cfg.ResHeight),
			},
		})
	}
	return patches
}

func applySinglePatch(filePath string, patch ConfigPatch) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	var newLines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if replaceWith, ok := patch.Replace[trimmedLine]; ok {
			newLines = append(newLines, replaceWith)
		} else {
			newLines = append(newLines, line)
		}

		if insertStr, ok := patch.InsertAfter[trimmedLine]; ok {
			newLines = append(newLines, insertStr)
		}
	}

	file.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	outputContent := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(outputContent), 0644)
}
