package layout_engine

// Point описывает точные координаты на экране
type Point struct {
	X, Y int
}

// Size описывает габариты окна или блока
type Size struct {
	Width, Height int
}

// Rect описывает прямоугольную область (например, "дыру" для картинки)
type Rect struct {
	Pos  Point
	Size Size
}
