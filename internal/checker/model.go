package checker

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"
)

type Phase struct {
	Schema          string
	Package         string
	Namespace       string
	Digest          string
	Effects         []string
	Capabilities    []string
	Precedence      []string
	UnknownFields   []string
	RiskEffects     []string
	Classifications map[string]string
	Diagnostics     map[string]Diagnostic
	Denominator     int
	Cases           []Case
	Grants          []Grant
	Plan            []string
	Generation      GenerationPlan
}

type Diagnostic struct {
	Stage         string
	Step          string
	Reason        string
	UnknownClass  string
	NextOperation string
	BlockedBy     string
}

type Case struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Expected   string `json:"expected"`
	Root       string `json:"root"`
	Capability string `json:"capability"`
}

type Grant struct {
	CaseID     string   `json:"case_id"`
	Capability string   `json:"capability"`
	Effects    []string `json:"effects"`
}

type GenerationPlan struct {
	OutputBoundary string   `json:"output_boundary"`
	Steps          []string `json:"steps"`
}

type Fixture struct {
	Schema    string              `json:"schema"`
	Scenario  string              `json:"scenario"`
	Root      string              `json:"root"`
	Functions map[string]Function `json:"functions"`
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

type CaseResult struct {
	ID                   string       `json:"id"`
	Expected             string       `json:"expected"`
	Decision             string       `json:"decision"`
	InferredEffects      []string     `json:"inferred_effects"`
	DeclaredRootEffects  []string     `json:"declared_root_effects"`
	MissingRootEffects   []string     `json:"missing_root_effects"`
	Unknowns             []Unknown    `json:"unknowns"`
	Refutations          []Refutation `json:"refutations"`
	OffendingCallPaths   []CallPath   `json:"offending_call_paths"`
	ExternalDependencies []string     `json:"external_dependencies,omitempty"`
}

type SemanticIR struct {
	Schema          string                `json:"schema"`
	PhaseDigest     string                `json:"phase_digest"`
	Effects         []string              `json:"effects"`
	Capabilities    []string              `json:"capabilities"`
	Precedence      []string              `json:"precedence"`
	UnknownFields   []string              `json:"unknown_fields"`
	RiskEffects     []string              `json:"risk_effects"`
	Classifications map[string]string     `json:"classifications"`
	Diagnostics     map[string]Diagnostic `json:"diagnostics"`
	Denominator     int                   `json:"denominator"`
	Cases           []Case                `json:"cases"`
	Grants          []Grant               `json:"grants"`
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

type RunReport struct {
	Schema             string         `json:"schema"`
	PhaseDigest        string         `json:"phase_digest"`
	SemanticIRDigest   string         `json:"semantic_ir_digest"`
	Precedence         []string       `json:"precedence"`
	Generation         GenerationPlan `json:"generation"`
	Summary            Summary        `json:"summary"`
	Cases              []CaseResult   `json:"cases"`
	Authority          Authority      `json:"authority"`
	GeneratedArtifacts []Artifact     `json:"generated_artifacts"`
	Artifacts          []Artifact     `json:"artifacts"`
}

type Authority struct {
	RepositoryWrites          int `json:"repository_writes"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}
