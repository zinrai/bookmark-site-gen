package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	exitFailure   = 1
	exitUnsettled = 2
)

func main() {
	var (
		output      = flag.String("output", "", "output PNG path (required)")
		force       = flag.Bool("force", false, "capture even if the output file already exists")
		timeout     = flag.Duration("timeout", 10*time.Second, "time budget for the page to settle")
		viewport    = flag.String("viewport", "1280x800", "browser viewport as WIDTHxHEIGHT")
		resize      = flag.String("size", "", "resize the capture to WIDTHxHEIGHT")
		noSandbox   = flag.Bool("no-sandbox", false, "disable Chrome sandbox (for CI environments)")
		showVersion = flag.Bool("version", false, "Print version information and exit")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <url>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExit status:\n")
		fmt.Fprintf(os.Stderr, "  0  captured, or skipped because the output already exists\n")
		fmt.Fprintf(os.Stderr, "  1  failed\n")
		fmt.Fprintf(os.Stderr, "  2  captured before the page settled\n")
	}

	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	if flag.NArg() != 1 || *output == "" {
		flag.Usage()
		os.Exit(exitFailure)
	}

	url := flag.Arg(0)

	if !*force && exists(*output) {
		fmt.Fprintf(os.Stderr, "Skipped: %s already exists\n", *output)
		os.Exit(0)
	}

	opts := Options{
		Timeout:   *timeout,
		NoSandbox: *noSandbox,
	}

	var err error
	opts.Viewport, err = parseSize(*viewport)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid -viewport: %v\n", err)
		os.Exit(exitFailure)
	}

	if *resize != "" {
		opts.Resize, err = parseSize(*resize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid -size: %v\n", err)
			os.Exit(exitFailure)
		}
	}

	data, settled, err := Capture(url, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to capture %s: %v\n", url, err)
		os.Exit(exitFailure)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write %s: %v\n", *output, err)
		os.Exit(exitFailure)
	}

	if !settled {
		fmt.Fprintf(os.Stderr, "Captured: %s before the page settled\n", *output)
		os.Exit(exitUnsettled)
	}

	fmt.Fprintf(os.Stderr, "Captured: %s\n", *output)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseSize(s string) (Size, error) {
	width, height, found := strings.Cut(s, "x")
	if !found {
		return Size{}, fmt.Errorf("%q is not WIDTHxHEIGHT", s)
	}

	w, err := strconv.Atoi(width)
	if err != nil || w <= 0 {
		return Size{}, fmt.Errorf("%q has an invalid width", s)
	}

	h, err := strconv.Atoi(height)
	if err != nil || h <= 0 {
		return Size{}, fmt.Errorf("%q has an invalid height", s)
	}

	return Size{Width: w, Height: h}, nil
}
