package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	strict := flag.Bool("strict", false, "exit non-zero on style-only findings too")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: cronlint [--strict] <crontab-file>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "cronlint:", err)
		os.Exit(1)
	}
	defer f.Close()

	findings := LintReader(f)
	for _, fnd := range findings {
		fmt.Println(fnd.String())
	}
	if HasErrors(findings) || (*strict && len(findings) > 0) {
		os.Exit(1)
	}
}
