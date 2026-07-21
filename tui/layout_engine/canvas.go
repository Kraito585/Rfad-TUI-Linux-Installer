package layout_engine

// Canvas - это виртуальный экран.
// Мы можем писать текст строго по заданным координатам X и Y.
type Canvas struct {
	Width  int
	Height int
	// Внутри это может быть двумерный массив рун (ячеек)
	// cells [][]Cell
}

// NewCanvas создает чистый холст размером с текущее окно терминала
func NewCanvas(width, height int) *Canvas {
	return &Canvas{Width: width, Height: height}
}

// DrawText размещает строку lipgloss строго по координатам X и Y
func (c *Canvas) DrawText(x, y int, text string) {
	// Здесь будет логика вставки текста в матрицу экрана
}

// ReserveHole сообщает движку, что эту зону перекрывать текстом нельзя
func (c *Canvas) ReserveHole(rect Rect) {
	// Движок заполнит эту область пробелами
}
