package layout_engine

// GraphicLayer хранит растровое изображение и его абсолютные координаты
type GraphicLayer struct {
	EscapeCode string // \x1b_Ga=T...
	Bounds     Rect   // Точное место, где это должно быть нарисовано
}

// ClearGPU отправляет команду очистки памяти терминала (a=d, d=a)
func ClearGPU() string {
	return "\x1b_Ga=d,d=a;\x1b\\"
}
