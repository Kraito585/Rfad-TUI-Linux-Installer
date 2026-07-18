//go:build gdrive

package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Зашиваем ID, так как они нужны только Google Drive
const (
	updateFolderID = "1JUOctbsugh2IIEUCWcBkupXYVYoJMg4G"
	prefixFileID   = "1HxR7TIpXculDlJ9qpd8gnCN4qSzCMOgH"
)

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

// DownloadUpdate скачивает обновление
func DownloadUpdate(ctx context.Context, creds []byte, destPath string, progressCb func(float64)) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = downloadLogic(ctx, creds, updateFolderID, destPath, progressCb)
		if err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("загрузка обновления прервана после %d попыток: %v", maxRetries, err)
}

func downloadLogic(ctx context.Context, creds []byte, folderID string, destPath string, progressCb func(float64)) error {
	LogInfo("DownloadUpdate: поиск архива в папке Google Drive ID=%s", folderID)

	// Инициализируем сервис, используя переданные байты credentials.json
	srv, err := drive.NewService(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		return fmt.Errorf("ошибка авторизации Google Drive API: %v", err)
	}

	query := fmt.Sprintf("'%s' in parents and mimeType='application/x-zip-compressed'", folderID)
	r, err := srv.Files.List().Q(query).Fields("files(id, name, size)").Do()
	if err != nil {
		return err
	}

	if len(r.Files) == 0 {
		LogError("DownloadUpdate: архив не найден в папке")
		return fmt.Errorf("архив не найден")
	}

	fileID := r.Files[0].Id
	LogInfo("DownloadUpdate: найден файл %s (ID=%s, размер=%d байт)", r.Files[0].Name, fileID, r.Files[0].Size)
	res, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		Total:    res.ContentLength,
		Callback: progressCb,
	}

	_, err = io.Copy(out, io.TeeReader(res.Body, pw))
	if err != nil {
		LogError("DownloadUpdate: ошибка при скачивании: %v", err)
		return err
	}
	LogInfo("DownloadUpdate: архив скачан в %s", destPath)
	return nil
}

// DownloadPrefix скачивает префикс. Обрати внимание: сюда тоже добавлен параметр creds []byte
func DownloadPrefix(ctx context.Context, creds []byte, destPath string, progressCb func(float64)) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = downloadPrefixLogic(ctx, creds, prefixFileID, destPath, progressCb)
		if err == nil {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("загрузка префикса прервана после %d попыток: %v", maxRetries, err)
}

func downloadPrefixLogic(ctx context.Context, creds []byte, fileID string, destPath string, progressCb func(float64)) error {
	LogInfo("DownloadPrefix: начало загрузки префикса, fileID=%s", fileID)

	// Инициализируем сервис
	srv, err := drive.NewService(ctx, option.WithCredentialsJSON(creds))
	if err != nil {
		return fmt.Errorf("ошибка авторизации Google Drive API: %v", err)
	}

	res, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("ошибка запроса файла префикса: %v", err)
	}
	defer res.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("ошибка создания локального файла: %v", err)
	}
	defer out.Close()

	pw := &progressWriter{
		Total:    res.ContentLength,
		Callback: progressCb,
	}

	_, err = io.Copy(out, io.TeeReader(res.Body, pw))
	if err != nil {
		LogError("DownloadPrefix: ошибка при скачивании: %v", err)
		return err
	}
	LogInfo("DownloadPrefix: префикс скачан в %s", destPath)
	return nil
}
