from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Resource Allocation Simulator")


class ResourceInput(BaseModel):
    staff: int
    shifts: list[str] | None = None


@app.post("/simulate")
def simulate(inp: ResourceInput):
    coverage = inp.staff * 0.8
    satisfaction = min(1.0, 0.5 + (inp.staff / 100.0))
    return {
        "metrics": {
            "coverage_units": coverage,
            "satisfaction": satisfaction,
        }
    }


