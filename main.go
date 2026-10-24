package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	if err := parseInterspersed(flag.CommandLine, os.Args[1:]); err != nil {
		return err
	}

	if *showVersion {
		fmt.Println(version)
		return nil
	}

	tools, err := media.FindTools()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := media.CheckEncoders(ctx, tools); err != nil {
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
		arg, err := filepath.Abs(flag.Arg(0))
		if err != nil {
			return err
		}
		st, err := os.Stat(arg)
		if err != nil {
			return err
		}
		if st.IsDir() {
			cfg.Dir = arg
		} else {
			cfg.File = arg
			cfg.Dir = filepath.Dir(arg)
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

// parseInterspersed allows options before or after the positional path.
// A bare -- ends option parsing, including for paths beginning with a dash.
func parseInterspersed(fs *flag.FlagSet, args []string) error {
	var options, paths []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
			continue
		}
		options = append(options, arg)
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		option := fs.Lookup(name)
		if option == nil || hasValue {
			continue
		}
		boolean, ok := option.Value.(interface{ IsBoolFlag() bool })
		if ok && boolean.IsBoolFlag() {
			continue
		}
		if i+1 == len(args) {
			return fmt.Errorf("flag needs an argument: %s", arg)
		}
		i++
		options = append(options, args[i])
	}
	options = append(options, "--")
	return fs.Parse(append(options, paths...))
}
