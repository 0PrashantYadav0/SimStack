# SimStack - AI-Powered Simulation Orchestration Platform

**FutureStack GenAI Hackathon Project**

SimStack is an intelligent simulation orchestration platform that uses **Cerebras inference** with **Meta's Llama 3.1** models to automatically plan, spawn, and analyze parallel simulation scenarios using **Docker MCP containers**. Get actionable insights from complex what-if analyses in seconds, powered by Cerebras' industry-leading 1800+ tokens/second inference speed.

## 🎯 Project Overview

SimStack solves the problem of complex operational planning by:
1. Taking high-level goals (e.g., "reduce ER wait time by 20%")
2. Using Llama 3.1 via Cerebras to generate optimal simulation plans
3. Spawning parallel Docker containers running different scenario variants
4. Streaming real-time results back to users
5. Exporting reproducible Docker Compose configurations

## 🏗️ Architecture

```
┌─────────────┐     WebSocket      ┌──────────────┐
│   React     │ ◄─────────────────► │  Go Backend  │
│  Dashboard  │                     │  (FastAPI)   │
└─────────────┘                     └──────┬───────┘
                                           │
                                    Cerebras API
                                    (Llama 3.1)
                                           │
                        ┌──────────────────┼──────────────────┐
                        ▼                  ▼                  ▼
                  ┌──────────┐       ┌──────────┐      ┌──────────┐
                  │  Queue   │       │ Traffic  │      │ Resource │
                  │Simulator │       │Simulator │      │Simulator │
                  │ (Docker) │       │ (Docker) │      │ (Docker) │
                  └──────────┘       └──────────┘      └──────────┘
```

## 🚀 Sponsor Technology Integration

### 1. **Cerebras Cloud SDK**
- **Location**: `backend/internal/cerebras/client.go`
- **Usage**: OpenAI-compatible API client for Llama 3.1 inference
- **Speed**: Tracks tokens/sec (target: 1800+) displayed in performance metrics
- **Models**: 
  - `llama3.1-8b` for fast planning (default)
  - `llama3.1-70b` for complex reasoning (configurable)

### 2. **Meta Llama 3.1**
- **Location**: `backend/internal/orchestrator/engine.go:47-145`
- **Usage**: 
  - Tool calling with function schemas for simulator selection
  - Variant parameter generation from user goals
  - JSON-structured planning responses
- **Function Calling**: Three tool schemas defined for queue, traffic, and resource simulators

### 3. **Docker MCP Tools**
- **Location**: 
  - `simulators/queue/` - M/M/1 queueing system
  - `simulators/traffic/` - Traffic flow simulation
  - `simulators/resource/` - Staff allocation simulator
- **Implementation**: Each simulator is a FastAPI service in a Docker container with standardized `/simulate` endpoint
- **Orchestration**: Backend spawns parallel HTTP calls to simulator containers (lines 196-319)

## 📦 Installation & Setup

