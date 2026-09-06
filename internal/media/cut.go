package media

import (
	"context"
	"os/exec"
	"strconv"
)

type Job struct {
	Source string
	Output string
	Start  float64
	End    float64
	Title  string
}

func (j Job) Duration() float64 { return j.End - j.Start }

func CutCommand(ctx context.Context, tools Tools, j Job) *exec.Cmd {
	return exec.CommandContext(ctx, tools.FFmpeg,
		"-hide_banner", "-nostats", "-v", "error", "-y",
		"-ss", strconv.FormatFloat(j.Start, 'f', 3, 64),
		"-i", j.Source,
		"-t", strconv.FormatFloat(j.Duration(), 'f', 3, 64),
		"-map", "0:v:0", "-map", "0:a?",
		"-c:v", "libx264", "-preset", "medium", "-crf", "18",
		"-c:a", "aac", "-b:a", "192k",
		"-movflags", "+faststart",
		"-progress", "pipe:1",
		j.Output,
	)
}
