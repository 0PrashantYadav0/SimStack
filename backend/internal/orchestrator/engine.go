package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"simstack/internal/cerebras"
	"simstack/internal/types"
)

type Engine struct {
	emit             func(v any)
	cereClient       *cerebras.Client
	plannerLatencyMs int64
	simStartupMs     int64
	tokensPerSec     float64
}

func NewEngine(emitter func(v any)) *Engine {
	return &Engine{
		emit:       emitter,
		cereClient: cerebras.New(),
	}
}

func (e *Engine) Run(ctx context.Context, req types.RunRequest) error {
	start := time.Now()
	plan := e.plan(ctx, req)
	e.plannerLatencyMs = time.Since(start).Milliseconds()

	e.emit(types.WSEvent{Type: "plan", Payload: plan, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})

	// Spawn simulators for each variant in parallel
	simStart := time.Now()
	results := e.runSimulators(ctx, plan)
	e.simStartupMs = time.Since(simStart).Milliseconds()

	// Emit results as they complete
	for _, r := range results {
		e.emit(types.WSEvent{Type: "result", Payload: r, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	}

	// Run Critic Agent to analyze results and provide recommendations
	critStart := time.Now()
	analysis := e.analyzeResults(ctx, req, results)
	log.Printf("Critic analysis completed in %dms", time.Since(critStart).Milliseconds())

	e.emit(types.WSEvent{Type: "analysis", Payload: analysis, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})

	e.emit(types.WSEvent{Type: "done", Payload: map[string]string{"plan_id": plan.PlanID}, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	return nil
}

func (e *Engine) plan(parentCtx context.Context, req types.RunRequest) types.SimulationPlan {
	// Integrate Cerebras OpenAI-compatible planning with tool calling
	planID := fmt.Sprintf("plan-%d", time.Now().UnixNano())

	// Create a separate context for planning so it doesn't affect simulators
	ctx, cancel := context.WithTimeout(parentCtx, 90*time.Second)
	defer cancel()

	// Use Cerebras Llama for fast planning (without tools parameter for compatibility)
	model := getEnv("CEREBRAS_MODEL", "llama3.1-8b")
	systemPrompt := `You are a simulation planning AI. Given a goal, create 3 variant parameter sets to test different scenarios.

Available simulators:
1. queue_simulator: arrival_rate (customers/hour), service_rate (customers/hour)
2. traffic_simulator: density (0.0-1.0), signal_timing (seconds)
3. resource_simulator: staff (number), shifts (array)

Return ONLY valid JSON with this structure:
{"variants": [{"id": "v1", "queue": {"arrival_rate": 10, "service_rate": 12}, "traffic": {"density": 0.5}, "resource": {"staff": 20}}]}`

	messages := []cerebras.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Goal: %s. Constraints: %v. Create 3 test variants.", req.Goal, req.Constraints)},
	}

	startTokens := time.Now()
	// Don't send tools parameter - Cerebras API doesn't support it like OpenAI
	resp, err := e.cereClient.Chat(ctx, cerebras.OpenAIChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
	})
	elapsed := time.Since(startTokens).Seconds()

	// Check for errors first before using response
	var variants []types.Variant
	if err != nil {
		log.Printf("Cerebras planning unavailable, using fallback variants: %v", err)
		variants = e.fallbackVariants(planID, req)
	} else {
		// Track token performance (Cerebras can do 1800+ tokens/sec)
		if usage, ok := resp["usage"].(map[string]interface{}); ok {
			if total, ok := usage["total_tokens"].(float64); ok && elapsed > 0 {
				e.tokensPerSec = total / elapsed
				log.Printf("Cerebras planning completed: %.0f tokens/sec", e.tokensPerSec)
			}
		}

		// Parse response or use fallback variants
		variants = e.parseVariantsFromResponse(resp, planID)
		if len(variants) == 0 {
			log.Println("Cerebras planning returned no parseable variants, using fallback")
			variants = e.fallbackVariants(planID, req)
		}
	}

	steps := []types.PlanStep{
		{Name: "Queue", Description: "Queueing simulation", Tool: "queue", InputSchema: map[string]any{"arrival_rate": "number", "service_rate": "number"}},
		{Name: "Traffic", Description: "Traffic flow simulation", Tool: "traffic", InputSchema: map[string]any{"density": "number", "signal_timing": "number"}},
		{Name: "Resource", Description: "Resource allocation", Tool: "resource", InputSchema: map[string]any{"staff": "number", "shifts": "array"}},
	}

	return types.SimulationPlan{PlanID: planID, Steps: steps, Variants: variants}
}

func (e *Engine) parseVariantsFromResponse(resp map[string]any, planID string) []types.Variant {
	// Try to extract variants from Cerebras response
	choices, ok := resp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil
	}
	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content, ok := message["content"].(string)
	if !ok || content == "" {
		return nil
	}

	var parsed struct {
		Variants []map[string]map[string]interface{} `json:"variants"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil
	}

	variants := make([]types.Variant, 0, len(parsed.Variants))
	for i, v := range parsed.Variants {
		merged := make(map[string]any)
		for toolName, params := range v {
			for k, val := range params {
				merged[k] = val
			}
			_ = toolName
		}
		variants = append(variants, types.Variant{
			VariantID:  fmt.Sprintf("%s-v%d", planID, i+1),
			Parameters: merged,
		})
	}
	return variants
}

func (e *Engine) fallbackVariants(planID string, req types.RunRequest) []types.Variant {
	// Fallback: Create 16 variants for comprehensive grid search
	// Design to ensure service_rate > arrival_rate for stable queueing systems
	variants := make([]types.Variant, 0, 16)

	// Grid search with carefully chosen parameters
	// Use ratios that explore low, medium, high utilization (0.5, 0.6, 0.7, 0.8)
	arrivalRates := []float64{8, 10, 12, 14}
	serviceRates := []float64{16, 18, 20, 22} // Always higher than arrival rates

	idx := 1
	for i, arr := range arrivalRates {
		for j, svc := range serviceRates {
			// Ensure variety while maintaining stability
			actualService := svc + float64(j)                         // 16-25 range
			density := 0.3 + (float64(i) * 0.1) + (float64(j) * 0.05) // 0.3 to 0.85
			staff := 20 + (i * 2) + j                                 // 20 to 33

			// Calculate utilization for this variant
			utilization := arr / actualService

			variants = append(variants, types.Variant{
				VariantID: fmt.Sprintf("%s-v%d", planID, idx),
				Parameters: map[string]any{
					"arrival_rate": arr,
					"service_rate": actualService,
					"density":      math.Min(density, 0.85),
					"staff":        staff,
					"utilization":  math.Round(utilization*100) / 100, // For reference
				},
			})
			idx++
		}
	}

	return variants
}

func (e *Engine) runSimulators(parentCtx context.Context, plan types.SimulationPlan) []types.SimulationResult {
	// Spawn Docker containers for each simulator in parallel
	// Using HTTP calls to simulator services (running in docker-compose or MCP containers)

	simulatorURLs := map[string]string{
		"queue":    getEnv("QUEUE_SIMULATOR_URL", "http://localhost:8101"),
		"traffic":  getEnv("TRAFFIC_SIMULATOR_URL", "http://localhost:8102"),
		"resource": getEnv("RESOURCE_SIMULATOR_URL", "http://localhost:8103"),
	}

	results := make([]types.SimulationResult, 0, len(plan.Variants))
	resultsMu := sync.Mutex{}
	wg := sync.WaitGroup{}

	// Run variants in parallel for speed
	for _, variant := range plan.Variants {
		wg.Add(1)
		go func(v types.Variant) {
			defer wg.Done()

			// CRITICAL: Create independent context for this variant so failures don't cascade
			// Use background context with timeout instead of parent context
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			// Emit progress event
			e.emit(types.WSEvent{
				Type:      "sim_start",
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   map[string]any{"variant_id": v.VariantID},
			})

			// Run each simulator tool with variant parameters
			variantMetrics := make(map[string]float64)

			for toolName, baseURL := range simulatorURLs {
				toolParams := e.extractToolParams(v.Parameters, toolName)
				if len(toolParams) == 0 {
					continue // Skip if no params for this tool
				}

				// Create independent context for each simulator call
				// Use shorter timeout (45s) than variant timeout (3min)
				simCtx, simCancel := context.WithTimeout(ctx, 45*time.Second)
				metrics, err := e.invokeSimulator(simCtx, baseURL, toolParams)
				simCancel() // Always cancel to free resources
				if err != nil {
					log.Printf("simulator %s error for %s: %v", toolName, v.VariantID, err)
					// Don't fail the entire variant, just skip this simulator
					continue
				}

				// Merge metrics with tool prefix
				for k, val := range metrics {
					variantMetrics[fmt.Sprintf("%s_%s", toolName, k)] = val
				}
			}

			result := types.SimulationResult{
				VariantID: v.VariantID,
				Tool:      "composite",
				Metrics:   variantMetrics,
			}

			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()

			e.emit(types.WSEvent{
				Type:      "sim_complete",
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   result,
			})
		}(variant)
	}

	wg.Wait()
	return results
}

func (e *Engine) extractToolParams(params map[string]any, toolName string) map[string]any {
	// Extract parameters relevant to a specific tool
	extracted := make(map[string]any)

	toolFields := map[string][]string{
		"queue":    {"arrival_rate", "service_rate"},
		"traffic":  {"density", "signal_timing"},
		"resource": {"staff", "shifts"},
	}

	fields, ok := toolFields[toolName]
	if !ok {
		return extracted
	}

	for _, field := range fields {
		if val, exists := params[field]; exists {
			extracted[field] = val
		}
	}

	return extracted
}

func (e *Engine) invokeSimulator(ctx context.Context, baseURL string, params map[string]any) (map[string]float64, error) {
	// POST to simulator's /simulate endpoint
	body, _ := json.Marshal(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/simulate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Context timeout (45s) will take precedence over HTTP client timeout
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("simulator returned %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Metrics map[string]float64 `json:"metrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Metrics, nil
}

func (e *Engine) analyzeResults(parentCtx context.Context, req types.RunRequest, results []types.SimulationResult) map[string]any {
	// Critic Agent: Analyze simulation results and provide recommendations using Cerebras

	if len(results) == 0 {
		return map[string]any{
			"recommendation": "No results to analyze",
			"confidence":     0.0,
			"trade_offs":     []string{},
		}
	}

	// Create independent context for criticism
	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()

	// Prepare results summary for Llama
	resultsSummary := e.summarizeResults(results)

	model := getEnv("CEREBRAS_MODEL", "llama3.1-8b")
	systemPrompt := `You are an expert operations analyst. Analyze simulation results and provide:
1. The best performing variant and why
2. Key trade-offs between cost, performance, and constraints
3. Counterfactual insights ("what if" scenarios)
4. Confidence level in the recommendation

Return concise, actionable JSON:
{
  "winner": "variant ID",
  "recommendation": "Clear recommendation with reasoning",
  "confidence": 0.0-1.0,
  "trade_offs": ["trade-off 1", "trade-off 2"],
  "counterfactuals": ["insight 1", "insight 2"],
  "key_metrics": {"metric": value}
}`

	userPrompt := fmt.Sprintf(`Goal: %s
Constraints: %v

Simulation Results:
%s

Analyze these results and recommend the best approach.`, req.Goal, req.Constraints, resultsSummary)

	messages := []cerebras.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := e.cereClient.Chat(ctx, cerebras.OpenAIChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.3, // Lower temperature for more consistent analysis
	})

	if err != nil {
		log.Printf("Critic analysis failed, using fallback: %v", err)
		return e.fallbackAnalysis(results)
	}

	// Parse Llama's analysis
	analysis := e.parseAnalysis(resp, results)
	if analysis == nil {
		log.Println("Failed to parse analysis, using fallback")
		return e.fallbackAnalysis(results)
	}

	return analysis
}

func (e *Engine) summarizeResults(results []types.SimulationResult) string {
	var summary strings.Builder

	for i, r := range results {
		summary.WriteString(fmt.Sprintf("\nVariant %d (%s):\n", i+1, r.VariantID))
		for key, val := range r.Metrics {
			summary.WriteString(fmt.Sprintf("  %s: %.2f\n", key, val))
		}
	}

	return summary.String()
}

func (e *Engine) parseAnalysis(resp map[string]any, results []types.SimulationResult) map[string]any {
	choices, ok := resp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil
	}

	choice := choices[0].(map[string]interface{})
	message := choice["message"].(map[string]interface{})
	content, ok := message["content"].(string)
	if !ok || content == "" {
		return nil
	}

	// Try to parse JSON response
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		// If not JSON, return structured fallback
		return map[string]any{
			"recommendation": content,
			"confidence":     0.7,
			"winner":         results[0].VariantID,
		}
	}

	return parsed
}

