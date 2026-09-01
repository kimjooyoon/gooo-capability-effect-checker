package checker

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
)

func LoadPhase(path string) (Phase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Phase{}, err
	}
	phase, err := ParsePhase(data)
	if err != nil {
		return Phase{}, fmt.Errorf("parse phase %s: %w", path, err)
	}
	return phase, nil
}

func ParsePhase(data []byte) (Phase, error) {
	phase := Phase{
		Schema:          "gooo/capability-effect-checker/v1",
		Classifications: make(map[string]string),
		Diagnostics:     make(map[string]Diagnostic),
	}
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	headerSeen := false
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if !headerSeen {
			if line != "gooo 1" {
				return Phase{}, fmt.Errorf("line %d: expected gooo 1", lineNumber)
			}
			headerSeen = true
			continue
		}
		if len(fields) == 0 {
			continue
		}
		key := fields[0]
		switch key {
		case "package", "namespace", "precedence", "unknown_fields", "risk_effects", "denominator", "plan":
			if seen[key] {
				return Phase{}, fmt.Errorf("line %d: duplicate %s", lineNumber, key)
			}
			seen[key] = true
		}
		switch key {
		case "package":
			if len(fields) != 2 {
				return Phase{}, fmt.Errorf("line %d: package expects one value", lineNumber)
			}
			phase.Package = fields[1]
		case "namespace":
			if len(fields) != 2 {
				return Phase{}, fmt.Errorf("line %d: namespace expects one value", lineNumber)
			}
			phase.Namespace = fields[1]
		case "effect":
			if len(fields) != 2 || fields[1] == "" {
				return Phase{}, fmt.Errorf("line %d: effect expects one value", lineNumber)
			}
			if contains(phase.Effects, fields[1]) {
				return Phase{}, fmt.Errorf("line %d: duplicate effect %q", lineNumber, fields[1])
			}
			phase.Effects = append(phase.Effects, fields[1])
		case "capability":
			if len(fields) != 2 || fields[1] == "" {
				return Phase{}, fmt.Errorf("line %d: capability expects one value", lineNumber)
			}
			if contains(phase.Capabilities, fields[1]) {
				return Phase{}, fmt.Errorf("line %d: duplicate capability %q", lineNumber, fields[1])
			}
			phase.Capabilities = append(phase.Capabilities, fields[1])
		case "precedence":
			if len(fields) != 4 {
				return Phase{}, fmt.Errorf("line %d: precedence expects three statuses", lineNumber)
			}
			phase.Precedence = append([]string(nil), fields[1:]...)
		case "unknown_fields":
			if len(fields) != 7 {
				return Phase{}, fmt.Errorf("line %d: unknown_fields expects six fields", lineNumber)
			}
			phase.UnknownFields = append([]string(nil), fields[1:]...)
		case "risk_effects":
			if len(fields) < 2 {
				return Phase{}, fmt.Errorf("line %d: risk_effects needs one or more effects", lineNumber)
			}
			phase.RiskEffects = append([]string(nil), fields[1:]...)
		case "classification":
			if len(fields) != 3 {
				return Phase{}, fmt.Errorf("line %d: classification expects key and status", lineNumber)
			}
			if _, exists := phase.Classifications[fields[1]]; exists {
				return Phase{}, fmt.Errorf("line %d: duplicate classification %q", lineNumber, fields[1])
			}
			phase.Classifications[fields[1]] = fields[2]
		case "diagnostic":
			if len(fields) != 8 {
				return Phase{}, fmt.Errorf("line %d: diagnostic expects seven values", lineNumber)
			}
			if _, exists := phase.Diagnostics[fields[1]]; exists {
				return Phase{}, fmt.Errorf("line %d: duplicate diagnostic %q", lineNumber, fields[1])
			}
			phase.Diagnostics[fields[1]] = Diagnostic{
				Stage: fields[2], Step: fields[3], Reason: fields[4],
				UnknownClass: fields[5], NextOperation: fields[6], BlockedBy: fields[7],
			}
		case "denominator":
			if len(fields) != 3 || fields[1] != "cases" {
				return Phase{}, fmt.Errorf("line %d: denominator expects cases and integer", lineNumber)
			}
			var value int
			if _, err := fmt.Sscan(fields[2], &value); err != nil || value < 1 {
				return Phase{}, fmt.Errorf("line %d: invalid denominator", lineNumber)
			}
			phase.Denominator = value
		case "case":
			if len(fields) != 6 {
				return Phase{}, fmt.Errorf("line %d: case expects id source expected root capability", lineNumber)
			}
			for _, item := range phase.Cases {
				if item.ID == fields[1] {
					return Phase{}, fmt.Errorf("line %d: duplicate case %q", lineNumber, fields[1])
				}
			}
			phase.Cases = append(phase.Cases, Case{ID: fields[1], Source: fields[2], Expected: fields[3], Root: fields[4], Capability: fields[5]})
		case "grant":
			if len(fields) < 4 {
				return Phase{}, fmt.Errorf("line %d: grant expects case capability effect...", lineNumber)
			}
			for _, item := range phase.Grants {
				if item.CaseID == fields[1] && item.Capability == fields[2] {
					return Phase{}, fmt.Errorf("line %d: duplicate grant %q/%q", lineNumber, fields[1], fields[2])
				}
			}
			effects := append([]string(nil), fields[3:]...)
			sort.Strings(effects)
			phase.Grants = append(phase.Grants, Grant{CaseID: fields[1], Capability: fields[2], Effects: effects})
		case "plan":
			if len(fields) != 4 || fields[1] != "parser" || fields[2] != "executor" || fields[3] != "emitter" {
				return Phase{}, fmt.Errorf("line %d: plan must declare parser executor emitter", lineNumber)
			}
			phase.Plan = append([]string(nil), fields[1:]...)
		case "generation":
			if len(fields) < 3 {
				return Phase{}, fmt.Errorf("line %d: generation expects a subcommand", lineNumber)
			}
			switch fields[1] {
			case "output":
				if len(fields) != 3 {
					return Phase{}, fmt.Errorf("line %d: generation output expects one value", lineNumber)
				}
				phase.Generation.OutputBoundary = fields[2]
			case "steps":
				if len(fields) < 3 {
					return Phase{}, fmt.Errorf("line %d: generation steps needs one or more values", lineNumber)
				}
				phase.Generation.Steps = append([]string(nil), fields[2:]...)
			default:
				return Phase{}, fmt.Errorf("line %d: unknown generation subcommand %q", lineNumber, fields[1])
			}
		default:
			return Phase{}, fmt.Errorf("line %d: unknown phase key %q", lineNumber, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return Phase{}, err
	}
	if !headerSeen {
		return Phase{}, fmt.Errorf("missing gooo 1 header")
	}
	if err := validatePhase(phase, seen); err != nil {
		return Phase{}, err
	}
	phase.Digest = digestBytes(data)
	return phase, nil
}

func LoadFixture(path string) (Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, err
	}
	fixture, err := ParseFixture(data)
	if err != nil {
		return Fixture{}, fmt.Errorf("parse fixture %s: %w", path, err)
	}
	return fixture, nil
}

