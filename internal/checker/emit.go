package checker

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func BuildSemanticIR(phase Phase) (SemanticIR, string, error) {
	effects := append([]string(nil), phase.Effects...)
	capabilities := append([]string(nil), phase.Capabilities...)
	riskEffects := append([]string(nil), phase.RiskEffects...)
	precedence := append([]string(nil), phase.Precedence...)
	unknownFields := append([]string(nil), phase.UnknownFields...)
	plan := append([]string(nil), phase.Plan...)
	sort.Strings(effects)
	sort.Strings(capabilities)
	sort.Strings(riskEffects)
	sort.Strings(unknownFields)
	grants := append([]Grant(nil), phase.Grants...)
	for index := range grants {
		grants[index].Effects = append([]string(nil), grants[index].Effects...)
		sort.Strings(grants[index].Effects)
	}
	sort.Slice(grants, func(i, j int) bool {
		left := grants[i].CaseID + "|" + grants[i].Capability
		right := grants[j].CaseID + "|" + grants[j].Capability
		return left < right
	})
	cases := append([]Case(nil), phase.Cases...)
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	semantic := SemanticIR{
		Schema: "gooo/capability-effect-checker/semantic-ir/v1", PhaseDigest: phase.Digest,
		Effects: effects, Capabilities: capabilities, Precedence: precedence,
		UnknownFields: unknownFields, RiskEffects: riskEffects,
		Classifications: cloneStringMap(phase.Classifications), Diagnostics: cloneDiagnosticMap(phase.Diagnostics),
		Denominator: phase.Denominator, Cases: cases, Grants: grants, Plan: plan, Generation: phase.Generation,
	}
	data, err := json.Marshal(semantic)
	if err != nil {
		return SemanticIR{}, "", err
	}
	return semantic, digestBytes(data), nil
}

func PrettyJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func EmitChecker(phase Phase, semantic SemanticIR, semanticDigest string) []byte {
	var out strings.Builder
	out.WriteString("// Code generated from .gooo/capability-effect-checker.gooo; DO NOT EDIT.\n")
	out.WriteString("package generated\n\n")
	out.WriteString("const PhaseDigest = ")
	out.WriteString(strconv.Quote(phase.Digest))
	out.WriteString("\nconst SemanticIRDigest = ")
	out.WriteString(strconv.Quote(semanticDigest))
	out.WriteString("\nconst DecisionClosed = ")
	out.WriteString(strconv.Quote(phase.Precedence[2]))
	out.WriteString("\nconst DecisionRefuted = ")
	out.WriteString(strconv.Quote(phase.Classifications["declared_grant_omission"]))
	out.WriteString("\n\n")
	out.WriteString("type Result struct {\n\tDecision string\n\tInferredEffects []string\n\tMissingEffects []string\n}\n\n")
	out.WriteString("var DeclaredEffects = []string{")
	writeQuotedList(&out, semantic.Effects)
	out.WriteString("}\n\n")
	out.WriteString("var Grants = map[string][]string{\n")
	for _, grant := range semantic.Grants {
		out.WriteString("\t")
		out.WriteString(strconv.Quote(grant.CaseID + "|" + grant.Capability))
		out.WriteString(": []string{")
		writeQuotedList(&out, grant.Effects)
		out.WriteString("},\n")
	}
	out.WriteString("}\n\n")
	out.WriteString("func InferEffects(root string, calls map[string][]string, direct map[string][]string) []string {\n")
	out.WriteString("\tseen := map[string]bool{}\n\teffects := map[string]bool{}\n")
	out.WriteString("\tvar visit func(string)\n\tvisit = func(name string) {\n")
	out.WriteString("\t\tif seen[name] { return }\n\t\tseen[name] = true\n")
	out.WriteString("\t\tfor _, effect := range direct[name] { effects[effect] = true }\n")
	out.WriteString("\t\tchildren := append([]string(nil), calls[name]...)\n")
	out.WriteString("\t\tfor i := 0; i < len(children); i++ { for j := i + 1; j < len(children); j++ { if children[j] < children[i] { children[i], children[j] = children[j], children[i] } } }\n")
	out.WriteString("\t\tfor _, child := range children { visit(child) }\n\t}\n")
	out.WriteString("\tvisit(root)\n\tout := make([]string, 0, len(effects))\n\tfor effect := range effects { out = append(out, effect) }\n")
	out.WriteString("\tfor i := 0; i < len(out); i++ { for j := i + 1; j < len(out); j++ { if out[j] < out[i] { out[i], out[j] = out[j], out[i] } } }\n")
	out.WriteString("\treturn out\n}\n\n")
	out.WriteString("func Check(caseID, root string, calls map[string][]string, direct map[string][]string) Result {\n")
	out.WriteString("\tinferred := InferEffects(root, calls, direct)\n\tallowed := map[string]bool{}\n")
	out.WriteString("\tfor _, effect := range Grants[caseID+\"|\"+root] { allowed[effect] = true }\n")
	out.WriteString("\tmissing := make([]string, 0)\n\tfor _, effect := range inferred { if !allowed[effect] { missing = append(missing, effect) } }\n")
	out.WriteString("\tdecision := DecisionClosed\n\tif len(missing) > 0 { decision = DecisionRefuted }\n")
	out.WriteString("\treturn Result{Decision: decision, InferredEffects: inferred, MissingEffects: missing}\n}\n")
	return []byte(out.String())
}

func writeQuotedList(out *strings.Builder, values []string) {
	for index, value := range values {
		if index > 0 {
			out.WriteString(", ")
		}
		out.WriteString(strconv.Quote(value))
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneDiagnosticMap(source map[string]Diagnostic) map[string]Diagnostic {
	clone := make(map[string]Diagnostic, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func digestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal digest: %w", err)
	}
	return digestBytes(data), nil
}
