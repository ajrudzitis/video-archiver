package media

import (
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
