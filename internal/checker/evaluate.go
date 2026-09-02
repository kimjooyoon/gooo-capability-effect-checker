package checker

import (
	"sort"
	"strings"
)

func EvaluateCase(phase Phase, input Case, fixture Fixture) CaseResult {
	unknowns := make([]Unknown, 0)
	refutations := make([]Refutation, 0)
	paths := make([]CallPath, 0)
	externalDependencies := make([]string, 0)
	stageResults := make([]StageResult, 0)
	attenuationEvidence := make([]AttenuationEvidence, 0)
	unknownGeneratedEffects := make([]string, 0)

	if fixture.Scenario != input.ID {
		refutations = append(refutations, refutationFrom(phase, "input_contract_mismatch", fixture.Scenario+" != "+input.ID))
	}
	if fixture.Root != input.Root {
		refutations = append(refutations, refutationFrom(phase, "input_contract_mismatch", "root:"+fixture.Root+" expected:"+input.Root))
	}

	reachable, graphUnknowns, graphRefutations, graphPaths := walkGraph(phase, fixture)
	unknowns = append(unknowns, graphUnknowns...)
	refutations = append(refutations, graphRefutations...)
	paths = append(paths, graphPaths...)

	memo := make(map[string][]string)
	inferred := inferEffects(fixture.Root, fixture, memo, make(map[string]bool))
	sort.Strings(inferred)

	if phase.Version >= 2 {
		if fixture.OutputScope == "" || fixture.OutputScope == "unknown" {
			unknowns = append(unknowns, unknownFrom(phase, "missing_path_scope", fixture.Root+":OUTPUT_SCOPE"))
		} else if fixture.OutputScope != phase.OutputScope {
			refutations = append(refutations, refutationFrom(phase, "output_scope_violation", fixture.OutputScope))
		}
	}

	functionNames := make([]string, 0, len(reachable))
	for name := range reachable {
		functionNames = append(functionNames, name)
	}
	sort.Strings(functionNames)
	for _, name := range functionNames {
		function := fixture.Functions[name]
		functionEffects := inferEffects(name, fixture, memo, make(map[string]bool))
		knownDirect := filterKnownEffects(phase, function.DirectEffects)
		unknownDirect := difference(function.DirectEffects, phase.Effects)
		for _, effect := range unknownDirect {
			if phase.Version == 1 {
				refutations = append(refutations, refutationFrom(phase, "direct_undeclared_effect", name+":"+effect))
				paths = append(paths, CallPath{Issue: "direct_undeclared_effect", Effect: effect, Function: name, Path: clonePath(reachable[name])})
				continue
			}
			unknownGeneratedEffects = append(unknownGeneratedEffects, effect)
			unknowns = append(unknowns, unknownFrom(phase, "unknown_generated_effect", name+":"+effect))
			paths = append(paths, CallPath{Issue: "unknown_generated_effect", Effect: effect, Function: name, Path: clonePath(reachable[name])})
		}
		if phase.Version >= 2 && contains(knownDirect, string(EffectNetworkReadPinned)) {
			pin, pinned := findNetworkPin(fixture, name)
			if !pinned || pin != "pinned" {
				unknowns = append(unknowns, unknownFrom(phase, "missing_path_scope", name+":NETWORK_PIN"))
				paths = append(paths, CallPath{Issue: "missing_path_scope", Effect: string(EffectNetworkReadPinned), Function: name, Path: clonePath(reachable[name])})
			}
		}

		activity, hasActivity := findActivity(phase, input.ID, name)
		if phase.Version >= 2 && !hasActivity {
			unknowns = append(unknowns, unknownFrom(phase, "missing_path_scope", name+":ACTIVITY"))
		}

		grant := []string(nil)
		grantExists := false
		declaredSummary := []string(nil)
		if hasActivity {
			grant, grantExists = findGrant(phase, input.ID, activity.Stage, activity.Capability)
			declaredSummary, _ = findEffectSummary(phase, input.ID, activity.Stage, activity.Capability)
		} else {
			grant, grantExists = findGrant(phase, input.ID, "", name)
		}

		if phase.Version >= 2 && hasActivity && declaredSummary == nil {
			unknowns = append(unknowns, unknownFrom(phase, "missing_effect_summary", name+":"+activity.Stage))
		}
		if phase.Version >= 2 && hasActivity && declaredSummary != nil && !sameSet(knownDirect, declaredSummary) {
			refutations = append(refutations, refutationFrom(phase, "effect_summary_mismatch", name+":"+strings.Join(knownDirect, ",")+" expected:"+strings.Join(declaredSummary, ",")))
		}

		if !grantExists {
			if phase.Version == 1 {
				if len(functionEffects) == 0 {
					unknowns = append(unknowns, unknownFrom(phase, "missing_indirect_grant", name+":CALLEE_GRANT"))
				}
				for _, effect := range functionEffects {
					if isKnownForbidden(phase, effect) {
						refutations = append(refutations, refutationFrom(phase, "forbidden_mutation", name+":"+effect))
						paths = append(paths, CallPath{Issue: "forbidden_mutation", Effect: effect, Function: name, Path: clonePath(reachable[name])})
					} else {
						unknowns = append(unknowns, unknownFrom(phase, "missing_indirect_grant", name+":"+effect))
						paths = append(paths, CallPath{Issue: "missing_indirect_grant", Effect: effect, Function: name, Path: clonePath(reachable[name])})
					}
				}
				continue
			}
			if hasKnownForbidden(phase, functionEffects) {
				for _, effect := range functionEffects {
					if isKnownForbidden(phase, effect) {
						refutations = append(refutations, refutationFrom(phase, "forbidden_effect", name+":"+effect))
						paths = append(paths, CallPath{Issue: "forbidden_effect", Effect: effect, Function: name, Path: clonePath(reachable[name])})
					}
				}
			}
			blocked := name + ":GRANT"
			if hasActivity {
				blocked = name + ":" + activity.Stage + ":" + activity.Capability
			}
			unknowns = append(unknowns, unknownFrom(phase, "missing_grant", blocked))
		} else {
			grantSet := NewEffectSet(grant)
			for _, effect := range functionEffects {
				if !contains(phase.Effects, effect) || grantSet.Contains(effect) {
					continue
				}
				issue := "effect_amplification"
				classification := "grant_amplification"
				if phase.Version == 1 {
					issue = "declared_grant_omission"
					classification = "declared_grant_omission"
				}
				if isKnownForbidden(phase, effect) {
					issue = "forbidden_effect"
					classification = "forbidden_effect"
					if phase.Version == 1 {
						issue = "forbidden_mutation"
						classification = "forbidden_mutation"
					}
				}
				refutations = append(refutations, refutationFrom(phase, classification, name+":"+effect))
				paths = append(paths, CallPath{Issue: issue, Effect: effect, Function: name, Path: clonePath(reachable[name])})
			}
		}

		if phase.Version == 1 {
			for _, effect := range function.DirectEffects {
				if !contains(phase.Effects, effect) {
					refutations = append(refutations, refutationFrom(phase, "direct_undeclared_effect", name+":"+effect))
					paths = append(paths, CallPath{Issue: "direct_undeclared_effect", Effect: effect, Function: name, Path: clonePath(reachable[name])})
				}
			}
		}

		for _, oracle := range function.Oracles {
			if oracle.Status == "available" {
				continue
			}
			dependency := name + ":" + oracle.Name
			unknowns = append(unknowns, unknownFrom(phase, "external_oracle_dependency", dependency))
			paths = append(paths, CallPath{Issue: "external_oracle_dependency", Function: name, Path: clonePath(reachable[name])})
			externalDependencies = append(externalDependencies, dependency)
		}

		if hasActivity {
			stageResults = append(stageResults, StageResult{
				Stage: activity.Stage, Capability: activity.Capability, Function: name,
				DirectEffects: knownDirect, InferredEffects: filterKnownEffects(phase, functionEffects),
				DeclaredGrant: cloneStrings(grant), DeclaredSummary: cloneStrings(declaredSummary),
				MissingEffects: NewEffectSet(functionEffects).Difference(NewEffectSet(grant)),
			})
		} else if phase.Version == 1 {
			stageResults = append(stageResults, StageResult{Function: name, DirectEffects: knownDirect, InferredEffects: filterKnownEffects(phase, functionEffects), DeclaredGrant: cloneStrings(grant)})
		}
	}

	if phase.Version >= 2 {
		for _, claim := range fixture.GrantClaims {
			activity, ok := findActivity(phase, input.ID, claim.Function)
			if ok && (activity.Stage == StageGeneratedCode || activity.Stage == StageRuntime) {
				refutations = append(refutations, refutationFrom(phase, "forged_grant", claim.Function+":"+claim.Capability+":"+strings.Join(claim.Effects, ",")))
				paths = append(paths, CallPath{Issue: "forged_grant", Function: claim.Function, Path: clonePath(reachable[claim.Function])})
			}
		}

		for _, ambiguous := range fixture.AmbiguousCalls {
			if _, ok := reachable[ambiguous.Caller]; !ok {
				continue
			}
			path := append(clonePath(reachable[ambiguous.Caller]), "?")
			unknowns = append(unknowns, unknownFrom(phase, "missing_call_edge", ambiguous.Caller+"?"))
			paths = append(paths, CallPath{Issue: "missing_call_edge", Function: ambiguous.Caller, Path: path})
		}

		for _, caller := range functionNames {
			callerActivity, callerOK := findActivity(phase, input.ID, caller)
			if !callerOK {
				continue
			}
			calls := append([]string(nil), fixture.Functions[caller].Calls...)
			sort.Strings(calls)
			for _, child := range calls {
				childActivity, childOK := findActivity(phase, input.ID, child)
				if !childOK {
					continue
				}
				path := append(clonePath(reachable[caller]), child)
				callerRank := stageRank(phase, callerActivity.Stage)
				childRank := stageRank(phase, childActivity.Stage)
				if childRank < callerRank {
					issue := "path_scope_expansion"
					classification := "path_scope_expansion"
					if contains(inferEffects(child, fixture, memo, make(map[string]bool)), string(EffectDestructiveDelete)) && stageRank(phase, childActivity.Stage) == 0 {
						issue = "destructive_ancestor_delete"
						classification = "destructive_ancestor_delete"
					}
					refutations = append(refutations, refutationFrom(phase, classification, strings.Join(path, "->")))
					paths = append(paths, CallPath{Issue: issue, Function: child, Path: path})
					continue
				}
				if childActivity.Stage == callerActivity.Stage && childActivity.Capability != callerActivity.Capability {
					refutations = append(refutations, refutationFrom(phase, "sibling_path_expansion", strings.Join(path, "->")))
					paths = append(paths, CallPath{Issue: "sibling_path_expansion", Function: child, Path: path})
					continue
				}
				if childRank <= callerRank {
					continue
				}
				attenuation, ok := findAttenuation(phase, input.ID, callerActivity.Stage, childActivity.Stage, callerActivity.Capability)
				if !ok {
					unknowns = append(unknowns, unknownFrom(phase, "missing_call_edge", strings.Join(path, "->")))
					paths = append(paths, CallPath{Issue: "missing_call_edge", Function: child, Path: path})
					continue
				}
				observed := filterKnownEffects(phase, inferEffects(child, fixture, memo, make(map[string]bool)))
				attenuationEvidence = append(attenuationEvidence, AttenuationEvidence{
					FromStage: callerActivity.Stage, ToStage: childActivity.Stage, Capability: callerActivity.Capability,
					AllowedEffects: cloneStrings(attenuation.Effects), ObservedEffects: observed, Path: path,
				})
				allowed := NewEffectSet(attenuation.Effects)
				for _, effect := range observed {
					if allowed.Contains(effect) {
						continue
					}
					issue := "effect_amplification"
					classification := "grant_amplification"
					if isKnownForbidden(phase, effect) {
						issue = "forbidden_effect"
						classification = "forbidden_effect"
					}
					refutations = append(refutations, refutationFrom(phase, classification, strings.Join(path, "->")+":"+effect))
					paths = append(paths, CallPath{Issue: issue, Effect: effect, Function: child, Path: path})
				}
			}
		}
	}

	rootGrant, rootGrantExists := findRootGrant(phase, input.ID, fixture.Root)
	if !rootGrantExists && phase.Version == 1 {
		unknowns = append(unknowns, unknownFrom(phase, "missing_indirect_grant", fixture.Root+":ROOT_GRANT"))
	}
	missingRoot := difference(inferred, rootGrant)
	sort.Strings(externalDependencies)
	sort.Strings(unknownGeneratedEffects)
	unknowns = uniqueUnknowns(unknowns)
	refutations = uniqueRefutations(refutations)
	return CaseResult{
		ID: input.ID, Expected: input.Expected, Cohort: input.Cohort, Decision: reduce(phase, unknowns, refutations),
		InferredEffects: inferred, DeclaredRootEffects: rootGrant, MissingRootEffects: missingRoot,
		UnknownGeneratedEffects: uniqueStrings(unknownGeneratedEffects), Unknowns: unknowns, Refutations: refutations,
		OffendingCallPaths: uniqueSortedPaths(paths), StageResults: stageResults, AttenuationEvidence: attenuationEvidence,
		ExternalDependencies: externalDependencies, OutputScope: fixture.OutputScope,
	}
}

