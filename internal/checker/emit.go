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
	stages := append([]string(nil), phase.Stages...)
	riskEffects := append([]string(nil), phase.RiskEffects...)
	precedence := append([]string(nil), phase.Precedence...)
	unknownFields := append([]string(nil), phase.UnknownFields...)
	plan := append([]string(nil), phase.Plan...)
	sort.Strings(effects)
	sort.Strings(capabilities)
	sort.Strings(stages)
	sort.Strings(riskEffects)
	sort.Strings(unknownFields)
	grants := append([]Grant(nil), phase.Grants...)
	for index := range grants {
		grants[index].Effects = append([]string(nil), grants[index].Effects...)
		sort.Strings(grants[index].Effects)
	}
	sort.Slice(grants, func(i, j int) bool {
		left := grants[i].CaseID + "|" + grants[i].Stage + "|" + grants[i].Capability
		right := grants[j].CaseID + "|" + grants[j].Stage + "|" + grants[j].Capability
		return left < right
	})
	cases := append([]Case(nil), phase.Cases...)
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	stageEdges := append([]StageEdge(nil), phase.StageEdges...)
	sort.Slice(stageEdges, func(i, j int) bool {
		return stageEdges[i].From+"|"+stageEdges[i].To < stageEdges[j].From+"|"+stageEdges[j].To
	})
	activities := append([]Activity(nil), phase.Activities...)
	sort.Slice(activities, func(i, j int) bool { return activityKey(activities[i]) < activityKey(activities[j]) })
	summaries := append([]EffectSummary(nil), phase.EffectSummaries...)
	for index := range summaries {
		summaries[index].Effects = append([]string(nil), summaries[index].Effects...)
		sort.Strings(summaries[index].Effects)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaryKey(summaries[i]) < summaryKey(summaries[j]) })
	attenuations := append([]Attenuation(nil), phase.Attenuations...)
	for index := range attenuations {
		attenuations[index].Effects = append([]string(nil), attenuations[index].Effects...)
		sort.Strings(attenuations[index].Effects)
	}
	sort.Slice(attenuations, func(i, j int) bool { return attenuationKey(attenuations[i]) < attenuationKey(attenuations[j]) })
	cells := append([]Cell(nil), phase.Cells...)
	sort.Slice(cells, func(i, j int) bool { return cells[i].ID < cells[j].ID })
	schema := "gooo/capability-effect-checker/semantic-ir/v2"
	if phase.Version == 1 {
		schema = "gooo/capability-effect-checker/semantic-ir/v1"
	}
	semantic := SemanticIR{
		Schema: schema, PhaseDigest: phase.Digest,
		Effects: effects, Capabilities: capabilities, Stages: stages, StageEdges: stageEdges,
		OutputScope: phase.OutputScope, Precedence: precedence, UnknownFields: unknownFields,
		RiskEffects: riskEffects, Classifications: cloneStringMap(phase.Classifications), Diagnostics: cloneDiagnosticMap(phase.Diagnostics),
		Denominator: phase.Denominator, Cases: cases, Grants: grants, EffectSummaries: summaries,
		Attenuations: attenuations, Activities: activities, Cells: cells,
		ProofQuotas: cloneIntMap(phase.ProofQuotas), IndicatorQuotas: cloneIntMap(phase.IndicatorQuotas),
		FixedPoint: phase.FixedPoint, AuthorityPolicy: phase.AuthorityPolicy, Plan: plan, Generation: phase.Generation,
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
	out.WriteString("\nconst FixedPoint = ")
	out.WriteString(strconv.Quote(phase.FixedPoint))
	out.WriteString("\nconst DecisionClosed = ")
	out.WriteString(strconv.Quote(DecisionClosed))
	out.WriteString("\nconst DecisionUnknown = ")
	out.WriteString(strconv.Quote(DecisionUnknown))
	out.WriteString("\nconst DecisionRefuted = ")
	out.WriteString(strconv.Quote(DecisionRefuted))
	out.WriteString("\n\n")
	out.WriteString("type Effect string\n\n")
	for _, effect := range CanonicalEffects {
		out.WriteString("const Effect")
		out.WriteString(effectConstantSuffix(effect))
		out.WriteString(" Effect = ")
		out.WriteString(strconv.Quote(string(effect)))
		out.WriteString("\n")
	}
	out.WriteString("\ntype EffectSet map[Effect]struct{}\n\n")
	out.WriteString("const StageMetacode = ")
	out.WriteString(strconv.Quote(StageMetacode))
	out.WriteString("\nconst StageGeneratedCode = ")
	out.WriteString(strconv.Quote(StageGeneratedCode))
	out.WriteString("\nconst StageRuntime = ")
	out.WriteString(strconv.Quote(StageRuntime))
	out.WriteString("\n\n")
	out.WriteString("type Result struct {\n\tDecision string\n\tInferredEffects []string\n\tMissingEffects []string\n\tUnknownEffects []string\n}\n\n")
	out.WriteString("var DeclaredEffects = []Effect{")
	writeQuotedEffectList(&out, semantic.Effects)
	out.WriteString("}\n\n")
	out.WriteString("var Grants = map[string][]Effect{\n")
	for _, grant := range semantic.Grants {
		out.WriteString("\t")
		out.WriteString(strconv.Quote(grantKey(grant.CaseID, grant.Stage, grant.Capability)))
		out.WriteString(": []Effect{")
		writeQuotedEffectList(&out, grant.Effects)
		out.WriteString("},\n")
	}
	out.WriteString("}\n\n")
	out.WriteString("var ForbiddenEffects = map[Effect]bool{\n")
	for _, effect := range semantic.RiskEffects {
		out.WriteString("\tEffect(")
		out.WriteString(strconv.Quote(effect))
		out.WriteString("): true,\n")
	}
	out.WriteString("}\n\n")
	out.WriteString("var StageForFunction = map[string]string{\n")
	for _, activity := range semantic.Activities {
		out.WriteString("\t")
		out.WriteString(strconv.Quote(activity.CaseID + "|" + activity.Function))
		out.WriteString(": ")
		out.WriteString(strconv.Quote(activity.Stage))
		out.WriteString(",\n")
	}
	out.WriteString("}\n\n")
	out.WriteString("var CapabilityForFunction = map[string]string{\n")
	for _, activity := range semantic.Activities {
		out.WriteString("\t")
		out.WriteString(strconv.Quote(activity.CaseID + "|" + activity.Function))
		out.WriteString(": ")
		out.WriteString(strconv.Quote(activity.Capability))
		out.WriteString(",\n")
	}
	out.WriteString("}\n\n")
	out.WriteString("var AttenuationEdges = map[string][]Effect{\n")
	for _, edge := range semantic.Attenuations {
		out.WriteString("\t")
		out.WriteString(strconv.Quote(edgeKey(edge)))
		out.WriteString(": []Effect{")
		writeQuotedEffectList(&out, edge.Effects)
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
	out.WriteString("\tinferred := InferEffects(root, calls, direct)\n\tkey := caseID + \"|\" + root\n\tstage := StageForFunction[key]\n\tcapability := CapabilityForFunction[key]\n\tgrantKey := caseID + \"|\" + stage + \"|\" + capability\n\tgrant, grantExists := Grants[grantKey]\n\tif stage == \"\" { grant, grantExists = Grants[caseID+\"|\"+root] }\n\tallowed := map[string]bool{}\n\tfor _, effect := range grant { allowed[string(effect)] = true }\n")
	out.WriteString("\tmissing := make([]string, 0)\n\tunknown := make([]string, 0)\n\trefuted := false\n\tfor _, effect := range inferred {\n\t\tif allowed[effect] { continue }\n\t\tmissing = append(missing, effect)\n\t\tif ForbiddenEffects[Effect(effect)] { refuted = true; continue }\n\t\tdeclared := false\n\t\tfor _, known := range DeclaredEffects { if string(known) == effect { declared = true; break } }\n\t\tif !grantExists || !declared { unknown = append(unknown, effect) } else { refuted = true }\n\t}\n\tdecision := DecisionClosed\n\tif len(unknown) > 0 { decision = DecisionUnknown }\n\tif refuted { decision = DecisionRefuted }\n\treturn Result{Decision: decision, InferredEffects: inferred, MissingEffects: missing, UnknownEffects: unknown}\n}\n\n")
	out.WriteString("func AllowsAttenuation(caseID, from, to, capability, effect string) bool {\n")
	out.WriteString("\tfor _, allowed := range AttenuationEdges[caseID+\"|\"+from+\"|\"+to+\"|\"+capability] { if string(allowed) == effect { return true } }\n\treturn false\n}\n")
	return []byte(out.String())
}

func effectConstantSuffix(effect Effect) string {
	value := strings.TrimPrefix(string(effect), "")
	value = strings.ToLower(value)
	parts := strings.Split(value, "_")
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		out.WriteString(strings.ToUpper(part[:1]))
		out.WriteString(part[1:])
	}
	return out.String()
}

func writeQuotedEffectList(out *strings.Builder, values []string) {
	for index, value := range values {
		if index > 0 {
			out.WriteString(", ")
		}
		out.WriteString("Effect(")
		out.WriteString(strconv.Quote(value))
		out.WriteString(")")
	}
}

func grantKey(caseID, stage, capability string) string {
	if stage == "" {
		return caseID + "|" + capability
	}
	return caseID + "|" + stage + "|" + capability
}

func edgeKey(edge Attenuation) string {
	return edge.CaseID + "|" + edge.From + "|" + edge.To + "|" + edge.Capability
}

func activityKey(activity Activity) string {
	return activity.CaseID + "|" + activity.Stage + "|" + activity.Capability + "|" + activity.Function
}

func summaryKey(summary EffectSummary) string {
	return summary.CaseID + "|" + summary.Stage + "|" + summary.Capability
}

func attenuationKey(edge Attenuation) string {
	return edgeKey(edge)
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

func cloneIntMap(source map[string]int) map[string]int {
	clone := make(map[string]int, len(source))
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
