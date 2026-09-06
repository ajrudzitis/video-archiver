package media

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireTools(t *testing.T) Tools {
	t.Helper()
	tools, err := FindTools()
	if err != nil {
		t.Skipf("ffmpeg not available: %v", err)
	}
	return tools
}

func hasEncoder(t *testing.T, tools Tools, name string) bool {
	t.Helper()
	out, err := exec.Command(tools.FFmpeg, "-hide_banner", "-encoders").Output()
	if err != nil {
		t.Skipf("cannot list encoders: %v", err)
	}
	return bytes.Contains(out, []byte(" "+name+" "))
}

func makeTestVideo(t *testing.T, tools Tools, dir string) string {
	t.Helper()
	if !hasEncoder(t, tools, "libx264") || !hasEncoder(t, tools, "aac") {
		t.Skip("libx264 or aac encoder not available")
	}
	path := filepath.Join(dir, "src.mp4")
	cmd := exec.Command(tools.FFmpeg,
		"-hide_banner", "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=duration=4:size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=4",
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:a", "aac", "-shortest", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate test video: %v: %s", err, out)
	}
	return path
}

func TestIntegrationProbeFrameCut(t *testing.T) {
	tools := requireTools(t)
	dir := t.TempDir()
	src := makeTestVideo(t, tools, dir)
	ctx := context.Background()

	info, err := Probe(ctx, tools, src)
	if err != nil {
		t.Fatal(err)
	}
	if info.Duration < 3.9 || info.Duration > 4.1 {
		t.Errorf("duration = %v, want ~4", info.Duration)
	}
	if info.Width != 320 || info.Height != 240 || info.FPS != 25 {
		t.Errorf("info = %+v", info)
	}
	if info.AudioCodec == "" {
		t.Error("expected audio codec")
	}

	png, err := ExtractFrame(ctx, tools, src, 1.5, 160)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(png, []byte("\x89PNG")) {
		t.Error("frame output is not PNG")
	}

	out := filepath.Join(dir, "seg.mp4")
	job := Job{Source: src, Output: out, Start: 1, End: 3, Title: "seg"}
	cmd := CutCommand(ctx, tools, job)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("cut: %v: %s", err, errBuf.String())
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		t.Fatalf("output missing or empty: %v", err)
	}

	cutInfo, err := Probe(ctx, tools, out)
	if err != nil {
		t.Fatal(err)
	}
	if cutInfo.Duration < 1.9 || cutInfo.Duration > 2.1 {
		t.Errorf("cut duration = %v, want ~2", cutInfo.Duration)
	}
	if !strings.HasSuffix(cutInfo.VideoCodec, "264") {
		t.Errorf("cut video codec = %s", cutInfo.VideoCodec)
	}
}
