package checker

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"
)

// Effect is intentionally a closed type. The .gooo vocabulary is the only
// authority for which Effect values are accepted by the typechecker.
type Effect string

const (
	EffectReadInput            Effect = "READ_INPUT"
	EffectNetworkReadPinned    Effect = "NETWORK_READ_PINNED"
	EffectGenerateCallerOutput Effect = "GENERATE_CALLER_OUTPUT"
	EffectRepositoryWrite      Effect = "REPOSITORY_WRITE"
	EffectRemoteMutation       Effect = "REMOTE_MUTATION"
	EffectDestructiveDelete    Effect = "DESTRUCTIVE_DELETE"
)

var CanonicalEffects = []Effect{
	EffectReadInput,
	EffectNetworkReadPinned,
	EffectGenerateCallerOutput,
	EffectRepositoryWrite,
	EffectRemoteMutation,
	EffectDestructiveDelete,
}

// EffectSet is used for all capability/effect comparisons. Slices are used
// only at the parser and JSON boundaries so emitted vectors remain stable.
type EffectSet map[Effect]struct{}

func NewEffectSet(values []string) EffectSet {
	set := make(EffectSet, len(values))
	for _, value := range values {
		set[Effect(value)] = struct{}{}
	}
	return set
}

func (set EffectSet) Contains(effect string) bool {
	_, ok := set[Effect(effect)]
	return ok
}

func (set EffectSet) Add(effect string) {
	set[Effect(effect)] = struct{}{}
}

func (set EffectSet) Union(other EffectSet) EffectSet {
	out := make(EffectSet, len(set)+len(other))
	for effect := range set {
		out[effect] = struct{}{}
	}
	for effect := range other {
		out[effect] = struct{}{}
	}
	return out
}

func (set EffectSet) Difference(other EffectSet) []string {
	out := make([]string, 0)
	for effect := range set {
		if _, ok := other[effect]; !ok {
			out = append(out, string(effect))
		}
	}
	sortStrings(out)
	return out
}

