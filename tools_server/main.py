import os
from contextlib import asynccontextmanager

from logsim import CustomLogger
from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from fastapi.staticfiles import StaticFiles
from starlette.middleware.cors import CORSMiddleware

from middleware.logging import LoggingMiddleware
from routers import documents, rules, templates, flow
from core.database import init_db_pool

# Load environment variables from .env file
load_dotenv()

# Config environment
POSTGRES_HOST = os.getenv("POSTGRES_HOST")
POSTGRES_PORT = os.getenv("POSTGRES_PORT")
POSTGRES_DB = os.getenv("POSTGRES_DB")
POSTGRES_USER = os.getenv("POSTGRES_USER")
POSTGRES_PASSWORD = os.getenv("POSTGRES_PASSWORD")

# Configure logging
log = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


@asynccontextmanager
async def lifespan(app: FastAPI):
    # connect to IDP database and run migrations
    try:
        await init_db_pool(
            host=POSTGRES_HOST,
            port=POSTGRES_PORT,
            username=POSTGRES_USER,
            password=POSTGRES_PASSWORD,
            database=POSTGRES_DB,
        )
        log.info("Database initialization and migrations completed successfully")
    except ConnectionRefusedError as conn_err:
        log.exception(f"Connection Error: {conn_err}")
        raise HTTPException(
            status_code=503,
            detail=f"Could not connect: {conn_err}",
        )
    except Exception as e:
        log.exception(f"Unexpected error: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Unexpected error: {e}",
        )
    yield


# Initialize FastAPI app
app = FastAPI(
    title="Intelligent Document Processing API",
    description="API for the Intelligent Document Processing Platform",
    version="1.0.0",
    debug=True,
    lifespan=lifespan,
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configure logging middleware
app.add_middleware(LoggingMiddleware)

# Add router
app.include_router(prefix="/v1", router=documents.router)
app.include_router(prefix="/v1", router=rules.router)
app.include_router(prefix="/v1", router=templates.router)
app.include_router(prefix="/v1", router=flow.router)


@app.get("/health")
async def health():
    """
    Health check endpoint.
    """
    return {"status": "ok"}


# Frontend endpoints
app.mount("/", StaticFiles(directory="frontend", html=True), name="static")
