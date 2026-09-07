package media

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Tools struct {
	FFmpeg  string
	FFprobe string
}

func FindTools() (Tools, error) {
	var t Tools
	var err error
	if t.FFmpeg, err = exec.LookPath("ffmpeg"); err != nil {
		return t, fmt.Errorf("ffmpeg not found on PATH (try: nix-shell -p ffmpeg)")
	}
	if t.FFprobe, err = exec.LookPath("ffprobe"); err != nil {
		return t, fmt.Errorf("ffprobe not found on PATH (try: nix-shell -p ffmpeg)")
	}
	return t, nil
}

// CheckEncoders verifies that the encoders needed for splitting are
// available in the ffmpeg build.
func CheckEncoders(ctx context.Context, tools Tools) error {
	out, err := exec.CommandContext(ctx, tools.FFmpeg, "-hide_banner", "-encoders").Output()
	if err != nil {
		return fmt.Errorf("cannot list ffmpeg encoders: %w", err)
	}
	for _, need := range []string{"libx264", "aac"} {
		if !bytes.Contains(out, []byte(" "+need+" ")) {
			return fmt.Errorf("this ffmpeg build lacks the %s encoder, which video-archiver needs for splitting", need)
		}
	}
	return nil
}
