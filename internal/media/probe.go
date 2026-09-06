package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Path       string
	Duration   float64
	Width      int
	Height     int
	FPS        float64
	VideoCodec string
	AudioCodec string
	Size       int64
}

type probeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
	RFrameRate   string `json:"r_frame_rate"`
}

type probeFormat struct {
	Duration string `json:"duration"`
}

type probeOutput struct {
	Streams []probeStream `json:"streams"`
	Format  probeFormat   `json:"format"`
}

func Probe(ctx context.Context, tools Tools, path string) (Info, error) {
	cmd := exec.CommandContext(ctx, tools.FFprobe,
		"-v", "error", "-print_format", "json",
		"-show_format", "-show_streams", path)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return Info{}, fmt.Errorf("ffprobe: %v: %s", err, strings.TrimSpace(errBuf.String()))
	}
	info, err := parseProbe(out.Bytes())
	if err != nil {
		return Info{}, err
	}
	info.Path = path
	if st, err := os.Stat(path); err == nil {
		info.Size = st.Size()
	}
	return info, nil
}

func parseProbe(data []byte) (Info, error) {
	var p probeOutput
	if err := json.Unmarshal(data, &p); err != nil {
		return Info{}, fmt.Errorf("parse ffprobe output: %w", err)
	}
	var info Info
	if p.Format.Duration != "" {
		d, err := strconv.ParseFloat(p.Format.Duration, 64)
		if err != nil {
			return Info{}, fmt.Errorf("parse duration %q: %w", p.Format.Duration, err)
		}
		info.Duration = d
	}
	for _, s := range p.Streams {
		switch s.CodecType {
		case "video":
			if info.VideoCodec != "" {
				continue
			}
			info.VideoCodec = s.CodecName
			info.Width = s.Width
			info.Height = s.Height
			info.FPS = parseRate(s.AvgFrameRate)
			if info.FPS <= 0 {
				info.FPS = parseRate(s.RFrameRate)
			}
		case "audio":
			if info.AudioCodec == "" {
				info.AudioCodec = s.CodecName
			}
		}
	}
	if info.VideoCodec == "" {
		return Info{}, fmt.Errorf("no video stream found")
	}
	return info, nil
}

func parseRate(s string) float64 {
	if s == "" {
		return 0
	}
	if num, den, ok := strings.Cut(s, "/"); ok {
		n, err1 := strconv.ParseFloat(num, 64)
		d, err2 := strconv.ParseFloat(den, 64)
		if err1 != nil || err2 != nil || d == 0 {
			return 0
		}
		return n / d
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