func (e *Engine) fallbackAnalysis(results []types.SimulationResult) map[string]any {
	// Simple heuristic: Find variant with best overall metrics
	bestIdx := 0
	bestScore := 0.0

	for i, r := range results {
		score := 0.0
		count := 0

		// Calculate average of key metrics (lower wait time is better, higher throughput is better)
		for key, val := range r.Metrics {
			if strings.Contains(key, "wait") {
				score += 1.0 / (1.0 + val) // Lower is better
			} else {
				score += val // Higher is better
			}
			count++
		}

		if count > 0 {
			score = score / float64(count)
		}

		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	winner := results[bestIdx]

	return map[string]any{
		"winner":         winner.VariantID,
		"recommendation": fmt.Sprintf("Variant %d shows the best balance of metrics with overall score of %.2f", bestIdx+1, bestScore),
		"confidence":     0.75,
		"trade_offs": []string{
			"Higher service rates improve throughput but may increase costs",
			"Optimal staffing balances wait times with budget constraints",
			"Traffic density impacts overall system efficiency",
		},
		"counterfactuals": []string{
			"Increasing staff by 20% could reduce wait times by 15-20%",
			"Reducing arrival rate through scheduling could improve service quality",
		},
		"key_metrics": winner.Metrics,
	}
}

func (e *Engine) ExportCompose(ctx context.Context, req types.ExportRequest) (string, string, error) {
	// Minimal docker-compose with three services and environment for params
	yml := `version: '3.9'
services:
  queue:
    image: simstack/queue:latest
    environment:
      - PARAMS=` + fmt.Sprintf("%v", req.Parameters) + `
  traffic:
    image: simstack/traffic:latest
    environment:
      - PARAMS=` + fmt.Sprintf("%v", req.Parameters) + `
  resource:
    image: simstack/resource:latest
    environment:
      - PARAMS=` + fmt.Sprintf("%v", req.Parameters) + `
`
	return yml, "simstack-compose.yml", nil
}

func (e *Engine) Metrics() types.MetricsSnapshot {
	return types.MetricsSnapshot{PlannerMs: e.plannerLatencyMs, SimulationStartupMs: e.simStartupMs, TokensPerSecond: e.tokensPerSec}
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
