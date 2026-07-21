package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"

	"github.com/nfnt/resize"
)

func GetKittyEscape(imagePath string, maxCols, maxRows int) (escape string, cols, rows int) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", 0, 0
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", 0, 0
	}

	// Усредненные пропорции символа в терминале
	cellWidth := 9
	cellHeight := 18

	targetWidth := uint(maxCols * cellWidth)
	targetHeight := uint(maxRows * cellHeight)

	// Делаем ресайз
	resizedImg := resize.Thumbnail(targetWidth, targetHeight, img, resize.Lanczos3)

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, resizedImg); err != nil {
		return "", 0, 0
	}

	b64 := base64.StdEncoding.EncodeToString(pngBuf.Bytes())

	bounds := resizedImg.Bounds()
	pixW := bounds.Dx()
	pixH := bounds.Dy()

	cols = (pixW + cellWidth - 1) / cellWidth
	rows = (pixH + cellHeight - 1) / cellHeight

	// a=T (передать и отобразить), q=2 (тихий режим), c=колонки, r=строки
	escape = fmt.Sprintf("\x1b_Ga=T,f=100,q=2,c=%d,r=%d;%s\x1b\\", cols, rows, b64)

	return escape, cols, rows
}
