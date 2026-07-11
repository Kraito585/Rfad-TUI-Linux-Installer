package core

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logFile *os.File
	logger  *log.Logger
	logPath string
	mu      sync.Mutex
)

// InitLogger открывает (или создаёт) файл install.log рядом с исполняемым файлом.
// Пишет ТОЛЬКО в файл, чтобы не сломать Bubble Tea TUI.
// При ошибке — пишет в io.Discard (вникуда).
func InitLogger() error {
	mu.Lock()
	defer mu.Unlock()

	if logger != nil {
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	logDir := filepath.Dir(exePath)
	logPath = filepath.Join(logDir, "install.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger = log.New(io.Discard, "", 0)
		logPath = ""
		return fmt.Errorf("не удалось открыть %s: %v", logPath, err)
	}

	logFile = f
	logger = log.New(f, "", 0)

	logger.Println(formatLog("INFO", "=== Логгер инициализирован: %s ===", logPath))
	return nil
}

// LogPath возвращает полный путь к файлу лога (пустая строка, если не инициализирован).
func LogPath() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

// CloseLogger закрывает лог-файл.
func CloseLogger() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	logger = nil
	logPath = ""
}

func formatLog(level, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, msg)
}

// LogInfo записывает информационное сообщение.
func LogInfo(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		logger.Println(formatLog("INFO", format, args...))
	}
}

// LogWarn записывает предупреждение.
func LogWarn(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		logger.Println(formatLog("WARN", format, args...))
	}
}

// LogError записывает ошибку.
func LogError(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		logger.Println(formatLog("ERROR", format, args...))
	}
}

// LogUnpacking записывает сообщение с тегом [Unpacking].
func LogUnpacking(format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		msg := fmt.Sprintf(format, args...)
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logger.Printf("[%s] [INFO] [Unpacking] %s", timestamp, msg)
	}
}
