package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Структуры под твой API
type PresetsResponse struct {
	Success bool     `json:"success"`
	Data    []Preset `json:"data"`
}

type Preset struct {
	ID                string         `json:"id"`
	URL               string         `json:"url"`
	Images            []string       `json:"images"`
	PerformanceImpact int            `json:"performance_impact"`
	Metadata          PresetMetadata `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
}

type PresetMetadata struct {
	OriginURL      string        `json:"originUrl"`
	AuthorNickname string        `json:"authorNickname"`
	Description    string        `json:"description"`
	OptionalMods   []OptionalMod `json:"optional_mods"`
}

type OptionalMod struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	IsRequired bool     `json:"is_required"`
	DependsOn  []string `json:"depends_on"`
}

var GlobalPresets []Preset

// FetchAndCachePresets вызываем в самом начале функции main() в main.go
func FetchAndCachePresets(cacheDir string) {
	apiURL := "https://api.kraito.ru/api/v1/community/shaders"

	resp, err := http.Get(apiURL)
	if err != nil {
		LogError("Не удалось получить пресеты: %v", err)
		return
	}
	defer resp.Body.Close()

	var result PresetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		LogError("Ошибка парсинга JSON пресетов: %v", err)
		return
	}

	GlobalPresets = result.Data

	// Запускаем фоновую загрузку картинок
	go func() {
		imagesDir := filepath.Join(cacheDir, "images")
		os.MkdirAll(imagesDir, 0755)

		for _, preset := range GlobalPresets {
			for i, imgPath := range preset.Images {
				// В идеале imgPath это относительный путь на S3
				fullURL := "https://mirror.kraito.ru/" + imgPath
				localPath := filepath.Join(imagesDir, fmt.Sprintf("%s_%d.png", preset.ID, i))

				if _, err := os.Stat(localPath); os.IsNotExist(err) {
					downloadFile(fullURL, localPath)
				}
			}
		}
	}()
}

func downloadFile(url, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
