package checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDeclaredV2Corpus(t *testing.T) {
	root := filepath.Join("..", "..")
	phase, err := LoadPhase(filepath.Join(root, ".gooo", "capability-effect-checker.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	if phase.Version != 2 || phase.Denominator != 12 || len(phase.Cells) != 12 {
		t.Fatalf("v2 authority shape = version %d, denominator %d, cells %d", phase.Version, phase.Denominator, len(phase.Cells))
	}
	want := map[string]string{
		"exact-attenuation":             DecisionClosed,
		"zero-effect-generated":         DecisionClosed,
		"caller-owned-output":           DecisionClosed,
		"pinned-network-read":           DecisionClosed,
		"missing-grant":                 DecisionUnknown,
		"missing-call-edge":             DecisionUnknown,
		"unknown-generated-effect":      DecisionUnknown,
		"missing-path-scope":            DecisionUnknown,
		"repo-write-amplification":      DecisionRefuted,
		"remote-mutation-amplification": DecisionRefuted,
		"destructive-ancestor-delete":   DecisionRefuted,
		"forged-grant":                  DecisionRefuted,
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

func TestExactVectorsAndPrecedence(t *testing.T) {
	root := filepath.Join("..", "..")
	phase, err := LoadPhase(filepath.Join(root, ".gooo", "capability-effect-checker.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"exact-attenuation":             {"GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"zero-effect-generated":         {"GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"caller-owned-output":           {"GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"pinned-network-read":           {"NETWORK_READ_PINNED", "READ_INPUT"},
		"missing-grant":                 {"GENERATE_CALLER_OUTPUT", "NETWORK_READ_PINNED", "READ_INPUT"},
		"missing-call-edge":             {"GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"unknown-generated-effect":      {"GENERATED_EFFECT_UNKNOWN", "GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"missing-path-scope":            {"GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"repo-write-amplification":      {"GENERATE_CALLER_OUTPUT", "READ_INPUT", "REPOSITORY_WRITE"},
		"remote-mutation-amplification": {"GENERATE_CALLER_OUTPUT", "READ_INPUT", "REMOTE_MUTATION"},
		"destructive-ancestor-delete":   {"DESTRUCTIVE_DELETE", "GENERATE_CALLER_OUTPUT", "READ_INPUT"},
		"forged-grant":                  {"GENERATE_CALLER_OUTPUT", "READ_INPUT"},
	}
	for _, input := range phase.Cases {
		fixture, err := LoadFixture(filepath.Join(root, "examples/corpus", input.Source))
		if err != nil {
			t.Fatal(err)
		}
		result := EvaluateCase(phase, input, fixture)
		if expected, ok := want[input.ID]; ok && !reflect.DeepEqual(result.InferredEffects, expected) {
			t.Fatalf("%s inferred effects = %#v, want %#v", input.ID, result.InferredEffects, expected)
		}
	}
	if !reflect.DeepEqual(phase.Precedence, []string{DecisionRefuted, DecisionUnknown, DecisionClosed}) || phase.FixedPoint != "explicit" {
		t.Fatalf("precedence/fixed point = %#v/%q", phase.Precedence, phase.FixedPoint)
	}
}

func TestStageAndOriginBoundaries(t *testing.T) {
	root := filepath.Join("..", "..")
	phase, err := LoadPhase(filepath.Join(root, ".gooo", "capability-effect-checker.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := LoadFixture(filepath.Join(root, "examples/corpus", "destructive-ancestor-delete.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	result := EvaluateCase(phase, phase.Cases[10], fixture)
	if result.Decision != DecisionRefuted || !hasPathIssue(result, "destructive_ancestor_delete", []string{"metacode:generator", "generated:generator", "metacode:cleanup"}) {
		t.Fatalf("destructive ancestor result = %#v", result)
	}
	fixture, err = LoadFixture(filepath.Join(root, "examples/corpus", "forged-grant.gooo"))
	if err != nil {
		t.Fatal(err)
	}
	result = EvaluateCase(phase, phase.Cases[11], fixture)
	if result.Decision != DecisionRefuted || !hasPathIssue(result, "forged_grant", []string{"metacode:generator", "generated:generator"}) {
		t.Fatalf("forged grant result = %#v", result)
	}
}

func TestUnknownHasExactlySixFields(t *testing.T) {
	if reflect.TypeOf(Unknown{}).NumField() != 6 {
		t.Fatal("UNKNOWN must remain a six-field frontier")
	}
	data, err := json.Marshal(Unknown{Stage: "a", Step: "b", Reason: "c", UnknownClass: "d", NextOperation: "e", BlockedBy: []string{"f"}})
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	if len(object) != 6 {
		t.Fatalf("UNKNOWN JSON fields = %d", len(object))
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
	for _, fragment := range []string{"type EffectSet", "AllowsAttenuation", "NETWORK_READ_PINNED", "REMOTE_MUTATION", "DESTRUCTIVE_DELETE", "GENERATE_CALLER_OUTPUT"} {
		if !containsString(generated, fragment) {
			t.Fatalf("generated checker is missing %q", fragment)
		}
	}
}

func hasPathIssue(result CaseResult, issue string, path []string) bool {
	for _, candidate := range result.OffendingCallPaths {
		if candidate.Issue == issue && reflect.DeepEqual(candidate.Path, path) {
			return true
		}
	}
	return false
}

func containsString(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
