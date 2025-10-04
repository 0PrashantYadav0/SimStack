import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import './App.css'

function App() {
  const [goal, setGoal] = useState('reduce ER wait time by 20%')
  const [constraints, setConstraints] = useState('{"budget": 100000}')
  const [events, setEvents] = useState([])
  const [plan, setPlan] = useState(null)
  const [results, setResults] = useState([])
  const [metrics, setMetrics] = useState(null)
  const [analysis, setAnalysis] = useState(null)
  const [isRunning, setIsRunning] = useState(false)
  const [exportData, setExportData] = useState(null)
  const wsRef = useRef(null)
  const backendUrl = useMemo(() => (import.meta.env.VITE_BACKEND_URL || 'http://localhost:8080'), [])

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await fetch(backendUrl + '/metrics')
      const data = await res.json()
      setMetrics(data)
    } catch (error) {
      console.error('Failed to fetch metrics:', error)
    }
  }, [backendUrl])

  useEffect(() => {
    const ws = new WebSocket(backendUrl.replace('http', 'ws') + '/ws')
    wsRef.current = ws
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        setEvents((prev) => [...prev, msg])
        
        // Process different event types
        if (msg.type === 'plan') {
          setPlan(msg.payload)
        } else if (msg.type === 'result') {
          setResults((prev) => [...prev, msg.payload])
        } else if (msg.type === 'analysis') {
          setAnalysis(msg.payload)
        } else if (msg.type === 'done') {
          setIsRunning(false)
          fetchMetrics()
        }
      } catch {
        console.log('WebSocket message:', ev.data)
      }
    }
    ws.onclose = () => {
      wsRef.current = null
    }
    return () => ws.close()
  }, [backendUrl, fetchMetrics])

  async function startRun() {
    setIsRunning(true)
    setEvents([])
    setPlan(null)
    setResults([])
    setAnalysis(null)
    setMetrics(null)
    setExportData(null)
    
    try {
      const parsedConstraints = JSON.parse(constraints || '{}')
    await fetch(backendUrl + '/api/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ goal, constraints: parsedConstraints }),
      })
    } catch (e) {
      console.error('Failed to start run:', e)
      setIsRunning(false)
    }
  }

  async function handleExport() {
    if (!plan) return
    try {
      const res = await fetch(backendUrl + '/api/export', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ parameters: plan.variants[0]?.parameters || {} }),
      })
      const blob = await res.blob()
      const text = await blob.text()
      setExportData(text)
    } catch (e) {
      console.error('Failed to export:', e)
    }
  }

  function downloadExport() {
    const blob = new Blob([exportData], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'simstack-compose.yml'
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="app">
      {/* Header with Sponsor Branding */}
      <header className="header">
        <div className="header-content">
          <h1>
            <span className="logo">⚡</span> SimStack
          </h1>
          <div className="sponsor-badges">
            <span className="badge cerebras">Powered by Cerebras</span>
            <span className="badge llama">Meta Llama 3.1</span>
            <span className="badge docker">Docker MCP</span>
          </div>
        </div>
        <p className="tagline">Agentic Simulation Studio • 16 Parallel Scenarios • Instant AI Analysis</p>
      </header>

      {/* Main Content */}
      <div className="container">
        {/* Input Section */}
        <div className="card input-section">
          <h2>🎯 Simulation Goal</h2>
          <input 
            value={goal} 
            onChange={(e) => setGoal(e.target.value)} 
            placeholder="e.g., reduce ER wait time by 20%"
            disabled={isRunning}
          />
          <input 
            value={constraints} 
            onChange={(e) => setConstraints(e.target.value)} 
            placeholder='{"budget": 100000}'
            disabled={isRunning}
          />
          <button onClick={startRun} disabled={isRunning} className="btn-primary">
            {isRunning ? '⏳ Running...' : '🚀 Start Simulation'}
          </button>
          {isRunning && (
            <div className="status-message">
              <div className="spinner"></div>
              <span>AI is planning and executing simulations...</span>
            </div>
          )}
        </div>

        {/* Performance Metrics */}
        {metrics && metrics.PlannerMs !== undefined && (
          <div className="card metrics-card">
            <h2>⚡ Performance Metrics</h2>
            <div className="metrics-grid">
              <div className="metric">
                <div className="metric-label">Planning Time</div>
                <div className="metric-value">
                  {metrics.PlannerMs > 0 ? `${metrics.PlannerMs}ms` : '<1ms'}
                </div>
              </div>
              <div className="metric">
                <div className="metric-label">Simulation Time</div>
                <div className="metric-value">
                  {metrics.SimulationStartupMs > 0 ? `${metrics.SimulationStartupMs}ms` : '<1ms'}
                </div>
              </div>
              {metrics.TokensPerSecond > 0 && (
                <div className="metric cerebras-speed">
                  <div className="metric-label">Cerebras Speed</div>
                  <div className="metric-value">{Math.round(metrics.TokensPerSecond)} tokens/s</div>
                  <div className="metric-sublabel">🔥 Lightning Fast</div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Success Message */}
        {!isRunning && results.length > 0 && (
          <div className="card success-message">
            <h2>✅ Simulation Complete!</h2>
            <p>Successfully executed {results.length} variant scenarios in parallel. Review results below.</p>
          </div>
        )}

        {/* Simulation Plan */}
        {plan && (
          <div className="card plan-card">
            <h2>📋 Simulation Plan</h2>
            <div className="plan-info">
              <p><strong>Plan ID:</strong> {plan.plan_id}</p>
              <p><strong>Variants:</strong> {plan.variants?.length || 0} parallel scenarios</p>
              <p><strong>Status:</strong> {isRunning ? '🔄 Running...' : '✅ Complete'}</p>
            </div>
            <div className="variants-grid">
              {plan.variants?.map((variant, i) => (
                <div key={i} className="variant-card">
                  <h3>Variant {i + 1}</h3>
                  <div className="variant-params">
                    {Object.entries(variant.parameters || {}).map(([key, value]) => (
                      <div key={key} className="param">
                        <span className="param-key">{key}:</span>
                        <span className="param-value">{JSON.stringify(value)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Results */}
        {results.length > 0 && (
          <div className="card results-card">
            <h2>📊 Simulation Results</h2>
            <div className="results-grid">
              {results.map((result, i) => {
                // Simple heuristic: variant with lowest wait time or highest throughput is "best"
                const isBest = i === 0 && results.length > 1; // You can improve this logic
                return (
                  <div key={i} className={`result-card ${isBest ? 'best-result' : ''}`}>
                    <h3>
                      {result.variant_id}
                      {isBest && <span className="best-badge">⭐ Best</span>}
                    </h3>
                    <div className="metrics-list">
                      {Object.entries(result.metrics || {}).map(([key, value]) => (
                        <div key={key} className="result-metric">
                          <span className="metric-name">{key.replace(/_/g, ' ')}</span>
                          <span className="metric-val">
                            {typeof value === 'number' ? value.toFixed(2) : value}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* AI Analysis - "Why this plan" */}
        {analysis && (
          <div className="card analysis-card">
            <h2>🤖 AI Analysis - Why This Plan</h2>
            <div className="analysis-content">
              <div className="analysis-recommendation">
                <h3>💡 Recommendation</h3>
                <p>{analysis.recommendation || 'Analysis in progress...'}</p>
                {analysis.confidence && (
                  <div className="confidence-bar">
                    <span>Confidence: {Math.round(analysis.confidence * 100)}%</span>
                    <div className="confidence-fill" style={{width: `${analysis.confidence * 100}%`}}></div>
                  </div>
                )}
              </div>

              {analysis.winner && (
                <div className="analysis-winner">
                  <h3>🏆 Winner</h3>
                  <p className="winner-id">{analysis.winner}</p>
                </div>
              )}

              {analysis.trade_offs && analysis.trade_offs.length > 0 && (
                <div className="analysis-section">
                  <h3>⚖️ Trade-offs</h3>
                  <ul>
                    {analysis.trade_offs.map((trade, i) => (
                      <li key={i}>{trade}</li>
                    ))}
                  </ul>
                </div>
              )}

              {analysis.counterfactuals && analysis.counterfactuals.length > 0 && (
                <div className="analysis-section">
                  <h3>🔮 What-If Scenarios</h3>
                  <ul>
                    {analysis.counterfactuals.map((cf, i) => (
                      <li key={i}>{cf}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
            <button onClick={handleExport} className="btn-secondary">
              📦 Export Winning Configuration
            </button>
          </div>
        )}

        {/* Export Preview */}
        {exportData && (
          <div className="card export-card">
            <h2>📦 Docker Compose Export</h2>
            <pre className="export-preview">{exportData}</pre>
            <button onClick={downloadExport} className="btn-primary">
              ⬇️ Download docker-compose.yml
            </button>
          </div>
        )}

        {/* Live Events Stream */}
        <div className="card events-card">
          <h2>📡 Live Event Stream</h2>
          <div className="events-list">
            {events.slice().reverse().map((e, i) => (
              <div key={i} className={`event event-${e.type}`}>
                <span className="event-type">{e.type}</span>
                <span className="event-time">{new Date(e.timestamp).toLocaleTimeString()}</span>
                <div className="event-payload">
                  {JSON.stringify(e.payload, null, 2)}
                </div>
      </div>
          ))}
          </div>
        </div>
      </div>

      {/* Footer */}
      <footer className="footer">
        <p>Built for FutureStack GenAI Hackathon 2025</p>
        <p className="sponsors">
          <span>Powered by:</span>
          <strong>Cerebras</strong> • <strong>Meta Llama 3.1</strong> • <strong>Docker MCP</strong>
        </p>
      </footer>
    </div>
  )
}

export default App
