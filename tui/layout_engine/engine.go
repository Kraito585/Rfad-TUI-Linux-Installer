package layout_engine

import "fmt"

type Engine struct {
	ScreenSize Size
}

func NewEngine() *Engine {
	return &Engine{}
}

// Resize вызывается при tea.WindowSizeMsg
func (e *Engine) Resize(width, height int) {
	e.ScreenSize = Size{Width: width, Height: height}
}

// Render — это финальный сборщик!
// Он берет наш текстовый Canvas и список растровых GraphicLayer,
// и превращает их в ОДНУ итоговую строку для метода View() в Bubble Tea.
func (e *Engine) Render(canvas *Canvas, graphics []GraphicLayer) string {
	var finalView string

	// 1. Сначала превращаем Canvas в обычную строку (текстовый TUI)
	// finalView = canvas.ToString()

	// 2. Очищаем старую графику
	finalView += ClearGPU()

	// 3. Поверх текста накладываем растровые картинки строго по координатам
	for _, g := range graphics {
		// ANSI позиционирование курсора + печать картинки + возврат курсора
		positionCmd := fmt.Sprintf("\x1b[s\x1b[%d;%dH", g.Bounds.Pos.Y, g.Bounds.Pos.X)
		finalView += positionCmd + g.EscapeCode + "\x1b[u"
	}

	return finalView
}
