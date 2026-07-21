package core

import (
	"bytes"
	"image"
	_ "image/jpeg" // Поддержка декодирования JPEG
	_ "image/png"  // Поддержка декодирования PNG
	"os"

	"github.com/mattn/go-sixel"
	"github.com/nfnt/resize"
)

// GetSixelString конвертирует картинку в Sixel-строку с нужными отступами
func GetSixelString(imagePath string, maxCols, maxRows int) string {
	file, err := os.Open(imagePath)
	if err != nil {
		return " [ Изображение загружается... ] "
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return " [ Ошибка декодирования изображения ] "
	}

	cellWidth := 8
	cellHeight := 16

	targetWidth := uint(maxCols * cellWidth)
	targetHeight := uint(maxRows * cellHeight)

	resizedImg := resize.Thumbnail(targetWidth, targetHeight, img, resize.Lanczos3)

	var buf bytes.Buffer
	// Очищаем область от текущей позиции курсора до конца экрана,
	// чтобы стереть предыдущий Sixel перед выводом нового
	buf.WriteString("\x1b[J")

	enc := sixel.NewEncoder(&buf)
	if err := enc.Encode(resizedImg); err != nil {
		return "Ошибка"
	}

	// Один перевод строки после картинки, чтобы курсор был под изображением
	return buf.String() + "\n"
}
