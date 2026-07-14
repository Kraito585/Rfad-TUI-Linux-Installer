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

// Правки конфигов ничего интересного
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

func generatePatchList(cfg *tui.InstallConfig) []ConfigPatch {
	patches := []ConfigPatch{
		{
			TargetFile: "MO2/mods/SSE Display Tweaks/SKSE/Plugins/SSEDisplayTweaks.ini",
			Replace:    map[string]string{"FramerateLimit = 60": "FramerateLimit = 0"},
		},
	}

	if cfg.UseFSR {
		var finalW, finalH string

		// Высчитываем итоговое разрешение рендера на основе выбранного уровня FSR
		switch cfg.FSRLevel {
		case 1: // 75% от базового разрешения
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.75))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.75))
		case 2: // 50% от базового разрешения
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.50))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.50))
		case 3: // 25% от базового разрешения (максимальная производительность)
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.25))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.25))
		case 4: // Своё значение (ручной ввод пользователя)
			finalW = cfg.ResWidth
			finalH = cfg.ResHeight
		default:
			// Фолбэк на случай непредвиденных ошибок (ставим 75%)
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.75))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.75))
		}

		// Патчим SkyrimPrefs.ini
		patches = append(patches, ConfigPatch{
			TargetFile: "MO2/profiles/RFAD_SE/SkyrimPrefs.ini",
			ReplacePrefix: map[string]string{
				"iSize W=": fmt.Sprintf("iSize W=%s", finalW),
				"iSize H=": fmt.Sprintf("iSize H=%s", finalH),
			},
		})

		// Патчим SSE Display Tweaks
		resString := fmt.Sprintf("Resolution = %sx%s", finalW, finalH)
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
