package media

import (
	"context"
	"strings"
	"testing"
)

func TestCutCommandArgs(t *testing.T) {
	tools := Tools{FFmpeg: "/bin/ffmpeg"}
	j := Job{Source: "/in/movie.avi", Output: "/out/title.mp4", Start: 12.5, End: 72.25, Title: "title"}
	cmd := CutCommand(context.Background(), tools, j)
	args := strings.Join(cmd.Args, " ")
	for _, want := range []string{
		"-ss 12.500000000",
		"-i /in/movie.avi",
		"-t 59.750000000",
		"-map 0:v:0",
		"-map 0:a?",
		"-c:v libx264",
		"-crf 18",
		"-c:a aac",
		"-movflags +faststart",
		"-progress pipe:1",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args missing %q: %s", want, args)
		}
	}
	if cmd.Args[len(cmd.Args)-1] != "/out/title.mp4" {
		t.Errorf("output not last arg: %s", args)
	}
}

func TestParseProgressLine(t *testing.T) {
	if s, ok := ParseProgressLine("out_time_us=1500000"); !ok || s != 1.5 {
		t.Errorf("out_time_us: %v %v", s, ok)
	}
	if s, ok := ParseProgressLine("out_time_ms=2500"); !ok || s != 2.5 {
		t.Errorf("out_time_ms: %v %v", s, ok)
	}
	if _, ok := ParseProgressLine("frame=123"); ok {
		t.Error("unexpected match")
	}
	if _, ok := ParseProgressLine("out_time_us=N/A"); ok {
		t.Error("N/A should not parse")
	}
}