func (set EffectSet) Sorted() []string {
	out := make([]string, 0, len(set))
	for effect := range set {
		out = append(out, string(effect))
	}
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

const (
	StageMetacode      = "METACODE"
	StageGeneratedCode = "GENERATED_CODE"
	StageRuntime       = "RUNTIME"
)

type Phase struct {
	Schema          string
	Version         int
	Package         string
	Namespace       string
	Digest          string
	Effects         []string
	Capabilities    []string
	Stages          []string
	StageEdges      []StageEdge
	OutputScope     string
	Precedence      []string
	UnknownFields   []string
	RiskEffects     []string
	Classifications map[string]string
	Diagnostics     map[string]Diagnostic
	Denominator     int
	Cases           []Case
	Grants          []Grant
	EffectSummaries []EffectSummary
	Attenuations    []Attenuation
	Activities      []Activity
	Cells           []Cell
	ProofQuotas     map[string]int
	IndicatorQuotas map[string]int
	FixedPoint      string
	AuthorityPolicy Authority
	Plan            []string
	Generation      GenerationPlan
}

type StageEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Activity struct {
	CaseID     string `json:"case_id"`
	Stage      string `json:"stage"`
	Capability string `json:"capability"`
	Function   string `json:"function"`
	Name       string `json:"name"`
}

type EffectSummary struct {
	CaseID     string   `json:"case_id"`
	Stage      string   `json:"stage"`
	Capability string   `json:"capability"`
	Effects    []string `json:"effects"`
}

type Attenuation struct {
	CaseID     string   `json:"case_id"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Capability string   `json:"capability"`
	Effects    []string `json:"effects"`
}

type Cell struct {
	ID        string `json:"id"`
	CaseID    string `json:"case_id"`
	Cohort    string `json:"cohort"`
	Proof     string `json:"proof"`
	Indicator string `json:"indicator"`
	Activity  string `json:"activity"`
}

type Diagnostic struct {
	Stage         string `json:"stage"`
	Step          string `json:"step"`
	Reason        string `json:"reason"`
	UnknownClass  string `json:"unknown_class"`
	NextOperation string `json:"next_operation"`
	BlockedBy     string `json:"blocked_by"`
}

type Case struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Expected   string `json:"expected"`
	Root       string `json:"root"`
	Capability string `json:"capability"`
	Cohort     string `json:"cohort,omitempty"`
}

type Grant struct {
	CaseID     string   `json:"case_id"`
	Stage      string   `json:"stage,omitempty"`
	Capability string   `json:"capability"`
	Effects    []string `json:"effects"`
}

type GenerationPlan struct {
	OutputBoundary string   `json:"output_boundary"`
	Steps          []string `json:"steps"`
}

type Fixture struct {
	Schema         string              `json:"schema"`
	Version        int                 `json:"version"`
	Scenario       string              `json:"scenario"`
	Root           string              `json:"root"`
	Functions      map[string]Function `json:"functions"`
	OutputScope    string              `json:"output_scope,omitempty"`
	NetworkPins    []NetworkPin        `json:"network_pins,omitempty"`
	AmbiguousCalls []AmbiguousCall     `json:"ambiguous_calls,omitempty"`
	GrantClaims    []GrantClaim        `json:"grant_claims,omitempty"`
}

type Function struct {
	Name          string   `json:"name"`
	DirectEffects []string `json:"direct_effects"`
	Calls         []string `json:"calls"`
	Oracles       []Oracle `json:"oracles,omitempty"`
}

type Oracle struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type NetworkPin struct {
	Function string `json:"function"`
	Pin      string `json:"pin"`
}

type AmbiguousCall struct {
	Caller     string   `json:"caller"`
	Candidates []string `json:"candidates"`
}

type GrantClaim struct {
	Function   string   `json:"function"`
	Capability string   `json:"capability"`
	Effects    []string `json:"effects"`
}

// Unknown intentionally has exactly six fields. Every incomplete boundary is
// represented through this shape, never by an empty or implicit status.
type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Refutation struct {
	Stage          string `json:"stage"`
	Step           string `json:"step"`
	Reason         string `json:"reason"`
	Counterexample string `json:"counterexample"`
}

type CallPath struct {
	Issue    string   `json:"issue"`
	Effect   string   `json:"effect,omitempty"`
	Function string   `json:"function"`
	Path     []string `json:"path"`
}

type StageResult struct {
	Stage           string   `json:"stage"`
	Capability      string   `json:"capability"`
	Function        string   `json:"function"`
	DirectEffects   []string `json:"direct_effects"`
	InferredEffects []string `json:"inferred_effects"`
	DeclaredGrant   []string `json:"declared_grant"`
	DeclaredSummary []string `json:"declared_summary"`
	MissingEffects  []string `json:"missing_effects"`
}

type AttenuationEvidence struct {
	FromStage       string   `json:"from_stage"`
	ToStage         string   `json:"to_stage"`
	Capability      string   `json:"capability"`
	AllowedEffects  []string `json:"allowed_effects"`
	ObservedEffects []string `json:"observed_effects"`
	Path            []string `json:"path"`
}

type CaseResult struct {
	ID                      string                `json:"id"`
	Expected                string                `json:"expected"`
	Cohort                  string                `json:"cohort,omitempty"`
	Decision                string                `json:"decision"`
	InferredEffects         []string              `json:"inferred_effects"`
	DeclaredRootEffects     []string              `json:"declared_root_effects"`
	MissingRootEffects      []string              `json:"missing_root_effects"`
	UnknownGeneratedEffects []string              `json:"unknown_generated_effects,omitempty"`
	Unknowns                []Unknown             `json:"unknowns"`
	Refutations             []Refutation          `json:"refutations"`
	OffendingCallPaths      []CallPath            `json:"offending_call_paths"`
	StageResults            []StageResult         `json:"stage_results,omitempty"`
	AttenuationEvidence     []AttenuationEvidence `json:"attenuation_edges,omitempty"`
	ExternalDependencies    []string              `json:"external_dependencies,omitempty"`
	OutputScope             string                `json:"output_scope,omitempty"`
}

type SemanticIR struct {
	Schema          string                `json:"schema"`
	PhaseDigest     string                `json:"phase_digest"`
	Effects         []string              `json:"effects"`
	Capabilities    []string              `json:"capabilities"`
	Stages          []string              `json:"stages"`
	StageEdges      []StageEdge           `json:"stage_edges"`
	OutputScope     string                `json:"output_scope"`
	Precedence      []string              `json:"precedence"`
	UnknownFields   []string              `json:"unknown_fields"`
	RiskEffects     []string              `json:"risk_effects"`
	Classifications map[string]string     `json:"classifications"`
	Diagnostics     map[string]Diagnostic `json:"diagnostics"`
	Denominator     int                   `json:"denominator"`
	Cases           []Case                `json:"cases"`
	Grants          []Grant               `json:"grants"`
	EffectSummaries []EffectSummary       `json:"effect_summaries"`
	Attenuations    []Attenuation         `json:"attenuation_edges"`
	Activities      []Activity            `json:"activities"`
	Cells           []Cell                `json:"cells"`
	ProofQuotas     map[string]int        `json:"proof_quotas"`
	IndicatorQuotas map[string]int        `json:"indicator_quotas"`
	FixedPoint      string                `json:"fixed_point"`
	AuthorityPolicy Authority             `json:"authority_policy"`
	Plan            []string              `json:"plan"`
	Generation      GenerationPlan        `json:"generation"`
}

type Summary struct {
	Generated int `json:"generated"`
	Closed    int `json:"closed"`
	Unknown   int `json:"unknown"`
	Refuted   int `json:"refuted"`
	Failed    int `json:"failed"`
}

type Artifact struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	Digest string `json:"digest"`
}

type EvidenceStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type Evidence struct {
	Improvement     EvidenceStatus `json:"improvement"`
	ExternalUtility EvidenceStatus `json:"external_utility"`
}

type EffectVector struct {
	ID      string   `json:"id"`
	Effects []string `json:"effects"`
}

type RunReport struct {
	Schema             string         `json:"schema"`
	PhaseDigest        string         `json:"phase_digest"`
	SemanticIRDigest   string         `json:"semantic_ir_digest"`
	Precedence         []string       `json:"precedence"`
	FixedPoint         string         `json:"fixed_point"`
	Generation         GenerationPlan `json:"generation"`
	Summary            Summary        `json:"summary"`
	DecisionVector     []string       `json:"decision_vector"`
	EffectVectors      []EffectVector `json:"effect_vectors"`
	ProofCounts        map[string]int `json:"proof_counts"`
	IndicatorCounts    map[string]int `json:"indicator_counts"`
	CohortCounts       map[string]int `json:"cohort_counts"`
	Cells              []Cell         `json:"cells"`
	Cases              []CaseResult   `json:"cases"`
	Authority          Authority      `json:"authority"`
	Evidence           Evidence       `json:"evidence"`
	GeneratedArtifacts []Artifact     `json:"generated_artifacts"`
	Artifacts          []Artifact     `json:"artifacts"`
}

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}
