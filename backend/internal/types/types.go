package types

type RunRequest struct {
	Goal        string         `json:"goal"`
	Constraints map[string]any `json:"constraints,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type ExportRequest struct {
	Goal       string         `json:"goal"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type WSEvent struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"ts,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

type SimulationPlan struct {
	PlanID   string     `json:"plan_id"`
	Steps    []PlanStep `json:"steps"`
	Variants []Variant  `json:"variants"`
}

type PlanStep struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Tool        string         `json:"tool"`
	InputSchema map[string]any `json:"input_schema"`
}

type Variant struct {
	VariantID  string         `json:"variant_id"`
	Parameters map[string]any `json:"parameters"`
}

type SimulationResult struct {
	VariantID string             `json:"variant_id"`
	Tool      string             `json:"tool"`
	Metrics   map[string]float64 `json:"metrics"`
	Artifacts map[string]string  `json:"artifacts,omitempty"`
}

type MetricsSnapshot struct {
	PlannerMs           int64   `json:"planner_ms"`
	SimulationStartupMs int64   `json:"simulation_startup_ms"`
	TokensPerSecond     float64 `json:"tokens_per_second"`
}
