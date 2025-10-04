package orchestrator

import (
	"testing"

	"simstack/internal/types"
)

func TestExtractToolParams(t *testing.T) {
	e := NewEngine(func(v any) {})

	params := map[string]any{
		"arrival_rate": 10.0,
		"service_rate": 12.0,
		"density":      0.5,
		"staff":        20,
	}

	queueParams := e.extractToolParams(params, "queue")
	if len(queueParams) != 2 {
		t.Errorf("expected 2 queue params, got %d", len(queueParams))
	}
	if queueParams["arrival_rate"] != 10.0 {
		t.Error("missing arrival_rate")
	}

	trafficParams := e.extractToolParams(params, "traffic")
	if len(trafficParams) != 1 {
		t.Errorf("expected 1 traffic param, got %d", len(trafficParams))
	}
	if trafficParams["density"] != 0.5 {
		t.Error("missing density")
	}
}

func TestFallbackVariants(t *testing.T) {
	e := NewEngine(func(v any) {})
	req := types.RunRequest{Goal: "test"}

	variants := e.fallbackVariants("test-plan", req)

	if len(variants) != 3 {
		t.Errorf("expected 3 variants, got %d", len(variants))
	}

	for _, v := range variants {
		if v.VariantID == "" {
			t.Error("variant missing ID")
		}
		if len(v.Parameters) == 0 {
			t.Error("variant missing parameters")
		}
	}
}

func TestGetEnv(t *testing.T) {
	result := getEnv("NONEXISTENT_VAR_12345", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}
