package checker

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
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
		Schema:          "gooo/capability-effect-checker/v2",
		Version:         2,
		Classifications: make(map[string]string),
		Diagnostics:     make(map[string]Diagnostic),
		ProofQuotas:     make(map[string]int),
		IndicatorQuotas: make(map[string]int),
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
			if len(fields) != 2 || fields[0] != "gooo" {
				return Phase{}, fmt.Errorf("line %d: expected gooo 1 or gooo 2", lineNumber)
			}
			version, err := strconv.Atoi(fields[1])
			if err != nil || (version != 1 && version != 2) {
				return Phase{}, fmt.Errorf("line %d: unsupported gooo version", lineNumber)
			}
			phase.Version = version
			if version == 1 {
				phase.Schema = "gooo/capability-effect-checker/v1"
			}
			headerSeen = true
			continue
		}
		key := fields[0]
		switch key {
		case "package", "namespace", "precedence", "unknown_fields", "risk_effects", "denominator", "plan", "output_scope", "fixed_point":
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
		case "stage":
			if len(fields) != 2 || fields[1] == "" || contains(phase.Stages, fields[1]) {
				return Phase{}, fmt.Errorf("line %d: invalid or duplicate stage", lineNumber)
			}
			phase.Stages = append(phase.Stages, fields[1])
		case "stage_edge":
			if len(fields) != 3 {
				return Phase{}, fmt.Errorf("line %d: stage_edge expects from and to", lineNumber)
			}
			for _, edge := range phase.StageEdges {
				if edge.From == fields[1] && edge.To == fields[2] {
					return Phase{}, fmt.Errorf("line %d: duplicate stage edge", lineNumber)
				}
			}
			phase.StageEdges = append(phase.StageEdges, StageEdge{From: fields[1], To: fields[2]})
		case "output_scope":
			if len(fields) != 2 {
				return Phase{}, fmt.Errorf("line %d: output_scope expects one value", lineNumber)
			}
			phase.OutputScope = fields[1]
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
			value, err := strconv.Atoi(fields[2])
			if err != nil || value < 1 {
				return Phase{}, fmt.Errorf("line %d: invalid denominator", lineNumber)
			}
			phase.Denominator = value
		case "case":
			if (phase.Version == 1 && len(fields) != 6) || (phase.Version >= 2 && len(fields) != 7) {
				return Phase{}, fmt.Errorf("line %d: case expects id source expected root capability [cohort]", lineNumber)
			}
			for _, item := range phase.Cases {
				if item.ID == fields[1] {
					return Phase{}, fmt.Errorf("line %d: duplicate case %q", lineNumber, fields[1])
				}
			}
			item := Case{ID: fields[1], Source: fields[2], Expected: fields[3], Root: fields[4], Capability: fields[5]}
			if phase.Version >= 2 {
				item.Cohort = fields[6]
			}
			phase.Cases = append(phase.Cases, item)
		case "grant":
			if len(fields) < 4 {
				return Phase{}, fmt.Errorf("line %d: grant expects case capability effect...", lineNumber)
			}
			grant := Grant{CaseID: fields[1]}
			start := 3
			if phase.Version >= 2 {
				if len(fields) < 5 {
					return Phase{}, fmt.Errorf("line %d: v2 grant expects case stage capability effect...", lineNumber)
				}
				grant.Stage = fields[2]
				grant.Capability = fields[3]
				start = 4
			} else {
				grant.Capability = fields[2]
			}
			effects, err := parseEffectTokens(fields, start, lineNumber)
			if err != nil {
				return Phase{}, err
			}
			grant.Effects = effects
			for _, item := range phase.Grants {
				if item.CaseID == grant.CaseID && item.Stage == grant.Stage && item.Capability == grant.Capability {
					return Phase{}, fmt.Errorf("line %d: duplicate grant", lineNumber)
				}
			}
			phase.Grants = append(phase.Grants, grant)
		case "effect_summary":
			if len(fields) < 5 {
				return Phase{}, fmt.Errorf("line %d: effect_summary expects case stage capability effect...", lineNumber)
			}
			effects, err := parseEffectTokens(fields, 4, lineNumber)
			if err != nil {
				return Phase{}, err
			}
			item := EffectSummary{CaseID: fields[1], Stage: fields[2], Capability: fields[3], Effects: effects}
			if hasSummary(phase.EffectSummaries, item) {
				return Phase{}, fmt.Errorf("line %d: duplicate effect summary", lineNumber)
			}
			phase.EffectSummaries = append(phase.EffectSummaries, item)
		case "attenuation":
			if len(fields) < 6 {
				return Phase{}, fmt.Errorf("line %d: attenuation expects case from to capability effect...", lineNumber)
			}
			effects, err := parseEffectTokens(fields, 5, lineNumber)
			if err != nil {
				return Phase{}, err
			}
			item := Attenuation{CaseID: fields[1], From: fields[2], To: fields[3], Capability: fields[4], Effects: effects}
			if hasAttenuation(phase.Attenuations, item) {
				return Phase{}, fmt.Errorf("line %d: duplicate attenuation edge", lineNumber)
			}
			phase.Attenuations = append(phase.Attenuations, item)
		case "activity":
			if len(fields) != 6 {
				return Phase{}, fmt.Errorf("line %d: activity expects case stage capability function name", lineNumber)
			}
			item := Activity{CaseID: fields[1], Stage: fields[2], Capability: fields[3], Function: fields[4], Name: fields[5]}
			if hasActivity(phase.Activities, item) {
				return Phase{}, fmt.Errorf("line %d: duplicate activity", lineNumber)
			}
			phase.Activities = append(phase.Activities, item)
		case "cell":
			if len(fields) != 7 {
				return Phase{}, fmt.Errorf("line %d: cell expects id case cohort proof indicator activity", lineNumber)
			}
			for _, item := range phase.Cells {
				if item.ID == fields[1] {
					return Phase{}, fmt.Errorf("line %d: duplicate cell %q", lineNumber, fields[1])
				}
			}
			phase.Cells = append(phase.Cells, Cell{ID: fields[1], CaseID: fields[2], Cohort: fields[3], Proof: fields[4], Indicator: fields[5], Activity: fields[6]})
		case "proof":
			if len(fields) != 3 {
				return Phase{}, fmt.Errorf("line %d: proof expects category and count", lineNumber)
			}
			value, err := strconv.Atoi(fields[2])
			if err != nil || value < 1 {
				return Phase{}, fmt.Errorf("line %d: invalid proof count", lineNumber)
			}
			if _, exists := phase.ProofQuotas[fields[1]]; exists {
				return Phase{}, fmt.Errorf("line %d: duplicate proof category", lineNumber)
			}
			phase.ProofQuotas[fields[1]] = value
		case "indicator":
			if len(fields) != 3 {
				return Phase{}, fmt.Errorf("line %d: indicator expects category and count", lineNumber)
			}
			value, err := strconv.Atoi(fields[2])
			if err != nil || value < 1 {
				return Phase{}, fmt.Errorf("line %d: invalid indicator count", lineNumber)
			}
			if _, exists := phase.IndicatorQuotas[fields[1]]; exists {
				return Phase{}, fmt.Errorf("line %d: duplicate indicator category", lineNumber)
			}
			phase.IndicatorQuotas[fields[1]] = value
		case "fixed_point":
			if len(fields) != 2 {
				return Phase{}, fmt.Errorf("line %d: fixed_point expects explicit", lineNumber)
			}
			phase.FixedPoint = fields[1]
		case "authority":
			if len(fields) != 3 {
				return Phase{}, fmt.Errorf("line %d: authority expects field and integer", lineNumber)
			}
			value, err := strconv.Atoi(fields[2])
			if err != nil || value < 0 {
				return Phase{}, fmt.Errorf("line %d: invalid authority value", lineNumber)
			}
			switch fields[1] {
			case "repository_writes":
				phase.AuthorityPolicy.RepositoryWrites = value
			case "local_test_executions":
				phase.AuthorityPolicy.LocalTestExecutions = value
			case "cross_project_required_gates":
				phase.AuthorityPolicy.CrossProjectRequiredGates = value
			default:
				return Phase{}, fmt.Errorf("line %d: unknown authority field", lineNumber)
			}
		case "plan":
			if len(fields) != 4 || fields[1] != "parser" || fields[2] != "typechecker" || fields[3] != "evaluator" {
				if phase.Version == 1 && len(fields) == 4 && fields[1] == "parser" && fields[2] == "executor" && fields[3] == "emitter" {
					phase.Plan = append([]string(nil), fields[1:]...)
					continue
				}
				return Phase{}, fmt.Errorf("line %d: plan must declare parser typechecker evaluator", lineNumber)
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
		return Phase{}, fmt.Errorf("missing gooo header")
	}
	if err := validatePhase(phase, seen); err != nil {
		return Phase{}, err
	}
	phase.Digest = digestBytes(data)
	return phase, nil
}

