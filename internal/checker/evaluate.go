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

	if fixture.Scenario != input.ID {
		refutations = append(refutations, refutationFrom(phase, "declared_grant_omission", fixture.Scenario+" != "+input.ID))
	}
	if fixture.Root != input.Root {
		refutations = append(refutations, refutationFrom(phase, "declared_grant_omission", "root:"+fixture.Root+" expected:"+input.Root))
	}

	reachable, graphUnknowns, graphRefutations, graphPaths := walkGraph(phase, fixture)
	unknowns = append(unknowns, graphUnknowns...)
	refutations = append(refutations, graphRefutations...)
	paths = append(paths, graphPaths...)

	memo := make(map[string][]string)
	inferred := inferEffects(fixture.Root, fixture, memo, make(map[string]bool))
	sort.Strings(inferred)

	rootGrant, rootGrantExists := findGrant(phase, input.ID, fixture.Root)
	if !rootGrantExists {
		unknowns = append(unknowns, unknownFrom(phase, "missing_indirect_grant", fixture.Root+":ROOT_GRANT"))
	} else {
		for _, effect := range inferred {
			if !contains(rootGrant, effect) {
				key := "declared_grant_omission"
				if contains(phase.RiskEffects, effect) {
					key = "forbidden_mutation"
				}
				refutations = append(refutations, refutationFrom(phase, key, fixture.Root+":"+effect))
				paths = append(paths, CallPath{Issue: key, Effect: effect, Function: fixture.Root, Path: clonePath(reachable[fixture.Root])})
			}
		}
	}

	functionNames := make([]string, 0, len(reachable))
	for name := range reachable {
		functionNames = append(functionNames, name)
	}
	sort.Strings(functionNames)
	for _, name := range functionNames {
		functionEffects := inferEffects(name, fixture, memo, make(map[string]bool))
		grant, grantExists := findGrant(phase, input.ID, name)
		if !grantExists {
			if len(functionEffects) == 0 {
				unknowns = append(unknowns, unknownFrom(phase, "missing_indirect_grant", name+":CALLEE_GRANT"))
			}
			for _, effect := range functionEffects {
				path := clonePath(reachable[name])
				if contains(phase.RiskEffects, effect) {
					refutations = append(refutations, refutationFrom(phase, "forbidden_mutation", name+":"+effect))
					paths = append(paths, CallPath{Issue: "forbidden_mutation", Effect: effect, Function: name, Path: path})
				} else {
					unknowns = append(unknowns, unknownFrom(phase, "missing_indirect_grant", name+":"+effect))
					paths = append(paths, CallPath{Issue: "missing_indirect_grant", Effect: effect, Function: name, Path: path})
				}
			}
		} else {
			for _, effect := range functionEffects {
				if contains(grant, effect) {
					continue
				}
				key := "declared_grant_omission"
				if contains(phase.RiskEffects, effect) {
					key = "forbidden_mutation"
				}
				refutations = append(refutations, refutationFrom(phase, key, name+":"+effect))
				paths = append(paths, CallPath{Issue: key, Effect: effect, Function: name, Path: clonePath(reachable[name])})
			}
		}

		function := fixture.Functions[name]
		for _, effect := range function.DirectEffects {
			if !contains(phase.Effects, effect) {
				refutations = append(refutations, refutationFrom(phase, "direct_undeclared_effect", name+":"+effect))
				paths = append(paths, CallPath{Issue: "direct_undeclared_effect", Effect: effect, Function: name, Path: clonePath(reachable[name])})
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
	}

	sort.Strings(externalDependencies)
	sort.Strings(rootGrant)
	missingRoot := difference(inferred, rootGrant)
	return CaseResult{
		ID: input.ID, Expected: input.Expected, Decision: reduce(phase, unknowns, refutations),
		InferredEffects: inferred, DeclaredRootEffects: rootGrant, MissingRootEffects: missingRoot,
		Unknowns: unknowns, Refutations: refutations, OffendingCallPaths: uniqueSortedPaths(paths),
		ExternalDependencies: externalDependencies,
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
	set := make(map[string]bool)
	for _, effect := range function.DirectEffects {
		set[effect] = true
	}
	calls := append([]string(nil), function.Calls...)
	sort.Strings(calls)
	for _, child := range calls {
		for _, effect := range inferEffects(child, fixture, memo, visiting) {
			set[effect] = true
		}
	}
	delete(visiting, name)
	effects := make([]string, 0, len(set))
	for effect := range set {
		effects = append(effects, effect)
	}
	sort.Strings(effects)
	memo[name] = append([]string(nil), effects...)
	return effects
}

func findGrant(phase Phase, caseID, function string) ([]string, bool) {
	if function == "" {
		return nil, false
	}
	for _, grant := range phase.Grants {
		if grant.CaseID == caseID && grant.Capability == function {
			effects := append([]string(nil), grant.Effects...)
			sort.Strings(effects)
			return effects, true
		}
	}
	return nil, false
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
