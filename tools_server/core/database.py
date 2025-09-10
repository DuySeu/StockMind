import os
import time

import asyncpg
from datetime import datetime

from contextlib import asynccontextmanager
from logsim import CustomLogger
from typing import Dict, List, Optional
from migrations.migrations import run_database_migrations

# Database pool instance
db_pool: Optional[asyncpg.Pool] = None
log = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


async def init_db_pool(
    host: str,
    port: int,
    username: str,
    password: str,
    database: str,
):
    """
    Initialize the database connection pool.
    """
    global db_pool
    db_pool = await asyncpg.create_pool(
        host=host,
        port=port,
        user=username,
        password=password,
        database=database,
        min_size=1,
        max_size=10000,
    )
    log.info(f"Database pool to {host}:{port} initialized")

    # Run database migrations
    try:
        async with db_pool.acquire() as connection:
            await run_database_migrations(connection)
    except Exception as e:
        log.error(f"Failed to run database migrations: {str(e)}")
        raise


async def close_db_pool():
    """
    Close the database connection pool.
    """
    global db_pool
    if db_pool:
        await db_pool.close()
        log.info("Database pool closed")
        db_pool = None


@asynccontextmanager
async def get_db_connection():
    """
    Get a database connection from the pool.
    """
    if not db_pool:
        raise Exception("Database pool not initialized")

    async with db_pool.acquire() as connection:
        yield connection


async def execute_query(
    query: str,
    params: tuple,
    request_id: str = None,
    connection: asyncpg.Connection = None,
) -> List[Dict]:
    """
    Execute a database query with proper error handling.
    """
    # Use provided connection or get one from pool
    if connection:
        return await _execute_with_connection(
            connection=connection,
            query=query,
            params=params,
            request_id=request_id,
        )
    else:
        async with get_db_connection() as conn:
            return await _execute_with_connection(
                connection=conn,
                query=query,
                params=params,
                request_id=request_id,
            )


async def _execute_with_connection(
    connection: asyncpg.Connection,
    query: str,
    params: tuple,
    request_id: str = None,
) -> List[Dict]:
    """Execute query with a specific connection."""
    log.debug(
        msg=f"Executing query: {query[:100]}...",
        extra={"request_id": request_id} if request_id else {},
    )

    start = time.time()
    result = await connection.fetch(query, *params)

    # Convert rows to dictionaries and handle datetime objects
    processed_result = []
    for row in result:
        row_dict = dict(row)
        # Convert datetime objects to ISO format strings
        for key, value in row_dict.items():
            if isinstance(value, datetime):
                row_dict[key] = value.isoformat()
        processed_result.append(row_dict)

    elapsed = time.time() - start

    log.debug(
        msg=f"Query returned {len(processed_result)} rows in {elapsed:.2f}s",
        extra={"request_id": request_id} if request_id else {},
    )

    return processed_result
