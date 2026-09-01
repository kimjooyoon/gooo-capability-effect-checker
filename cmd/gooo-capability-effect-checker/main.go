package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-capability-effect-checker/internal/checker"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "generate" {
		fmt.Fprintln(os.Stderr, "usage: gooo-capability-effect-checker generate --phase PATH --corpus-root PATH --out PATH --source-root PATH")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("generate", flag.ExitOnError)
	phasePath := flags.String("phase", "", "metacode phase path")
	corpusRoot := flags.String("corpus-root", "", "caller-provided corpus root")
	out := flags.String("out", "", "caller-owned output directory")
	sourceRoot := flags.String("source-root", "", "read-only input repository root")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	report, err := checker.Generate(checker.GenerateOptions{
		PhasePath: *phasePath, CorpusRoot: *corpusRoot, OutputDir: *out, SourceRoot: *sourceRoot,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("generated=%d closed=%d unknown=%d refuted=%d output=%s\n", report.Summary.Generated, report.Summary.Closed, report.Summary.Unknown, report.Summary.Refuted, *out)
}