func walkGraph(phase Phase, fixture Fixture) (map[string][]string, []Unknown, []Refutation, []CallPath) {
	reachable := map[string][]string{fixture.Root: []string{fixture.Root}}
	queue := []string{fixture.Root}
	unknowns := make([]Unknown, 0)
	refutations := make([]Refutation, 0)
	paths := make([]CallPath, 0)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		function := fixture.Functions[name]
		calls := append([]string(nil), function.Calls...)
		sort.Strings(calls)
		for _, child := range calls {
			candidate := append(clonePath(reachable[name]), child)
			if contains(reachable[name], child) {
				refutations = append(refutations, refutationFrom(phase, "call_graph_cycle", strings.Join(candidate, "->")))
				paths = append(paths, CallPath{Issue: "call_graph_cycle", Function: child, Path: candidate})
				continue
			}
			if _, exists := fixture.Functions[child]; !exists {
				unknowns = append(unknowns, unknownFrom(phase, "unknown_call_target", child))
				paths = append(paths, CallPath{Issue: "unknown_call_target", Function: child, Path: candidate})
				continue
			}
			if _, visited := reachable[child]; visited {
				continue
			}
			reachable[child] = candidate
			queue = append(queue, child)
		}
	}
	return reachable, unknowns, refutations, paths
}