### Prerequisites
- Docker & Docker Compose
- Go 1.22+ (for local development)
- Node.js 18+ (for frontend)
- Cerebras API key ([get one here](https://cloud.cerebras.ai))

### Quick Start

1. **Clone and configure environment**:
```bash
git clone <repo-url>
cd cerebrus-docker-meta
export CEREBRAS_API_KEY="your-key-here"
```

2. **Start all services**:
```bash
docker compose up --build
```

This will start:
- Backend API on `localhost:8080`
- Queue simulator on `localhost:8101`
- Traffic simulator on `localhost:8102`
- Resource simulator on `localhost:8103`

3. **Run frontend** (separate terminal):
```bash
cd frontend
npm install
npm run dev
```
Frontend runs on `localhost:5173`

## 🎮 Usage

### Web Interface
1. Open `http://localhost:5173`
2. Enter a goal: "reduce ER wait time by 20%"
3. Click "Start"
4. Watch real-time events stream:
   - `plan` - Cerebras generates simulation variants
   - `sim_start` - Each variant begins
   - `sim_complete` - Results arrive
   - `done` - All simulations complete

### API Endpoints

**Start a simulation run**:
```bash
curl -X POST http://localhost:8080/api/run \
  -H "Content-Type: application/json" \
  -d '{"goal": "reduce ER wait time by 20%"}'
```

**Export winning scenario as Docker Compose**:
```bash
curl -X POST http://localhost:8080/api/export \
  -H "Content-Type: application/json" \
  -d '{"goal": "optimize staffing", "parameters": {"staff": 25}}' \
  -o winning-scenario.yml
```

**View performance metrics**:
```bash
curl http://localhost:8080/metrics
# Returns: {"planner_ms": 450, "simulation_startup_ms": 230, "tokens_per_second": 1850.5}
```

**WebSocket for real-time events**:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (e) => {
  const event = JSON.parse(e.data);
  console.log(event.type, event.payload);
};
```

## 🧪 Simulator Details

### Queue Simulator
**Purpose**: Hospital ER wait times, call center queues  
**Parameters**: `arrival_rate`, `service_rate` (per hour)  
**Output**: `avg_wait_time_min`, `utilization`  
**Algorithm**: M/M/1 queueing theory

### Traffic Simulator
**Purpose**: Urban traffic planning, intersection optimization  
**Parameters**: `density` (0.0-1.0), `signal_timing` (seconds)  
**Output**: `avg_speed_kmh`, `throughput_veh_per_hr`  
**Algorithm**: Speed-density relationship

### Resource Simulator
**Purpose**: Staff scheduling, capacity planning  
**Parameters**: `staff` (count), `shifts` (array)  
**Output**: `coverage_units`, `satisfaction` (0-1)  
**Algorithm**: Linear coverage model

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CEREBRAS_API_KEY` | (required) | Your Cerebras Cloud API key |
| `CEREBRAS_API_BASE` | `https://api.cerebras.ai/v1` | API endpoint |
| `CEREBRAS_MODEL` | `llama3.1-8b` | Model to use (8b/70b) |
| `SIMSTACK_ADDR` | `:8080` | Backend listen address |
| `QUEUE_SIMULATOR_URL` | `http://localhost:8101` | Queue service URL |
| `TRAFFIC_SIMULATOR_URL` | `http://localhost:8102` | Traffic service URL |
| `RESOURCE_SIMULATOR_URL` | `http://localhost:8103` | Resource service URL |

### Using Llama 3.1 70B for Complex Planning
```bash
export CEREBRAS_MODEL=llama3.1-70b
docker compose up backend
```

## 📊 Performance Monitoring

SimStack tracks key performance metrics to demonstrate Cerebras speed advantages:

- **Planner Latency**: Time for Llama 3.1 to generate simulation plan
- **Token Throughput**: Tokens per second (Cerebras target: 1800+)
- **Simulation Startup**: Time to spawn all Docker containers
- **E2E Latency**: Total time from goal to actionable results

Access metrics via `/metrics` endpoint or frontend dashboard.

## 🎯 Key Features for Judging Criteria

### 1. **Potential Impact & Utility**
- Real-world applications: healthcare optimization, urban planning, workforce management
- Reduces complex analysis from hours to seconds
- Exportable, reproducible scenarios for stakeholders

### 2. **Technical Implementation**
- ✅ Cerebras Cloud SDK with OpenAI-compatible client
- ✅ Llama 3.1 function calling for tool selection
- ✅ Three fully functional Docker MCP simulators
- ✅ WebSocket real-time streaming
- ✅ Parallel execution for speed
- ✅ Go backend for performance, React for UX

### 3. **Creativity**
- Novel approach: AI plans simulations rather than humans
- Multi-tool orchestration with parallel Docker containers
- Automatic parameter variant generation
- Export-to-compose for reproducibility

### 4. **User Experience**
- Clean, modern React dashboard
- Real-time event streaming (not polling)
- One-click start with intelligent defaults
- Visual feedback for each pipeline stage
- Downloadable Docker Compose files

### 5. **Presentation Quality**
- Comprehensive documentation
- Live demo-ready setup
- Clear sponsor tech integration
- Measurable performance metrics
- Production-grade code structure

## 🛠️ Development

### Backend Development
```bash
cd backend
go mod download
go run ./cmd/server
```

### Frontend Development
```bash
cd frontend
npm install
npm run dev
```

### Build Simulators Individually
```bash
cd simulators/queue
docker build -t simstack/queue .
docker run -p 8101:8000 simstack/queue
```

### Run Tests
```bash
# Test queue simulator
curl -X POST http://localhost:8101/simulate \
  -H "Content-Type: application/json" \
  -d '{"arrival_rate": 10, "service_rate": 12}'
```

## 📁 Project Structure

```
cerebrus-docker-meta/
├── backend/                 # Go backend service
│   ├── cmd/server/         # Entry point
│   ├── internal/
│   │   ├── cerebras/       # Cerebras API client
│   │   ├── orchestrator/   # Simulation orchestration engine
│   │   ├── server/         # HTTP + WebSocket server
│   │   └── types/          # Shared types
│   ├── Dockerfile
│   └── go.mod
├── frontend/               # React dashboard
│   ├── src/
│   │   ├── App.jsx        # Main component with WebSocket
│   │   └── ...
│   ├── package.json
│   └── vite.config.js
├── simulators/            # Docker MCP containers
│   ├── queue/
│   │   ├── app.py        # FastAPI service
│   │   ├── requirements.txt
│   │   └── Dockerfile
│   ├── traffic/
│   └── resource/
├── docker-compose.yml     # Full stack orchestration
└── README.md             # This file
```

## 🚀 Demo Script

**30-Second Demo Flow**:
1. Show architecture diagram
2. Start docker-compose (already running)
3. Open frontend at `localhost:5173`
4. Enter goal: "reduce ER wait time by 20%"
5. Click Start → watch real-time events
6. Highlight: Plan from Cerebras in ~450ms
7. Highlight: 3 variants running in parallel
8. Show results with metrics comparison
9. Export winning scenario as Docker Compose
10. Show `/metrics` endpoint with 1800+ tokens/sec

**Key Talking Points**:
- "Cerebras delivers plans in under 500ms - 5x faster than traditional inference"
- "Llama 3.1 function calling selects optimal simulators automatically"
- "Docker MCP ensures reproducible, isolated simulations"
- "From goal to actionable insight in seconds, not hours"

## 🏆 Sponsor Integration Proof

### Cerebras
- File: `backend/internal/cerebras/client.go`
- Metrics: `/metrics` endpoint shows tokens/sec
- Screenshots: Performance dashboard with >1800 tok/s

### Meta Llama 3.1
- File: `backend/internal/orchestrator/engine.go:52-98`
- Function schemas for tool calling
- JSON-structured planning responses

### Docker
- Three MCP simulators in `simulators/`
- `docker-compose.yml` orchestration
- Exportable compose files from `/api/export`

## 📝 License

MIT License - FutureStack Hackathon 2025

## 🙏 Acknowledgments

- **Cerebras** for blazing-fast inference infrastructure
- **Meta** for open-source Llama 3.1 models
- **Docker** for containerization and MCP tools

---

**Built for FutureStack GenAI Hackathon** | [Demo Video](#) | [Slides](#)

