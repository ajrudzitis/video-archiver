package media

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func ExtractFrame(ctx context.Context, tools Tools, path string, at float64, width int) ([]byte, error) {
	if at < 0 {
		at = 0
	}
	args := []string{
		"-v", "error",
		"-ss", strconv.FormatFloat(at, 'f', 3, 64),
		"-i", path,
		"-frames:v", "1",
	}
	if width > 0 {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:-2", width))
	}
	args = append(args, "-f", "image2", "-c:v", "png", "pipe:1")
	cmd := exec.CommandContext(ctx, tools.FFmpeg, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg frame: %v: %s", err, strings.TrimSpace(errBuf.String()))
	}
	return out.Bytes(), nil
}
