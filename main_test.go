package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestInterspersedFlags(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		preview string
		paths   []string
	}{
		{[]string{"movie.mp4", "-preview", "off"}, "off", []string{"movie.mp4"}},
		{[]string{"-preview", "kitty", "videos"}, "kitty", []string{"videos"}},
		{[]string{"videos", "--preview=halfblock"}, "halfblock", []string{"videos"}},
		{[]string{"--", "-movie.mp4"}, "auto", []string{"-movie.mp4"}},
	} {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		preview := fs.String("preview", "auto", "")
		fs.Bool("version", false, "")
		if err := parseInterspersed(fs, tc.args); err != nil {
			t.Fatal(err)
		}
		if *preview != tc.preview || !reflect.DeepEqual(fs.Args(), tc.paths) {
			t.Fatalf("%v: preview=%s paths=%v", tc.args, *preview, fs.Args())
		}
	}
}
func TestInterspersedInvalidFlags(t *testing.T) {
	for _, args := range [][]string{{"movie.mp4", "-preview"}, {"movie.mp4", "-unknown"}} {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		fs.String("preview", "auto", "")
		if err := parseInterspersed(fs, args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
