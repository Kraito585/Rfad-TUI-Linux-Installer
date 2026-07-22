package core

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"rfad-installer/tui"
	"strconv"
	"strings"
)

type ConfigPatch struct {
	TargetFile    string
	Replace       map[string]string
	ReplacePrefix map[string]string
	InsertAfter   map[string]string
}

func getQualityMode(cfg *tui.InstallConfig) int {
	if cfg.FSRLevel == 4 {
		scale, err := strconv.ParseFloat(cfg.CustomFSRScale, 64)
		if err != nil || scale <= 0 {
			return 1 // По умолчанию Quality (67%)
		}

		if scale >= 95.0 {
			return 0 // Native
		}

		if scale >= 63.0 {
			return 1 // Quality
		}

		if scale >= 55.0 {
			return 2 // Balanced
		}

		if scale >= 45.0 {
			return 3 // Performance
		}

		return 4
	}

	if !cfg.UseFSR {
		return 0
	}
	return cfg.FSRLevel
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

	if cfg.UseFSR && cfg.GraphicsMod != "Community Shaders" {
		var finalW, finalH string

		switch cfg.FSRLevel {
		case 1:
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.75))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.75))
		case 2:
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.50))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.50))
		case 3:
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*0.25))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*0.25))
		case 4:
			scale, err := strconv.ParseFloat(cfg.CustomFSRScale, 64)
			if err != nil || scale <= 0 {
				scale = 67.0
			}
			multiplier := scale / 100.0
			finalW = fmt.Sprintf("%d", int(float64(cfg.BaseWidth)*multiplier))
			finalH = fmt.Sprintf("%d", int(float64(cfg.BaseHeight)*multiplier))
		default:
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

	// Влкючение Community Shaders и лечение его зависимостей
	if cfg.GraphicsMod == "Community Shaders" {
		patches = append(patches, ConfigPatch{
			TargetFile: "MO2/profiles/RFAD_SE/modlist.txt",
			Replace: map[string]string{
				"+Enhanced Volumetric Lighting and Shadows (EVLaS)": "-Enhanced Volumetric Lighting and Shadows (EVLaS)",
			},
			InsertAfter: map[string]string{
				"-MY MODS_separator": "\n+Community Shaders 86492 1.7.3 2026-06-27T10-38Z 6Xybdafll\n+Upscaling 156952 1.4.0 2026-05-31T10-27Z L5WQbqiov\n-KRAITO PATCH",
			},
		})
	}

	if cfg.UseFSR && cfg.GraphicsMod == "Community Shaders" {
		fsrLevel := fmt.Sprintf("%v", getQualityMode(cfg))
		patches = append(patches, ConfigPatch{
			TargetFile: "MO2/overwrite/SKSE/Plugins/CommunityShaders/SettingsUser.json",
			Replace: map[string]string{
				"Fullscreen = false": "Fullscreen = true",
				"Borderless = true":  "Borderless = false",
			},
		})
	}

	// Патч для ReShade отключение UI
	if cfg.GraphicsMod == "ReShade" {
		patches = append(patches, ConfigPatch{
			TargetFile: "ReShade.ini",
			InsertAfter: map[string]string{
				"[GENERAL]": "KeyOverlay=0,0,0,0\nShowOverlay=0\nNoReloadOnInit=1",
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
				for _, subLine := range strings.Split(insertStr, "\n") {
					if subLine != "" {
						newLines = append(newLines, subLine)
					}
				}
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
