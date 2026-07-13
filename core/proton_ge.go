package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallGEProton скачивает и устанавливает указанную версию GE-Proton для Steam
// в пиратской версии PortProton делает это самостоятельно
func InstallGEProton(version string) error {
	LogInfo("Начинаем проверку/установку %s...", version)

	homeDir, _ := os.UserHomeDir()
	compatDir := filepath.Join(homeDir, ".steam", "root", "compatibilitytools.d")

	if err := os.MkdirAll(compatDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать папку compatibilitytools.d: %v", err)
	}

	targetFolder := filepath.Join(compatDir, version)

	if _, err := os.Stat(targetFolder); !os.IsNotExist(err) {
		LogInfo("%s уже установлен в Steam. Пропускаем загрузку.", version)
		return nil
	}

	tarPath := filepath.Join(compatDir, version+".tar.gz")

	downloadURL := fmt.Sprintf("https://github.com/GloriousEggroll/proton-ge-custom/releases/download/%s/%s.tar.gz", version, version)

	LogInfo("Скачивание %s из GitHub. Это может занять пару минут...", version)

	curlCmd := exec.Command("curl", "-L", "-o", tarPath, downloadURL)
	if err := curlCmd.Run(); err != nil {
		return fmt.Errorf("ошибка при скачивании архива: %v", err)
	}

	LogInfo("Распаковка архива...")

	tarCmd := exec.Command("tar", "-xzf", tarPath, "-C", compatDir)
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("ошибка при распаковке архива: %v", err)
	}

	os.Remove(tarPath)

	LogInfo("%s успешно установлен в Steam!", version)
	return nil
}
