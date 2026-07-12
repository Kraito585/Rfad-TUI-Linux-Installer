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
	TargetFile    string
	Replace       map[string]string
	ReplacePrefix map[string]string
	InsertAfter   map[string]string
}

// ApplyPatches теперь сам решает, какие патчи нужны, на основе конфига
func ApplyPatches(cfg *tui.InstallConfig, progressCallback func(percent float64, fileName string)) error {
	patches := generatePatchList(cfg)
	LogInfo("ApplyPatches: сгенерировано %d патчей для применения", len(patches))

	totalPatches := len(patches)
	for i, patch := range patches {
		fullPath := filepath.Join(cfg.InstallPath, patch.TargetFile)

		if progressCallback != nil {
			progressCallback(float64(i+1)/float64(totalPatches), patch.TargetFile)
		}

		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			LogError("ApplyPatches: файл не найден: %s", fullPath)
			return fmt.Errorf("файл не найден: %s", fullPath)
		}

		LogInfo("ApplyPatches: патчинг %s...", patch.TargetFile)
		if err := applySinglePatch(fullPath, patch); err != nil {
			LogError("ApplyPatches: ошибка при патчинге %s: %v", patch.TargetFile, err)
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
			ReplacePrefix: map[string]string{
				"iSize W=": fmt.Sprintf("iSize W=%s", cfg.ResWidth),
				"iSize H=": fmt.Sprintf("iSize H=%s", cfg.ResHeight),
			},
		})

		resString := fmt.Sprintf("Resolution = %sx%s", cfg.ResWidth, cfg.ResHeight)
		patches = append(patches, ConfigPatch{
			TargetFile: "MO2/mods/SSE Display Tweaks/SKSE/Plugins/SSEDisplayTweaks.ini",
			Replace: map[string]string{
				"Fullscreen = false": "Fullscreen = true",
				"Borderless = true":  "Borderless = false",
			},
			ReplacePrefix: map[string]string{
				"Resolution =": resString,
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

		replaced := false

		if patch.Replace != nil {
			if replaceWith, ok := patch.Replace[trimmedLine]; ok {
				newLines = append(newLines, replaceWith)
				replaced = true
			}
		}

		if !replaced && patch.ReplacePrefix != nil {
			for prefix, replaceWith := range patch.ReplacePrefix {
				if strings.HasPrefix(trimmedLine, prefix) {
					newLines = append(newLines, replaceWith)
					replaced = true
					break
				}
			}
		}

		if !replaced {
			newLines = append(newLines, line)
		}

		if patch.InsertAfter != nil {
			if insertStr, ok := patch.InsertAfter[trimmedLine]; ok {
				newLines = append(newLines, insertStr)
			}
		}
	}

	file.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	outputContent := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(outputContent), 0644)
}
