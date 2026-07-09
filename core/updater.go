package core

import (
	"archive/zip"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ProcessUpdate перемещает архив, распаковывает его и заменяет EngineFixes.dll
func ProcessUpdate(gamePath string, archiveSrc string, bundledAssets embed.FS, progressCb func(float64, string)) error {
	targetDir := filepath.Join(gamePath, "MO2/mods/RFAD_PATCH")

	// 1. Создаем директорию для мода-патча, если её нет
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию патча: %v", err)
	}

	// 2. Перемещение архива в целевую папку мода
	archiveDest := filepath.Join(targetDir, "update.zip")
	if progressCb != nil {
		progressCb(0.05, "Перемещение архива в менеджер модов...")
	}

	if err := moveFile(archiveSrc, archiveDest); err != nil {
		return fmt.Errorf("ошибка перемещения архива: %v", err)
	}

	// 3. Распаковка архива
	r, err := zip.OpenReader(archiveDest)
	if err != nil {
		return fmt.Errorf("не удалось открыть скачанный ZIP: %v", err)
	}
	defer r.Close()

	totalFiles := len(r.File)
	for i, f := range r.File {
		// Формируем финальный путь для каждого файла внутри папки мода
		fpath := filepath.Join(targetDir, f.Name)

		// Защита от Zip Slip уязвимости (проверка путей)
		if !filepath.HasPrefix(fpath, filepath.Clean(targetDir)) {
			return fmt.Errorf("обнаружен некорректный путь в архиве: %s", f.Name)
		}

		// Если это директория — создаем её
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Если это файл — распаковываем
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
		if err != nil {
			return err
		}

		// Отправляем прогресс распаковки (займет диапазон от 10% до 90% на общем статус-баре)
		if progressCb != nil {
			percent := 0.10 + (float64(i+1)/float64(totalFiles))*0.80
			progressCb(percent, fmt.Sprintf("Распаковка: %s", filepath.Base(f.Name)))
		}
	}

	// Удаляем сам ZIP-архив после успешной распаковки, чтобы не занимал место
	_ = os.Remove(archiveDest)

	// 4. Удаление и чистая замена EngineFixes.dll из вшитых ресурсов
	if progressCb != nil {
		progressCb(0.95, "Инъекция стабильного EngineFixes.dll...")
	}

	// Стандартный путь плагина внутри структуры мода Skyrim
	dllPath := filepath.Join(targetDir, "SKSE/Plugins/EngineFixes.dll")

	// Насильно удаляем старый файл, если он распаковался из апдейта
	_ = os.Remove(dllPath)

	// Гарантируем наличие подпапок SKSE/Plugins
	if err := os.MkdirAll(filepath.Dir(dllPath), 0755); err != nil {
		return err
	}

	// Достаем стабильный DLL из бинарника
	stableDLL, err := bundledAssets.ReadFile("src/EngineFixes.dll")
	if err != nil {
		return fmt.Errorf("критическая ошибка: EngineFixes.dll не найден в src инсталлятора: %v", err)
	}

	// Записываем его на диск
	if err := os.WriteFile(dllPath, stableDLL, 0644); err != nil {
		return fmt.Errorf("не удалось записать стабильный EngineFixes.dll: %v", err)
	}

	if progressCb != nil {
		progressCb(1.0, "Мод-патч успешно сформирован!")
	}
	return nil
}

// Вспомогательная функция для безопасного перемещения между разными дисками
func moveFile(src, dst string) error {
	// Пробуем быстрый нативный перенос
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Если разные разделы диска — переходим на ручное копирование потока
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	// Закрываем файлы перед удалением оригинала
	source.Close()
	destination.Close()

	return os.Remove(src)
}
