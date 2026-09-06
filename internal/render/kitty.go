package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
)

const kittyChunkSize = 4096

type kittyRenderer struct {
	id int
}

func (k *kittyRenderer) Kind() Kind { return KindKitty }

func (k *kittyRenderer) RenderPNG(png []byte, cols, rows int) string {
	w, h, ok := decodeSize(png)
	if !ok {
		return ""
	}
	cw, ch := fitBox(w, h, cols, rows)
	if cw == 0 || ch == 0 {
		return ""
	}

	b64 := base64.StdEncoding.EncodeToString(png)
	var buf bytes.Buffer
	buf.WriteString(k.Clear())

	opts := fmt.Sprintf("a=T,f=100,i=%d,c=%d,r=%d", k.id, cw, ch)
	for off := 0; off < len(b64); off += kittyChunkSize {
		end := min(off+kittyChunkSize, len(b64))
		chunk := b64[off:end]
		last := end == len(b64)
		first := off == 0
		buf.WriteString("\x1b_G")
		switch {
		case first && !last:
			buf.WriteString(opts + ",m=1")
		case first:
			buf.WriteString(opts)
		case !last:
			buf.WriteString("m=1")
		default:
			buf.WriteString("m=0")
		}
		buf.WriteString(";")
		buf.WriteString(chunk)
		buf.WriteString("\x1b\\")
	}
	return buf.String()
}

func (k *kittyRenderer) Clear() string {
	return "\x1b_Ga=d,d=i,i=" + strconv.Itoa(k.id) + "\x1b\\"
}

type noneRenderer struct{}

func (noneRenderer) Kind() Kind                        { return KindNone }
func (noneRenderer) RenderPNG([]byte, int, int) string { return "" }
func (noneRenderer) Clear() string                     { return "" }
