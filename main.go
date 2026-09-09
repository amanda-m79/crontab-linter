package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: cronlint <crontab-file>")
		os.Exit(2)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "cronlint:", err)
		os.Exit(1)
	}
	defer f.Close()

	findings := LintReader(f)
	for _, fnd := range findings {
		fmt.Println(fnd.String())
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}
