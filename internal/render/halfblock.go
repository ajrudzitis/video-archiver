package render

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"

	"golang.org/x/image/draw"
)

type halfBlockRenderer struct{}

func (halfBlockRenderer) Kind() Kind { return KindHalfBlock }

func (halfBlockRenderer) Clear() string { return "" }

// RenderPNG decodes the PNG, scales it to fit cols x rows cells (each cell
// holding two vertical pixels via the upper-half-block glyph), and emits
// 24-bit ANSI color output.
func (halfBlockRenderer) RenderPNG(data []byte, cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	sb := src.Bounds()
	cw, ch := fitBox(sb.Dx(), sb.Dy(), cols, rows)
	if cw == 0 || ch == 0 {
		return ""
	}
	dst := image.NewRGBA(image.Rect(0, 0, cw, ch*2))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, sb, draw.Src, nil)

	pad := strings.Repeat(" ", (cols-cw)/2)
	var buf bytes.Buffer
	for row := 0; row < ch; row++ {
		buf.WriteString(pad)
		for x := 0; x < cw; x++ {
			top := dst.RGBAAt(x, row*2)
			bot := dst.RGBAAt(x, row*2+1)
			fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm\u2580",
				top.R, top.G, top.B, bot.R, bot.G, bot.B)
		}
		buf.WriteString("\x1b[0m")
		if row < ch-1 {
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}
