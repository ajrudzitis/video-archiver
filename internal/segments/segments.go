package segments

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SidecarVersion = 1

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Title string  `json:"title"`
}

func (s Segment) Duration() float64 { return s.End - s.Start }

func (s Segment) Valid() error {
	if s.Title == "" {
		return fmt.Errorf("segment title must not be empty")
	}
	if s.Start < 0 {
		return fmt.Errorf("segment start must not be negative")
	}
	if s.End <= s.Start {
		return fmt.Errorf("segment end must be after start")
	}
	return nil
}

type File struct {
	Version   int       `json:"version"`
	Source    string    `json:"source"`
	Duration  float64   `json:"duration"`
	OutputDir string    `json:"output_dir,omitempty"`
	Segments  []Segment `json:"segments"`
}

func SidecarPath(sourcePath string) string {
	return sourcePath + ".segments.json"
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &f, nil
}

func (f *File) Save(path string) error {
	f.Version = SidecarVersion
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".segments-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (f *File) Sorted() []Segment {
	out := make([]Segment, len(f.Segments))
	copy(out, f.Segments)
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

func (f *File) HasDuplicateTitle(title string, skip int) bool {
	for i, s := range f.Segments {
		if i == skip {
			continue
		}
		if strings.EqualFold(s.Title, title) {
			return true
		}
	}
	return false
}

var filenameReplacer = strings.NewReplacer(
	"/", " ", "\\", " ", ":", " ", "*", " ", "?", " ",
	`"`, " ", "<", " ", ">", " ", "|", " ",
)

func SanitizeTitle(title string) string {
	var b strings.Builder
	for _, r := range filenameReplacer.Replace(title) {
		if r < 0x20 || r == 0x7f {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	s := strings.TrimSpace(b.String())
	s = strings.TrimLeft(s, ".")
	s = strings.TrimRight(s, ".")
	return strings.TrimSpace(s)
}

func OutputName(title string) string {
	return SanitizeTitle(title) + ".mp4"
}
