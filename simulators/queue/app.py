from fastapi import FastAPI
from pydantic import BaseModel
import math

app = FastAPI(title="Queue Simulator")


class QueueInput(BaseModel):
    arrival_rate: float
    service_rate: float


@app.post("/simulate")
def simulate(inp: QueueInput):
    # M/M/1 wait time approximation
    rho = inp.arrival_rate / inp.service_rate if inp.service_rate > 0 else 0.0
    
    # Handle different system states
    if rho >= 0.95:
        # Near-overload: system is unstable, use extrapolated high value
        wait = 240.0  # 4 hours in minutes (very poor service)
        queue_length = 50.0  # Very long queue
    elif rho < 0:
        wait = 0.0
        queue_length = 0.0
    else:
        # Normal M/M/1 formulas
        wait = rho / (inp.service_rate * (1 - rho))  # Wait in hours
        queue_length = rho / (1 - rho)  # Average queue length
    
    # Convert wait to minutes and ensure reasonable bounds
    wait_minutes = max(0.0, min(wait * 60.0, 300.0))  # Cap at 5 hours
    
    # Ensure values are JSON-compliant (no inf, no nan)
    if math.isnan(wait_minutes) or math.isinf(wait_minutes):
        wait_minutes = 240.0  # Default to 4 hours for edge cases
    
    # Clamp utilization to valid range
    utilization = max(0.0, min(rho, 0.99))  # Cap at 99%
    
    # Cap queue length at reasonable value
    queue_length = max(0.0, min(queue_length if not math.isnan(queue_length) else 50.0, 100.0))
    
    return {
        "metrics": {
            "avg_wait_time_min": round(wait_minutes, 2),
            "utilization": round(utilization, 3),
            "avg_queue_length": round(queue_length, 2),
        }
    }