func inferEffects(name string, fixture Fixture, memo map[string][]string, visiting map[string]bool) []string {
	if effects, ok := memo[name]; ok {
		return append([]string(nil), effects...)
	}
	if visiting[name] {
		return append([]string(nil), fixture.Functions[name].DirectEffects...)
	}
	function, exists := fixture.Functions[name]
	if !exists {
		return []string{}
	}
	visiting[name] = true
	set := NewEffectSet(function.DirectEffects)
	for _, child := range function.Calls {
		set = set.Union(NewEffectSet(inferEffects(child, fixture, memo, visiting)))
	}
	delete(visiting, name)
	effects := set.Sorted()
	memo[name] = append([]string(nil), effects...)
	return effects
}

func findActivity(phase Phase, caseID, function string) (Activity, bool) {
	for _, activity := range phase.Activities {
		if activity.CaseID == caseID && activity.Function == function {
			return activity, true
		}
	}
	return Activity{}, false
}

func findGrant(phase Phase, caseID string, parts ...string) ([]string, bool) {
	stage := ""
	capability := ""
	if len(parts) == 1 {
		capability = parts[0]
	} else if len(parts) >= 2 {
		stage, capability = parts[0], parts[1]
	}
	for _, grant := range phase.Grants {
		if grant.CaseID == caseID && grant.Stage == stage && grant.Capability == capability {
			effects := append([]string(nil), grant.Effects...)
			sort.Strings(effects)
			return effects, true
		}
	}
	return nil, false
}

