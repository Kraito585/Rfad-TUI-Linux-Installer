package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

// passThru оборачивает io.Reader для отслеживания прогресса скачивания
type passThru struct {
	io.Reader
	total int64
	curr  int64
	cb    func(float64, string)
}

func (pt *passThru) Read(p []byte) (int, error) {
	n, err := pt.Reader.Read(p)
	pt.curr += int64(n)
	if pt.cb != nil && pt.total > 0 {
		percent := float64(pt.curr) / float64(pt.total)
		// Прогресс скачивания от 0 до 80%
		pt.cb(percent*0.8, fmt.Sprintf("Загрузка GE-Proton: %d%%", int(percent*100)))
	}
	return n, err
}

// InstallProtonGE скачивает и устанавливает кастомный Proton в PortProton
func InstallProtonGE(progressCb func(float64, string)) (string, error) {
	protonVersion := "GE-Proton9-10" // Жестко фиксируем стабильную версию
	home, _ := os.UserHomeDir()

	// PortProton может лежать в двух местах в зависимости от типа установки
	ppDist := filepath.Join(home, "PortProton", "data", "dist")
	if stat, err := os.Stat(ppDist); err != nil || !stat.IsDir() {
		ppDist = filepath.Join(home, ".local", "share", "PortProton", "data", "dist")
	}

	targetDir := filepath.Join(ppDist, protonVersion)

	// Если папка уже существует, значит Proton уже установлен
	if _, err := os.Stat(targetDir); err == nil {
		if progressCb != nil {
			progressCb(1.0, protonVersion+" уже установлен.")
		}
		return protonVersion, nil
	}

	// Формируем прямую ссылку на релиз в GitHub
	downloadURL := fmt.Sprintf("https://github.com/GloriousEggroll/proton-ge-custom/releases/download/%s/%s.tar.gz", protonVersion, protonVersion)
	tmpArchive := filepath.Join(os.TempDir(), protonVersion+".tar.gz")

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("ошибка доступа к GitHub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("не удалось скачать архив (HTTP %d)", resp.StatusCode)
	}

	out, err := os.Create(tmpArchive)
	if err != nil {
		return "", err
	}

	// Скачиваем с отслеживанием прогресса
	pt := &passThru{
		Reader: resp.Body,
		total:  resp.ContentLength,
		cb:     progressCb,
	}

	_, err = io.Copy(out, pt)
	out.Close()
	if err != nil {
		return "", fmt.Errorf("ошибка записи архива: %v", err)
	}
	defer os.Remove(tmpArchive) // Убираем за собой временный файл

	if progressCb != nil {
		progressCb(0.9, "Распаковка "+protonVersion+" (это займет минуту)...")
	}

	// Распаковываем архив прямо в директорию dist PortProton'а
	cmd := exec.Command("tar", "-xzf", tmpArchive, "-C", ppDist)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ошибка распаковки tar.gz: %v", err)
	}

	if progressCb != nil {
		progressCb(1.0, protonVersion+" успешно установлен!")
	}

	return protonVersion, nil
}