func parseEffectTokens(fields []string, start, lineNumber int) ([]string, error) {
	if start >= len(fields) {
		return []string{}, nil
	}
	if len(fields[start:]) == 1 && fields[start] == "NONE" {
		return []string{}, nil
	}
	for _, value := range fields[start:] {
		if value == "NONE" || value == "" {
			return nil, fmt.Errorf("line %d: NONE must be the only effect token", lineNumber)
		}
	}
	effects := append([]string(nil), fields[start:]...)
	sort.Strings(effects)
	return effects, nil
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
	fixture := Fixture{Schema: "gooo/capability-effect-checker/fixture/v2", Version: 2, Functions: make(map[string]Function)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	headerSeen := false
	scenarioSeen := false
	rootSeen := false
	directSeen := make(map[string]bool)
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if !headerSeen {
			if len(fields) != 2 || fields[0] != "gooo" {
				return Fixture{}, fmt.Errorf("line %d: expected gooo 1 or gooo 2", lineNumber)
			}
			version, err := strconv.Atoi(fields[1])
			if err != nil || (version != 1 && version != 2) {
				return Fixture{}, fmt.Errorf("line %d: unsupported fixture version", lineNumber)
			}
			fixture.Version = version
			fixture.Schema = fmt.Sprintf("gooo/capability-effect-checker/fixture/v%d", version)
			headerSeen = true
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
			if len(fields) < 2 {
				return Fixture{}, fmt.Errorf("line %d: direct expects function [effect...]", lineNumber)
			}
			if directSeen[fields[1]] {
				return Fixture{}, fmt.Errorf("line %d: duplicate direct effects for %q", lineNumber, fields[1])
			}
			effects, err := parseEffectTokens(fields, 2, lineNumber)
			if err != nil {
				return Fixture{}, err
			}
			function := ensureFunction(fixture.Functions, fields[1])
			function.DirectEffects = effects
			fixture.Functions[fields[1]] = function
			directSeen[fields[1]] = true
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
		case "ambiguous_call":
			if len(fields) < 3 {
				return Fixture{}, fmt.Errorf("line %d: ambiguous_call expects caller and candidates", lineNumber)
			}
			candidates := append([]string(nil), fields[2:]...)
			sort.Strings(candidates)
			fixture.AmbiguousCalls = append(fixture.AmbiguousCalls, AmbiguousCall{Caller: fields[1], Candidates: candidates})
		case "oracle":
			if len(fields) != 4 {
				return Fixture{}, fmt.Errorf("line %d: oracle expects function name status", lineNumber)
			}
			function := ensureFunction(fixture.Functions, fields[1])
			function.Oracles = append(function.Oracles, Oracle{Name: fields[2], Status: fields[3]})
			sort.Slice(function.Oracles, func(i, j int) bool { return function.Oracles[i].Name < function.Oracles[j].Name })
			fixture.Functions[fields[1]] = function
		case "output":
			if len(fields) != 2 || fixture.OutputScope != "" {
				return Fixture{}, fmt.Errorf("line %d: output expects one value and may appear once", lineNumber)
			}
			fixture.OutputScope = fields[1]
		case "network":
			if len(fields) != 3 {
				return Fixture{}, fmt.Errorf("line %d: network expects function pin", lineNumber)
			}
			fixture.NetworkPins = append(fixture.NetworkPins, NetworkPin{Function: fields[1], Pin: fields[2]})
		case "grant", "generated_grant", "self_grant":
			if len(fields) < 4 {
				return Fixture{}, fmt.Errorf("line %d: generated_grant expects function capability effect...", lineNumber)
			}
			effects, err := parseEffectTokens(fields, 3, lineNumber)
			if err != nil {
				return Fixture{}, err
			}
			fixture.GrantClaims = append(fixture.GrantClaims, GrantClaim{Function: fields[1], Capability: fields[2], Effects: effects})
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
	if phase.Version == 1 {
		return validateV1Phase(phase, seen)
	}
	if !sameSlice(phase.Effects, effectStrings(CanonicalEffects)) {
		return fmt.Errorf("v2 effect vocabulary must be the six canonical typed effects")
	}
	if !sameSlice(phase.Stages, []string{StageMetacode, StageGeneratedCode, StageRuntime}) {
		return fmt.Errorf("v2 stages must be METACODE, GENERATED_CODE, RUNTIME")
	}
	if len(phase.StageEdges) != 2 || phase.StageEdges[0] != (StageEdge{From: StageMetacode, To: StageGeneratedCode}) || phase.StageEdges[1] != (StageEdge{From: StageGeneratedCode, To: StageRuntime}) {
		return fmt.Errorf("v2 stage edges must be METACODE->GENERATED_CODE->RUNTIME")
	}
	if phase.OutputScope != "caller-owned" || !seen["output_scope"] {
		return fmt.Errorf("v2 output scope must be caller-owned")
	}
	if phase.FixedPoint != "explicit" || !seen["fixed_point"] {
		return fmt.Errorf("v2 requires fixed_point explicit")
	}
	if phase.Denominator != 12 || len(phase.Cases) != 12 {
		return fmt.Errorf("v2 denominator must contain exactly 12 cases")
	}
	if len(phase.Cells) != 12 {
		return fmt.Errorf("v2 must contain exactly 12 cells")
	}
	if !sameIntMap(phase.ProofQuotas, map[string]int{"FOUNDATION": 4, "COHERENCE": 4, "REGRESSION": 4}) {
		return fmt.Errorf("proof quota must be FOUNDATION/COHERENCE/REGRESSION 4/4/4")
	}
	if !sameIntMap(phase.IndicatorQuotas, map[string]int{"DRIVER": 4, "OUTCOME": 4, "GUARDRAIL": 4}) {
		return fmt.Errorf("indicator quota must be DRIVER/OUTCOME/GUARDRAIL 4/4/4")
	}
	if phase.AuthorityPolicy != (Authority{}) {
		return fmt.Errorf(".gooo authority policy must keep all runtime mutation counts at zero")
	}
	for _, effect := range phase.RiskEffects {
		if !contains(phase.Effects, effect) || !contains([]string{"REPOSITORY_WRITE", "REMOTE_MUTATION", "DESTRUCTIVE_DELETE"}, effect) {
			return fmt.Errorf("risk effect %q is not a known forbidden effect", effect)
		}
	}
	if !sameSet(phase.RiskEffects, []string{"REPOSITORY_WRITE", "REMOTE_MUTATION", "DESTRUCTIVE_DELETE"}) {
		return fmt.Errorf("v2 requires the three known forbidden effects")
	}
	caseIDs := make(map[string]bool, len(phase.Cases))
	for _, item := range phase.Cases {
		if caseIDs[item.ID] {
			return fmt.Errorf("duplicate case %q", item.ID)
		}
		caseIDs[item.ID] = true
		if !contains(phase.Precedence, item.Expected) || !contains(phase.Capabilities, item.Capability) {
			return fmt.Errorf("case %q uses unsupported expected status or capability", item.ID)
		}
	}
	cellCases := make(map[string]bool, len(phase.Cells))
	proofCounts := make(map[string]int)
	indicatorCounts := make(map[string]int)
	cohortCounts := make(map[string]int)
	for _, cell := range phase.Cells {
		if !caseIDs[cell.CaseID] || cellCases[cell.CaseID] || cell.Cohort == "" || cell.Proof == "" || cell.Indicator == "" || cell.Activity == "" {
			return fmt.Errorf("invalid or duplicate cell %q", cell.ID)
		}
		cellCases[cell.CaseID] = true
		for _, item := range phase.Cases {
			if item.ID == cell.CaseID && item.Cohort != cell.Cohort {
				return fmt.Errorf("cell %q cohort does not match case", cell.ID)
			}
		}
		proofCounts[cell.Proof]++
		indicatorCounts[cell.Indicator]++
		cohortCounts[cell.Cohort]++
	}
	if len(cellCases) != 12 || !sameIntMap(proofCounts, phase.ProofQuotas) || !sameIntMap(indicatorCounts, phase.IndicatorQuotas) || !sameIntMap(cohortCounts, map[string]int{"FOUNDATION": 4, "COHERENCE": 4, "REGRESSION": 4}) {
		return fmt.Errorf("v2 cell proof, indicator, and case cohorts must be exact 4/4/4")
	}
	for _, grant := range phase.Grants {
		if !caseIDs[grant.CaseID] || !contains(phase.Stages, grant.Stage) || !contains(phase.Capabilities, grant.Capability) {
			return fmt.Errorf("grant references unknown case, stage, or capability")
		}
		if !allKnownEffects(phase, grant.Effects) {
			return fmt.Errorf("grant references unknown effect")
		}
	}
	for _, summary := range phase.EffectSummaries {
		if !caseIDs[summary.CaseID] || !contains(phase.Stages, summary.Stage) || !contains(phase.Capabilities, summary.Capability) || !allKnownEffects(phase, summary.Effects) {
			return fmt.Errorf("effect summary references unknown authority")
		}
	}
	for _, edge := range phase.Attenuations {
		if !caseIDs[edge.CaseID] || !contains(phase.Stages, edge.From) || !contains(phase.Stages, edge.To) || !contains(phase.Capabilities, edge.Capability) || !allKnownEffects(phase, edge.Effects) || !hasStageEdge(phase, edge.From, edge.To) {
			return fmt.Errorf("attenuation references unknown or undeclared stage edge")
		}
	}
	for _, item := range phase.Cases {
		for _, stageEdge := range phase.StageEdges {
			parent, parentOK := findGrant(phase, item.ID, stageEdge.From, item.Capability)
			child, childOK := findGrant(phase, item.ID, stageEdge.To, item.Capability)
			if parentOK && childOK && len(NewEffectSet(child).Difference(NewEffectSet(parent))) > 0 {
				return fmt.Errorf("grant for %s widens from %s to %s", item.ID, stageEdge.From, stageEdge.To)
			}
			attenuation, attenuationOK := findAttenuation(phase, item.ID, stageEdge.From, stageEdge.To, item.Capability)
			if !attenuationOK {
				continue
			}
			if parentOK && len(NewEffectSet(attenuation.Effects).Difference(NewEffectSet(parent))) > 0 {
				return fmt.Errorf("attenuation for %s widens beyond parent grant", item.ID)
			}
			if childOK && len(NewEffectSet(attenuation.Effects).Difference(NewEffectSet(child))) > 0 {
				return fmt.Errorf("attenuation for %s widens beyond child grant", item.ID)
			}
		}
	}
	for _, activity := range phase.Activities {
		if !caseIDs[activity.CaseID] || !contains(phase.Stages, activity.Stage) || !contains(phase.Capabilities, activity.Capability) || activity.Function == "" || activity.Name == "" {
			return fmt.Errorf("activity references unknown authority")
		}
	}
	return validateDiagnostics(phase)
}

func validateV1Phase(phase Phase, seen map[string]bool) error {
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
		if !contains(phase.Precedence, item.Expected) || !contains(phase.Capabilities, item.Capability) {
			return fmt.Errorf("case %q has unsupported expected decision or capability", item.ID)
		}
	}
	for _, grant := range phase.Grants {
		if !containsCase(phase.Cases, grant.CaseID) || !contains(phase.Capabilities, grant.Capability) {
			return fmt.Errorf("grant references unknown case or capability")
		}
		for _, effect := range grant.Effects {
			if !contains(phase.Effects, effect) {
				return fmt.Errorf("grant references unknown effect %q", effect)
			}
		}
	}
	return validateDiagnostics(phase)
}

func validateDiagnostics(phase Phase) error {
	for name, status := range phase.Classifications {
		if !contains(phase.Precedence, status) {
			return fmt.Errorf("classification %q has unsupported status %q", name, status)
		}
		if _, ok := phase.Diagnostics[name]; !ok {
			return fmt.Errorf("classification %q has no diagnostic", name)
		}
	}
	return nil
}

func effectStrings(values []Effect) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = string(value)
	}
	return out
}

func allKnownEffects(phase Phase, effects []string) bool {
	for _, effect := range effects {
		if !contains(phase.Effects, effect) {
			return false
		}
	}
	return true
}

func hasStageEdge(phase Phase, from, to string) bool {
	for _, edge := range phase.StageEdges {
		if edge.From == from && edge.To == to {
			return true
		}
	}
	return false
}

func hasSummary(values []EffectSummary, wanted EffectSummary) bool {
	for _, value := range values {
		if value.CaseID == wanted.CaseID && value.Stage == wanted.Stage && value.Capability == wanted.Capability {
			return true
		}
	}
	return false
}

func hasAttenuation(values []Attenuation, wanted Attenuation) bool {
	for _, value := range values {
		if value.CaseID == wanted.CaseID && value.From == wanted.From && value.To == wanted.To && value.Capability == wanted.Capability {
			return true
		}
	}
	return false
}

func hasActivity(values []Activity, wanted Activity) bool {
	for _, value := range values {
		if value.CaseID == wanted.CaseID && value.Function == wanted.Function {
			return true
		}
	}
	return false
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

func sameIntMap(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
