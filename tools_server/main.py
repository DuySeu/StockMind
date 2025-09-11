import os
import uvicorn

from logsim import CustomLogger
from dotenv import load_dotenv
from fastapi import FastAPI
from starlette.middleware.cors import CORSMiddleware

from routers import tool_server

# Load environment variables from .env file
load_dotenv()

# Config environment

# Configure logging
log = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


# Initialize FastAPI app
app = FastAPI(
    title="StockMind tool server",
    description="Tool server for the function calling",
    version="1.0.0",
    debug=True,
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add router
app.include_router(prefix="/tool_server", router=tool_server.router)


@app.get("/health")
async def health():
    """
    Health check endpoint.
    """
    return {"status": "ok"}


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)