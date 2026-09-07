package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"video-archiver/internal/app"
	"video-archiver/internal/media"
)

var version = "0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "video-archiver:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		preview     = flag.String("preview", "auto", "frame preview mode: auto, kitty, halfblock, off")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: video-archiver [dir|file] [flags]\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return nil
	}

	tools, err := media.FindTools()
	if err != nil {
		return err
	}

	runtime := &app.Runtime{}
	cfg := app.Config{Preview: *preview, Tools: tools, Runtime: runtime}
	switch flag.NArg() {
	case 0:
		cfg.Dir, err = os.Getwd()
		if err != nil {
			return err
		}
	case 1:
		arg := flag.Arg(0)
		st, err := os.Stat(arg)
		if err != nil {
			return err
		}
		if st.IsDir() {
			cfg.Dir = arg
		} else {
			cfg.File = arg
		}
	default:
		flag.Usage()
		return fmt.Errorf("expected at most one argument")
	}

	p := tea.NewProgram(app.New(cfg), tea.WithAltScreen())
	runtime.SetProgram(p)
	_, err = p.Run()
	return err
}
