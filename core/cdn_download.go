//go:build cdn

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type UpdateResponse struct {
	Success bool       `json:"success"`
	Data    UpdateData `json:"data"`
}

type UpdateData struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	CreatedAt int64  `json:"created_at"`
}

type progressWriter struct {
	Total      int64
	Downloaded int64
	Callback   func(float64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.Downloaded += int64(n)
	if pw.Total > 0 && pw.Callback != nil {
		pw.Callback(float64(pw.Downloaded) / float64(pw.Total))
	}
	return n, nil
}

func fetchLatestURLFromServer(ctx context.Context) (string, error) {
	apiUrl := "https://api.kraito.ru/api/v1/updates/latest"

	req, err := http.NewRequestWithContext(ctx, "GET", apiUrl, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка соединения с сервером: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("сервер вернул статус: %d", resp.StatusCode)
	}

	var apiResult UpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResult); err != nil {
		return "", fmt.Errorf("ошибка чтения ответа API: %w", err)
	}

	if !apiResult.Success {
		return "", fmt.Errorf("API вернул success: false")
	}
	s3BaseURL := "https://mirror.kraito.ru/"
	return s3BaseURL + apiResult.Data.URL, nil
}

// Универсальная функция для скачивания файла по прямой HTTP ссылке
func downloadFileHTTP(ctx context.Context, url string, destPath string, progressCb func(float64)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка скачивания, статус HTTP: %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		Total:    resp.ContentLength,
		Callback: progressCb,
	}

	// io.TeeReader прогоняет скачиваемый поток через наш progressWriter
	_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
	return err
}

// === ГЛАВНЫЕ ФУНКЦИИ (ВЫЗЫВАЮТСЯ ИЗ MAIN.GO) ===

// DownloadUpdate запрашивает URL у твоего API и качает обновление по HTTP
func DownloadUpdate(ctx context.Context, creds []byte, destPath string, progressCb func(float64)) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		LogInfo("DownloadUpdate: запрашиваем актуальный URL обновления...")

		var downloadURL string
		downloadURL, err = fetchLatestURLFromServer(ctx)
		if err != nil {
			LogError("Ошибка получения URL обновления: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		LogInfo("DownloadUpdate: начинаем скачивание %s", downloadURL)
		err = downloadFileHTTP(ctx, downloadURL, destPath, progressCb)
		if err == nil {
			return nil // Успех
		}

		LogError("DownloadUpdate: обрыв загрузки, повторяем... Ошибка: %v", err)
		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("загрузка обновления прервана после %d попыток: %v", maxRetries, err)
}

// DownloadPrefix делает то же самое для префикса
func DownloadPrefix(ctx context.Context, creds []byte, destPath string, progressCb func(float64)) error {
	prefixURL := "https://mirror.kraito.ru/rfad/prefix/prefix.tar.gz"

	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		LogInfo("DownloadPrefix: начинаем скачивание %s", prefixURL)

		err = downloadFileHTTP(ctx, prefixURL, destPath, progressCb)
		if err == nil {
			return nil
		}

		LogError("DownloadPrefix: обрыв загрузки, повторяем... Ошибка: %v", err)
		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("загрузка префикса прервана после %d попыток: %v", maxRetries, err)
}
