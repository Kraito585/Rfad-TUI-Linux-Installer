package core

import (
	"archive/zip"
	"bufio"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// =======================================================================
//                Текущая версия используемая в проекте
//      Подниму бэкэнд чтобы перезаливать обновы на S3 тогда и заменю
//	    Пока используем загрузку из google drive что крайне не надёжно
// Функции использующие google drive работают стабильно обновлятся не будут

// █   █  ███  █████     ████ █   █ ████   ███  ████  █████ █████ ████
// ██  █░█ ░░█  ░█░░░   █ ░░░░█░  █░█░░░█ █ ░░█ █░░░█  ░█░░░█░░░░░█░░░█
// █░█ █░█░ ░█░  █░░░░   ███░░█░░ █░████░░█░ ░█░████░░  █░░░████░░█░░░█░
// █░░██░█░░ █░░ █░░      ░░█ █░░ █░█░░░░ █░░ █░█░░█░ ░ █░░ █░░░░ █░░ █░░
// █░░ █░░███ ░░ █░░    ████░░ ███ ░█░░░░░ ███ ░█░░░█░  █░░ █████░████ ░░
//  ░░  ░░ ░░░ ░  ░░     ░░░░ ░ ░░░ ░░░     ░░░ ░░░  ░   ░░  ░░░░░ ░░░░ ░
//   ░   ░  ░░░    ░      ░░░░   ░░░  ░      ░░░  ░   ░   ░   ░░░░░ ░░░░

// =======================================================================

// ProcessUpdate перемещает архив, распаковывает его и заменяет EngineFixes.dll
func ProcessUpdate(gamePath string, archiveSrc string, bundledAssets embed.FS, progressCb func(float64, string)) error {
	LogUnpacking("ProcessUpdate: начало обработки обновления, gamePath=%s, archiveSrc=%s", gamePath, archiveSrc)

	targetDir := filepath.Join(gamePath, "MO2/mods/RFAD_PATCH")

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию патча: %v", err)
	}

	archiveDest := filepath.Join(targetDir, "update.zip")
	if progressCb != nil {
		progressCb(0.05, "Перемещение архива в менеджер модов...")
	}

	if err := moveFile(archiveSrc, archiveDest); err != nil {
		LogError("ProcessUpdate: ошибка перемещения архива: %v", err)
		return fmt.Errorf("ошибка перемещения архива: %v", err)
	}
	LogUnpacking("Архив перемещён: %s", archiveDest)

	LogUnpacking("Открытие ZIP-архива: %s", archiveDest)
	r, err := zip.OpenReader(archiveDest)
	if err != nil {
		LogError("ProcessUpdate: ошибка открытия ZIP: %v", err)
		return fmt.Errorf("не удалось открыть скачанный ZIP: %v", err)
	}
	defer r.Close()

	totalFiles := len(r.File)
	LogUnpacking("ZIP содержит %d файлов, начало распаковки...", totalFiles)
	for i, f := range r.File {

		fpath := filepath.Join(targetDir, f.Name)

		if !filepath.HasPrefix(fpath, filepath.Clean(targetDir)) {
			return fmt.Errorf("обнаружен некорректный путь в архиве: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

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

		if progressCb != nil {
			percent := 0.10 + (float64(i+1)/float64(totalFiles))*0.80

			cleanName := sanitize(filepath.Base(f.Name))

			if len(cleanName) > 40 {
				cleanName = "..." + cleanName[len(cleanName)-37:]
			}

			progressCb(percent, fmt.Sprintf("Распаковка: %s", cleanName))
		}
	}

	_ = os.Remove(archiveDest)
	LogUnpacking("ZIP-архив удалён после распаковки: %s", archiveDest)

	if progressCb != nil {
		progressCb(0.95, "Инъекция стабильного EngineFixes.dll...")
	}

	if progressCb != nil {
		progressCb(0.95, "Инъекция стабильного EngineFixes.dll...")
	}

	LogInfo("Извлечение EngineFixes.dll из встроенных ресурсов")
	stableDLL, err := bundledAssets.ReadFile("src/EngineFixes.dll")
	if err != nil {
		return fmt.Errorf("критическая ошибка: EngineFixes.dll не найден в src инсталлятора: %v", err)
	}

	modsDir := filepath.Join(gamePath, "MO2/mods")
	replacedCount := 0

	LogInfo("Сканирование модов для замены EngineFixes.dll в %s", modsDir)
	err = filepath.Walk(modsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "enginefixes.dll") {
			if writeErr := os.WriteFile(path, stableDLL, 0644); writeErr == nil {
				replacedCount++
			}
		}
		return nil
	})

	if replacedCount == 0 {
		LogWarn("EngineFixes.dll не найден ни в одном моде, размещаем в RFAD_PATCH")
		fallbackPath := filepath.Join(targetDir, "SKSE/Plugins/EngineFixes.dll")
		os.MkdirAll(filepath.Dir(fallbackPath), 0755)
		os.WriteFile(fallbackPath, stableDLL, 0644)
	}
	LogInfo("Заменено %d экземпляров EngineFixes.dll", replacedCount)

	if progressCb != nil {
		progressCb(1.0, "Установка успешно завершена!")
	}

	return nil
}

func moveFile(src, dst string) error {
	// Пробуем быстрый нативный перенос
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

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

	source.Close()
	destination.Close()

	return os.Remove(src)
}

func EnablePlugin(gamePath string, pluginName string) error {
	// Путь к файлу плагинов внутри профиля
	pluginTxtPath := filepath.Join(gamePath, "MO2/profiles/RFAD_SE/plugins.txt")

	// 1. Читаем существующий файл
	file, err := os.OpenFile(pluginTxtPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 2. Проверяем, нет ли уже такого плагина (ищем как с *, так и без)
	found := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "*"+pluginName || line == pluginName {
			found = true
			break
		}
	}

	// 3. Если не нашли, дописываем плагин со звездочкой
	if !found {
		// Переходим в конец файла для записи
		_, err := file.Seek(0, 2)
		if err != nil {
			return err
		}

		// Добавляем новую строку со звездочкой
		if _, err := file.WriteString("\n*" + pluginName); err != nil {
			return err
		}
		LogInfo("Плагин %s успешно активирован в plugins.txt", pluginName)
	} else {
		LogInfo("Плагин %s уже активен", pluginName)
	}

	return nil
}