func ParseFixture(data []byte) (Fixture, error) {
	fixture := Fixture{Schema: "gooo/capability-effect-checker/fixture/v1", Functions: make(map[string]Function)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	headerSeen := false
	scenarioSeen := false
	rootSeen := false
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if !headerSeen {
			if line != "gooo 1" {
				return Fixture{}, fmt.Errorf("line %d: expected gooo 1", lineNumber)
			}
			headerSeen = true
			continue
		}
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "scenario":
			if len(fields) != 2 || scenarioSeen {
				return Fixture{}, fmt.Errorf("line %d: invalid or duplicate scenario", lineNumber)
			}
			fixture.Scenario, scenarioSeen = fields[1], true
		case "root":
			if len(fields) != 2 || rootSeen {
				return Fixture{}, fmt.Errorf("line %d: invalid or duplicate root", lineNumber)
			}
			fixture.Root, rootSeen = fields[1], true
		case "function":
			if len(fields) != 2 || fields[1] == "" {
				return Fixture{}, fmt.Errorf("line %d: function expects one value", lineNumber)
			}
			if _, exists := fixture.Functions[fields[1]]; exists {
				return Fixture{}, fmt.Errorf("line %d: duplicate function %q", lineNumber, fields[1])
			}
			fixture.Functions[fields[1]] = Function{Name: fields[1]}
		case "direct":
			if len(fields) < 3 {
				return Fixture{}, fmt.Errorf("line %d: direct expects function effect...", lineNumber)
			}
			function := ensureFunction(fixture.Functions, fields[1])
			if len(function.DirectEffects) > 0 {
				return Fixture{}, fmt.Errorf("line %d: duplicate direct effects for %q", lineNumber, fields[1])
			}
			function.DirectEffects = append([]string(nil), fields[2:]...)
			sort.Strings(function.DirectEffects)
			fixture.Functions[fields[1]] = function
		case "call":
			if len(fields) != 3 {
				return Fixture{}, fmt.Errorf("line %d: call expects caller callee", lineNumber)
			}
			function := ensureFunction(fixture.Functions, fields[1])
			if contains(function.Calls, fields[2]) {
				return Fixture{}, fmt.Errorf("line %d: duplicate call %q -> %q", lineNumber, fields[1], fields[2])
			}
			function.Calls = append(function.Calls, fields[2])
			sort.Strings(function.Calls)
			fixture.Functions[fields[1]] = function
		case "oracle":
			if len(fields) != 4 {
				return Fixture{}, fmt.Errorf("line %d: oracle expects function name status", lineNumber)
			}
			function := ensureFunction(fixture.Functions, fields[1])
			function.Oracles = append(function.Oracles, Oracle{Name: fields[2], Status: fields[3]})
			sort.Slice(function.Oracles, func(i, j int) bool { return function.Oracles[i].Name < function.Oracles[j].Name })
			fixture.Functions[fields[1]] = function
		default:
			return Fixture{}, fmt.Errorf("line %d: unknown fixture key %q", lineNumber, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Fixture{}, err
	}
	if !headerSeen || !scenarioSeen || !rootSeen || fixture.Root == "" {
		return Fixture{}, fmt.Errorf("fixture requires header, scenario, and root")
	}
	if _, exists := fixture.Functions[fixture.Root]; !exists {
		return Fixture{}, fmt.Errorf("root function %q is not declared", fixture.Root)
	}
	return fixture, nil
}

func validatePhase(phase Phase, seen map[string]bool) error {
	if phase.Package == "" || phase.Namespace == "" {
		return fmt.Errorf("phase requires package and namespace")
	}
	if !seen["precedence"] || !sameSlice(phase.Precedence, []string{DecisionRefuted, DecisionUnknown, DecisionClosed}) {
		return fmt.Errorf("phase precedence must be REFUTED, UNKNOWN, CLOSED")
	}
	if !seen["unknown_fields"] || !sameSlice(phase.UnknownFields, []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"}) {
		return fmt.Errorf("phase must declare the six UNKNOWN fields")
	}
	if len(phase.Effects) == 0 || len(phase.Capabilities) == 0 {
		return fmt.Errorf("phase requires effects and capabilities")
	}
	if phase.Denominator != len(phase.Cases) || phase.Denominator == 0 {
		return fmt.Errorf("phase denominator does not match cases")
	}
	if !seen["plan"] || len(phase.Plan) != 3 || phase.Generation.OutputBoundary == "" || len(phase.Generation.Steps) == 0 {
		return fmt.Errorf("phase generation plan is incomplete")
	}
	for _, effect := range phase.RiskEffects {
		if !contains(phase.Effects, effect) {
			return fmt.Errorf("risk effect %q is outside effect vocabulary", effect)
		}
	}
	for _, item := range phase.Cases {
		if !contains(phase.Precedence, item.Expected) {
			return fmt.Errorf("case %q has unsupported expected decision %q", item.ID, item.Expected)
		}
		if !contains(phase.Capabilities, item.Capability) {
			return fmt.Errorf("case %q uses undeclared capability %q", item.ID, item.Capability)
		}
	}
	for _, grant := range phase.Grants {
		if !containsCase(phase.Cases, grant.CaseID) {
			return fmt.Errorf("grant references unknown case %q", grant.CaseID)
		}
		if !contains(phase.Capabilities, grant.Capability) {
			return fmt.Errorf("grant references unknown capability %q", grant.Capability)
		}
		for _, effect := range grant.Effects {
			if !contains(phase.Effects, effect) {
				return fmt.Errorf("grant references unknown effect %q", effect)
			}
		}
	}
	for name := range phase.Classifications {
		if !contains(phase.Precedence, phase.Classifications[name]) {
			return fmt.Errorf("classification %q has unsupported status %q", name, phase.Classifications[name])
		}
		if _, ok := phase.Diagnostics[name]; !ok {
			return fmt.Errorf("classification %q has no diagnostic", name)
		}
	}
	return nil
}

func ensureFunction(functions map[string]Function, name string) Function {
	if function, ok := functions[name]; ok {
		return function
	}
	return Function{Name: name}
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsCase(values []Case, wanted string) bool {
	for _, value := range values {
		if value.ID == wanted {
			return true
		}
	}
	return false
}

func sameSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for _, value := range left {
		if !contains(right, value) {
			return false
		}
	}
	return true
}

func sameSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
