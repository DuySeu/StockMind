import os
import asyncpg

from logsim import CustomLogger
from fastapi import Request
from typing import AsyncGenerator, Any

from core.s3_client import get_s3_client
from core.database import get_db_connection


log = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


def get_request_id(request: Request) -> str:
    """
    Get request ID from request state.
    """
    return getattr(request.state, "request_id", "unknown")


async def get_db() -> AsyncGenerator[asyncpg.Connection, None]:
    """
    Dependency to get database connection pool.
    """
    async with get_db_connection() as connection:
        yield connection


def get_s3() -> Any:
    """
    Dependency to get S3 bucket client
    """
    return get_s3_client()
