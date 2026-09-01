package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type GenerateOptions struct {
	PhasePath  string
	CorpusRoot string
	OutputDir  string
	SourceRoot string
}

func Generate(options GenerateOptions) (RunReport, error) {
	if options.PhasePath == "" || options.CorpusRoot == "" || options.OutputDir == "" || options.SourceRoot == "" {
		return RunReport{}, fmt.Errorf("phase, corpus-root, output, and source-root are required")
	}
	phase, err := LoadPhase(options.PhasePath)
	if err != nil {
		return RunReport{}, err
	}
	outputDir, err := filepath.Abs(options.OutputDir)
	if err != nil {
		return RunReport{}, err
	}
	sourceRoot, err := filepath.Abs(options.SourceRoot)
	if err != nil {
		return RunReport{}, err
	}
	if pathWithin(sourceRoot, outputDir) {
		return RunReport{}, fmt.Errorf("output directory must be caller-owned and outside source repository")
	}
	corpusRoot, err := filepath.Abs(options.CorpusRoot)
	if err != nil {
		return RunReport{}, err
	}
	semantic, semanticDigest, err := BuildSemanticIR(phase)
	if err != nil {
		return RunReport{}, err
	}

	results := make([]CaseResult, 0, len(phase.Cases))
	for _, input := range phase.Cases {
		fixturePath, pathErr := safeJoin(corpusRoot, input.Source)
		if pathErr != nil {
			return RunReport{}, pathErr
		}
		fixture, fixtureErr := LoadFixture(fixturePath)
		if fixtureErr != nil {
			return RunReport{}, fixtureErr
		}
		results = append(results, EvaluateCase(phase, input, fixture))
	}

	summary := Summary{Generated: len(results)}
	for _, result := range results {
		switch result.Decision {
		case DecisionClosed:
			summary.Closed++
		case DecisionUnknown:
			summary.Unknown++
		case DecisionRefuted:
			summary.Refuted++
		}
		if result.Decision != result.Expected {
			summary.Failed++
		}
	}
	generated := EmitChecker(phase, semantic, semanticDigest)
	if err := os.MkdirAll(filepath.Join(outputDir, "generated"), 0o755); err != nil {
		return RunReport{}, err
	}
	semanticBytes, err := PrettyJSON(semantic)
	if err != nil {
		return RunReport{}, err
	}
	semanticPath := filepath.Join(outputDir, "semantic-ir.json")
	generatedPath := filepath.Join(outputDir, "generated", "checker.go")
	if err := os.WriteFile(semanticPath, semanticBytes, 0o644); err != nil {
		return RunReport{}, err
	}
	if err := os.WriteFile(generatedPath, generated, 0o644); err != nil {
		return RunReport{}, err
	}
	generatedArtifact := artifact("generated/checker.go", generated)
	report := RunReport{
		Schema: "gooo/capability-effect-checker/run-report/v1", PhaseDigest: phase.Digest,
		SemanticIRDigest: semanticDigest, Precedence: append([]string(nil), phase.Precedence...), Generation: phase.Generation, Summary: summary, Cases: results,
		Authority:          Authority{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0},
		GeneratedArtifacts: []Artifact{generatedArtifact},
		Artifacts:          []Artifact{artifact("semantic-ir.json", semanticBytes), generatedArtifact},
	}
	reportBytes, err := PrettyJSON(report)
	if err != nil {
		return RunReport{}, err
	}
	human := RenderReport(report)
	if err := os.WriteFile(filepath.Join(outputDir, "run-report.json"), reportBytes, 0o644); err != nil {
		return RunReport{}, err
	}
	if err := os.WriteFile(filepath.Join(outputDir, "human-report.md"), []byte(human), 0o644); err != nil {
		return RunReport{}, err
	}
	if summary.Failed > 0 {
		return report, fmt.Errorf("%d corpus cases differed from declared expected decisions", summary.Failed)
	}
	return report, nil
}

func RenderReport(report RunReport) string {
	var out strings.Builder
	out.WriteString("# Gooo capability effect checker run\n\n")
	out.WriteString("- phase digest: `" + report.PhaseDigest + "`\n")
	out.WriteString("- semantic IR digest: `" + report.SemanticIRDigest + "`\n")
	out.WriteString("- decision precedence: `" + strings.Join(report.Precedence, " > ") + "`\n")
	out.WriteString("- generated cases: `" + fmt.Sprint(report.Summary.Generated) + "`\n")
	out.WriteString("- CLOSED: `" + fmt.Sprint(report.Summary.Closed) + "`; UNKNOWN: `" + fmt.Sprint(report.Summary.Unknown) + "`; REFUTED: `" + fmt.Sprint(report.Summary.Refuted) + "`\n")
	out.WriteString("- repository writes: `0`; local test executions: `0`; cross-project required gates: `0`\n\n")
	for _, result := range report.Cases {
		out.WriteString("## " + result.ID + " — " + result.Decision + "\n\n")
		out.WriteString("- exact inferred effects: `" + strings.Join(result.InferredEffects, ", ") + "`\n")
		out.WriteString("- declared root grant: `" + strings.Join(result.DeclaredRootEffects, ", ") + "`\n")
		if len(result.OffendingCallPaths) == 0 {
			out.WriteString("- minimum offending call path: none\n")
		} else {
			out.WriteString("- minimum offending call paths:\n")
			for _, path := range result.OffendingCallPaths {
				out.WriteString("  - " + path.Issue + " " + path.Effect + ": `" + strings.Join(path.Path, " -> ") + "`\n")
			}
		}
		for _, unknown := range result.Unknowns {
			out.WriteString("- UNKNOWN frontier: `" + unknown.Stage + "/" + unknown.Step + "` — " + unknown.Reason + " — blocked by `" + strings.Join(unknown.BlockedBy, ",") + "`\n")
		}
		for _, refutation := range result.Refutations {
			out.WriteString("- REFUTED: `" + refutation.Stage + "/" + refutation.Step + "` — " + refutation.Reason + " — `" + refutation.Counterexample + "`\n")
		}
		out.WriteString("\n")
	}
	return out.String()
}

func artifact(path string, data []byte) Artifact {
	return Artifact{Path: path, Bytes: len(data), Digest: digestBytes(data)}
}

func safeJoin(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("absolute fixture path is not allowed")
	}
	joined, err := filepath.Abs(filepath.Join(root, relative))
	if err != nil {
		return "", err
	}
	if !pathWithin(root, joined) {
		return "", fmt.Errorf("fixture path escapes corpus root")
	}
	return joined, nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func sortedCaseResults(results []CaseResult) []CaseResult {
	copyResults := append([]CaseResult(nil), results...)
	sort.Slice(copyResults, func(i, j int) bool { return copyResults[i].ID < copyResults[j].ID })
	return copyResults
}
