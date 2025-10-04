from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Traffic Simulator")


class TrafficInput(BaseModel):
    density: float
    signal_timing: float | None = None


@app.post("/simulate")
def simulate(inp: TrafficInput):
    speed = max(5.0, 60.0 * (1.0 - min(1.0, max(0.0, inp.density))))
    throughput = speed * 10.0
    return {
        "metrics": {
            "avg_speed_kmh": speed,
            "throughput_veh_per_hr": throughput,
        }
    }