func findRootGrant(phase Phase, caseID, root string) ([]string, bool) {
	if activity, ok := findActivity(phase, caseID, root); ok {
		return findGrant(phase, caseID, activity.Stage, activity.Capability)
	}
	return findGrant(phase, caseID, "", root)
}

func findEffectSummary(phase Phase, caseID, stage, capability string) ([]string, bool) {
	for _, summary := range phase.EffectSummaries {
		if summary.CaseID == caseID && summary.Stage == stage && summary.Capability == capability {
			effects := append([]string{}, summary.Effects...)
			sort.Strings(effects)
			return effects, true
		}
	}
	return nil, false
}

func findAttenuation(phase Phase, caseID, from, to, capability string) (Attenuation, bool) {
	for _, edge := range phase.Attenuations {
		if edge.CaseID == caseID && edge.From == from && edge.To == to && edge.Capability == capability {
			return edge, true
		}
	}
	return Attenuation{}, false
}

func findNetworkPin(fixture Fixture, function string) (string, bool) {
	for _, network := range fixture.NetworkPins {
		if network.Function == function {
			return network.Pin, true
		}
	}
	return "", false
}

func filterKnownEffects(phase Phase, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if contains(phase.Effects, value) {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return uniqueStrings(out)
}

func isKnownForbidden(phase Phase, effect string) bool {
	return contains(phase.RiskEffects, effect)
}

func hasKnownForbidden(phase Phase, effects []string) bool {
	for _, effect := range effects {
		if isKnownForbidden(phase, effect) {
			return true
		}
	}
	return false
}

func stageRank(phase Phase, stage string) int {
	for index, declared := range phase.Stages {
		if declared == stage {
			return index
		}
	}
	return -1
}

func reduce(phase Phase, unknowns []Unknown, refutations []Refutation) string {
	for _, status := range phase.Precedence {
		switch status {
		case DecisionRefuted:
			if len(refutations) > 0 {
				return status
			}
		case DecisionUnknown:
			if len(unknowns) > 0 {
				return status
			}
		case DecisionClosed:
			return status
		}
	}
	return DecisionUnknown
}

func unknownFrom(phase Phase, key, blocked string) Unknown {
	diagnostic := phase.Diagnostics[key]
	if blocked == "" {
		blocked = diagnostic.BlockedBy
	}
	return Unknown{
		Stage: diagnostic.Stage, Step: diagnostic.Step, Reason: diagnostic.Reason,
		UnknownClass: diagnostic.UnknownClass, NextOperation: diagnostic.NextOperation,
		BlockedBy: []string{blocked},
	}
}

func refutationFrom(phase Phase, key, counterexample string) Refutation {
	diagnostic := phase.Diagnostics[key]
	return Refutation{Stage: diagnostic.Stage, Step: diagnostic.Step, Reason: diagnostic.Reason, Counterexample: counterexample}
}

func difference(left, right []string) []string {
	out := make([]string, 0)
	for _, value := range left {
		if !contains(right, value) {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueUnknowns(values []Unknown) []Unknown {
	seen := make(map[string]bool, len(values))
	out := make([]Unknown, 0, len(values))
	for _, value := range values {
		key := value.Stage + "|" + value.Step + "|" + value.Reason + "|" + value.UnknownClass + "|" + value.NextOperation + "|" + strings.Join(value.BlockedBy, ",")
		if seen[key] {
			continue
		}
		seen[key] = true
		value.BlockedBy = append([]string(nil), value.BlockedBy...)
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Stage + "|" + out[i].Step + "|" + out[i].UnknownClass + "|" + strings.Join(out[i].BlockedBy, ",")
		right := out[j].Stage + "|" + out[j].Step + "|" + out[j].UnknownClass + "|" + strings.Join(out[j].BlockedBy, ",")
		return left < right
	})
	return out
}

func uniqueRefutations(values []Refutation) []Refutation {
	seen := make(map[string]bool, len(values))
	out := make([]Refutation, 0, len(values))
	for _, value := range values {
		key := value.Stage + "|" + value.Step + "|" + value.Reason + "|" + value.Counterexample
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Stage + "|" + out[i].Step + "|" + out[i].Counterexample
		right := out[j].Stage + "|" + out[j].Step + "|" + out[j].Counterexample
		return left < right
	})
	return out
}

func uniqueSortedPaths(paths []CallPath) []CallPath {
	seen := make(map[string]bool)
	out := make([]CallPath, 0, len(paths))
	for _, path := range paths {
		key := path.Issue + "|" + path.Effect + "|" + path.Function + "|" + strings.Join(path.Path, "->")
		if seen[key] {
			continue
		}
		seen[key] = true
		path.Path = clonePath(path.Path)
		out = append(out, path)
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Issue + "|" + out[i].Effect + "|" + out[i].Function + "|" + strings.Join(out[i].Path, "->")
		right := out[j].Issue + "|" + out[j].Effect + "|" + out[j].Function + "|" + strings.Join(out[j].Path, "->")
		return left < right
	})
	return out
}

func clonePath(path []string) []string {
	return append([]string(nil), path...)
}

func cloneStrings(values []string) []string {
	return append([]string{}, values...)
}
