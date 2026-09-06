package editor

import (
	"path/filepath"
	"testing"

	"video-archiver/internal/segments"
)

func TestBuildJobs(t *testing.T) {
	segs := []segments.Segment{
		{Start: 0, End: 10.5, Title: "opening"},
		{Start: 20, End: 30, Title: "bad/name"},
	}
	jobs := BuildJobs("/movies/holiday.avi", "/out", segs)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs", len(jobs))
	}
	if jobs[0].Output != filepath.Join("/out", "opening.mp4") {
		t.Errorf("job 0 output = %s", jobs[0].Output)
	}
	if jobs[1].Output != filepath.Join("/out", "bad name.mp4") {
		t.Errorf("job 1 output = %s", jobs[1].Output)
	}
	if jobs[0].Source != "/movies/holiday.avi" || jobs[0].Duration() != 10.5 {
		t.Errorf("job 0 = %+v", jobs[0])
	}
}
