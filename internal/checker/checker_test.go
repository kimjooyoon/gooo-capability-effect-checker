package checker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeclaredCorpus(t *testing.T) {
	root := filepath.Join("..", "..")
	phase, err := LoadPhase(filepath.Join(root, ".gooo", "capability-effect-checker.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"safe-generator":              DecisionClosed,
		"repository-write-escalation": DecisionRefuted,
		"indirect-missing-grant":      DecisionUnknown,
		"indirect-repository-write":   DecisionRefuted,
		"external-oracle":             DecisionUnknown,
	}
	if len(phase.Cases) != len(want) || phase.Denominator != len(want) {
		t.Fatalf("case denominator = %d/%d, want %d", len(phase.Cases), phase.Denominator, len(want))
	}
	for _, input := range phase.Cases {
		fixture, err := LoadFixture(filepath.Join(root, "examples/corpus", input.Source))
		if err != nil {
			t.Fatal(err)
		}
		result := EvaluateCase(phase, input, fixture)
		if result.Decision != want[input.ID] || result.Decision != input.Expected {
			t.Fatalf("%s decision = %s, want %s", input.ID, result.Decision, want[input.ID])
		}
		for _, unknown := range result.Unknowns {
			if unknown.Stage == "" || unknown.Step == "" || unknown.Reason == "" || unknown.UnknownClass == "" || unknown.NextOperation == "" || len(unknown.BlockedBy) == 0 {
				t.Fatalf("%s has incomplete UNKNOWN: %#v", input.ID, unknown)
			}
		}
	}
}

func TestExactEffectsAndMinimumPaths(t *testing.T) {
	root := filepath.Join("..", "..")
	phase, err := LoadPhase(filepath.Join(root, ".gooo", "capability-effect-checker.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := LoadFixture(filepath.Join(root, "examples/corpus/indirect-repository-write.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	result := EvaluateCase(phase, phase.Cases[3], fixture)
	if !reflect.DeepEqual(result.InferredEffects, []string{"READ_INPUT", "REPOSITORY_WRITE", "WRITE_CALLER_OUTPUT"}) {
		t.Fatalf("inferred effects = %#v", result.InferredEffects)
	}
	if len(result.OffendingCallPaths) == 0 || !reflect.DeepEqual(result.OffendingCallPaths[0].Path, []string{"generator"}) {
		t.Fatalf("minimum offending path = %#v", result.OffendingCallPaths)
	}
}

func TestOutputBoundary(t *testing.T) {
	if !pathWithin("/tmp/input", "/tmp/input/nested/output") {
		t.Fatal("nested output should be inside input")
	}
	if pathWithin("/tmp/input", "/tmp/input-sibling/output") {
		t.Fatal("sibling output should not be inside input")
	}
}

func TestGeneratedCheckerIsEmitted(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".gooo", "capability-effect-checker.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	phase, err := ParsePhase(data)
	if err != nil {
		t.Fatal(err)
	}
	semantic, digest, err := BuildSemanticIR(phase)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(EmitChecker(phase, semantic, digest))
	for _, fragment := range []string{"func InferEffects", "func Check", "REPOSITORY_WRITE", "WRITE_CALLER_OUTPUT"} {
		if !containsString(generated, fragment) {
			t.Fatalf("generated checker is missing %q", fragment)
		}
	}
}

func containsString(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
