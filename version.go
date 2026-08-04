package main

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	fmt.Printf("pageshot %s (commit %s, built %s)\n", version, commit, date)
}
