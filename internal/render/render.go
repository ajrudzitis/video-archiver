package render

import (
	"bytes"
	"image"
	"os"
	"strings"
)

type Kind int

const (
	KindNone Kind = iota
	KindHalfBlock
	KindKitty
)

func (k Kind) String() string {
	switch k {
	case KindKitty:
		return "kitty"
	case KindHalfBlock:
		return "halfblock"
	default:
		return "off"
	}
}

type Renderer interface {
	Kind() Kind
	// RenderPNG renders a PNG image into a string that fits within a box of
	// cols x rows terminal cells, preserving aspect ratio.
	RenderPNG(png []byte, cols, rows int) string
	// Clear returns sequences that remove any image previously rendered.
	Clear() string
}

func New(kind Kind) Renderer {
	switch kind {
	case KindKitty:
		return &kittyRenderer{id: 99}
	case KindHalfBlock:
		return &halfBlockRenderer{}
	default:
		return noneRenderer{}
	}
}

// Detect picks a renderer kind from a user preference ("auto", "kitty",
// "halfblock", "off") and the terminal environment.
func Detect(preference string) Kind {
	switch strings.ToLower(preference) {
	case "kitty":
		return KindKitty
	case "halfblock", "blocks":
		return KindHalfBlock
	case "off", "none":
		return KindNone
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return KindKitty
	}
	if os.Getenv("TERM") == "xterm-ghostty" {
		return KindKitty
	}
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "ghostty", "wezterm", "kitty":
		return KindKitty
	}
	return KindHalfBlock
}

// fitBox computes the display size in terminal cells for an image of w x h
// pixels inside a box of cols x rows cells, assuming cells are twice as tall
// as they are wide.
func fitBox(w, h, cols, rows int) (int, int) {
	if w <= 0 || h <= 0 || cols <= 0 || rows <= 0 {
		return 0, 0
	}
	// Work in half-pixel units: cols wide, rows*2 tall.
	halfH := rows * 2
	fw, fh := float64(w), float64(h)
	scale := min(float64(cols)/fw, float64(halfH)/fh)
	cw := int(fw*scale + 0.5)
	ch := int(fh*scale + 0.5)
	if cw < 1 {
		cw = 1
	}
	if ch < 1 {
		ch = 1
	}
	if cw > cols {
		cw = cols
	}
	if ch > halfH {
		ch = halfH
	}
	cellRows := (ch + 1) / 2
	return cw, cellRows
}

func decodeSize(pngData []byte) (int, int, bool) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(pngData))
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}
