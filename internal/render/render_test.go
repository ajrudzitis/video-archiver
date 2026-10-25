package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"testing"
)

func makePNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDetect(t *testing.T) {
	for _, tc := range []struct {
		pref        string
		termProgram string
		kittyWin    string
		want        Kind
	}{
		{"kitty", "", "", KindKitty},
		{"halfblock", "ghostty", "1", KindHalfBlock},
		{"off", "ghostty", "1", KindNone},
		{"auto", "ghostty", "", KindKitty},
		{"auto", "WezTerm", "", KindKitty},
		{"auto", "", "1", KindKitty},
		{"auto", "xterm", "", KindHalfBlock},
		{"auto", "", "", KindHalfBlock},
		{"bogus", "", "", KindHalfBlock},
	} {
		t.Setenv("TERM_PROGRAM", tc.termProgram)
		t.Setenv("KITTY_WINDOW_ID", tc.kittyWin)
		t.Setenv("TERM", "xterm-256color")
		if got := Detect(tc.pref); got != tc.want {
			t.Errorf("Detect(%q) with TERM_PROGRAM=%q KITTY_WINDOW_ID=%q = %v, want %v",
				tc.pref, tc.termProgram, tc.kittyWin, got, tc.want)
		}
	}
}

func TestFitBox(t *testing.T) {
	for _, tc := range []struct {
		w, h, cols, rows int
		wantW, wantR     int
	}{
		{1920, 1080, 80, 20, 71, 20},
		{320, 240, 40, 10, 27, 10},
		{100, 100, 10, 10, 10, 5},
		{0, 100, 10, 10, 0, 0},
		{100, 100, 0, 10, 0, 0},
	} {
		gw, gr := fitBox(tc.w, tc.h, tc.cols, tc.rows)
		if gw != tc.wantW || gr != tc.wantR {
			t.Errorf("fitBox(%d,%d,%d,%d) = (%d,%d), want (%d,%d)",
				tc.w, tc.h, tc.cols, tc.rows, gw, gr, tc.wantW, tc.wantR)
		}
	}
}

func TestKittyRender(t *testing.T) {
	r := &kittyRenderer{id: 99}
	out := r.RenderPNG(makePNG(t, 160, 120, color.RGBA{255, 0, 0, 255}), 40, 10)
	if !strings.HasPrefix(out, r.Clear()) {
		t.Error("expected clear sequence first")
	}
	if !strings.Contains(out, "\x1b_Ga=T,f=100,i=99,c=") {
		t.Errorf("missing transmit header: %q", out[:60])
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("missing ST terminator")
	}
	if strings.Contains(out, "m=1") {
		t.Error("small image should not be chunked")
	}
	if !strings.Contains(out, ",C=1;") {
		t.Error("image placement must leave the layout's cursor position unchanged")
	}
}

func TestKittyChunking(t *testing.T) {
	r := &kittyRenderer{id: 1}
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	rnd := rand.New(rand.NewSource(1))
	for i := range img.Pix {
		img.Pix[i] = byte(rnd.Intn(256))
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if len(buf.Bytes()) < kittyChunkSize {
		t.Fatalf("test PNG too small to force chunking: %d", buf.Len())
	}
	out := r.RenderPNG(buf.Bytes(), 10, 5)
	if out == "" {
		t.Fatal("expected output")
	}
	if n := strings.Count(out, "\x1b_G"); n < 4 {
		t.Errorf("expected >=4 APC sequences (clear + chunks), got %d", n)
	}
	if !strings.Contains(out, ",m=1;") || !strings.Contains(out, "\x1b_Gm=1;") || !strings.Contains(out, "\x1b_Gm=0;") {
		t.Error("expected chunked transfer with m=1 continuation and m=0 final")
	}
	if !strings.Contains(out, ",C=1,m=1;") {
		t.Error("chunked image placement must disable cursor movement in the first chunk")
	}
}

func TestKittyInvalidPNG(t *testing.T) {
	r := &kittyRenderer{id: 1}
	if out := r.RenderPNG([]byte("not a png"), 10, 5); out != "" {
		t.Errorf("expected empty output for invalid PNG, got %q", out)
	}
}

func TestHalfBlockRender(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			c := color.RGBA{0, 0, 255, 255}
			if y < 2 {
				c = color.RGBA{255, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out := halfBlockRenderer{}.RenderPNG(buf.Bytes(), 20, 10)
	if out == "" {
		t.Fatal("expected output")
	}
	lines := strings.Split(out, "\n")
	if len(lines) > 10 {
		t.Errorf("too many lines: %d", len(lines))
	}
	if !strings.Contains(out, "\u2580") {
		t.Error("missing half-block glyph")
	}
	if !strings.Contains(out, "\x1b[38;2;") || !strings.Contains(out, "\x1b[48;2;") {
		t.Error("missing truecolor sequences")
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Error("missing style reset")
	}
}

func TestHalfBlockInvalid(t *testing.T) {
	if out := (halfBlockRenderer{}).RenderPNG([]byte("nope"), 10, 5); out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
	if out := (halfBlockRenderer{}).RenderPNG(makePNG(t, 4, 4, color.White), 0, 5); out != "" {
		t.Error("expected empty output for zero cols")
	}
}
