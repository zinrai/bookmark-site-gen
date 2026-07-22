package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		output      = flag.String("output", "public", "output directory")
		timeout     = flag.Duration("timeout", 10*time.Second, "screenshot timeout")
		dryRun      = flag.Bool("dry-run", false, "show changes without applying")
		noSandbox   = flag.Bool("no-sandbox", false, "disable Chrome sandbox (for CI environments)")
		showVersion = flag.Bool("version", false, "Print version information and exit")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <bookmarks.json>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	jsonPath := flag.Arg(0)

	bookmarks, err := LoadBookmarks(jsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load bookmarks: %v\n", err)
		os.Exit(1)
	}

	gen := &Generator{
		OutputDir: *output,
		Timeout:   *timeout,
		DryRun:    *dryRun,
		NoSandbox: *noSandbox,
	}

	if err := gen.Run(bookmarks); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
