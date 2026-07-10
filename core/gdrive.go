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

// DownloadArchive теперь включает логику повторов при сбоях сети
func DownloadArchive(ctx context.Context, credsJSON []byte, folderID string, destPath string, progressCb func(float64)) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = downloadLogic(ctx, credsJSON, folderID, destPath, progressCb)
		if err == nil {
			return nil // Успех
		}

		// Если ошибка - ждем перед повтором
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("загрузка прервана после %d попыток: %v", maxRetries, err)
}

// Вспомогательная функция для логики загрузки
func downloadLogic(ctx context.Context, credsJSON []byte, folderID string, destPath string, progressCb func(float64)) error {
	srv, err := drive.NewService(ctx, option.WithCredentialsJSON(credsJSON))
	if err != nil {
		return err
	}

	query := fmt.Sprintf("'%s' in parents and mimeType='application/x-zip-compressed'", folderID)
	r, err := srv.Files.List().Q(query).Fields("files(id, name, size)").Do()
	if err != nil {
		return err
	}

	if len(r.Files) == 0 {
		return fmt.Errorf("архив не найден")
	}

	fileID := r.Files[0].Id
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
	return err
}

func DownloadPrefixDirectly(ctx context.Context, credsJSON []byte, destPath string, progressCb func(float64)) error {
	fileID := "1HxR7TIpXculDlJ9qpd8gnCN4qSzCMOgH" // ID из твоей ссылки
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = downloadPrefixLogic(ctx, credsJSON, fileID, destPath, progressCb)
		if err == nil {
			return nil
		}
		// Пауза перед повтором при обрыве связи
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("загрузка префикса прервана после %d попыток: %v", maxRetries, err)
}

func downloadPrefixLogic(ctx context.Context, credsJSON []byte, fileID string, destPath string, progressCb func(float64)) error {
	srv, err := drive.NewService(ctx, option.WithCredentialsJSON(credsJSON))
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
	return err
}
