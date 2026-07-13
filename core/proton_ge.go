package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallGEProton скачивает и устанавливает указанную версию GE-Proton
func InstallGEProton(version string) error {
	LogInfo("Начинаем проверку/установку %s...", version)

	// Папка для кастомных версий Proton в Steam
	homeDir, _ := os.UserHomeDir()
	compatDir := filepath.Join(homeDir, ".steam", "root", "compatibilitytools.d")

	// Создаем папку, если её нет
	if err := os.MkdirAll(compatDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать папку compatibilitytools.d: %v", err)
	}

	targetFolder := filepath.Join(compatDir, version)

	// Проверяем, не установлен ли он уже
	if _, err := os.Stat(targetFolder); !os.IsNotExist(err) {
		LogInfo("%s уже установлен в Steam. Пропускаем загрузку.", version)
		return nil
	}

	tarPath := filepath.Join(compatDir, version+".tar.gz")

	// Формируем правильную ссылку на стандартный (не ARM) архив
	downloadURL := fmt.Sprintf("https://github.com/GloriousEggroll/proton-ge-custom/releases/download/%s/%s.tar.gz", version, version)

	LogInfo("Скачивание %s из GitHub. Это может занять пару минут...", version)

	// Качаем через curl
	curlCmd := exec.Command("curl", "-L", "-o", tarPath, downloadURL)
	if err := curlCmd.Run(); err != nil {
		return fmt.Errorf("ошибка при скачивании архива: %v", err)
	}

	LogInfo("Распаковка архива...")

	// Распаковываем через tar
	tarCmd := exec.Command("tar", "-xzf", tarPath, "-C", compatDir)
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("ошибка при распаковке архива: %v", err)
	}

	// Удаляем скачанный архив, оставляя только распакованную папку
	os.Remove(tarPath)

	LogInfo("%s успешно установлен в Steam!", version)
	return nil
}
