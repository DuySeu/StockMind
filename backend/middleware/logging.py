import os

from fastapi import Request
from logsim import CustomLogger
from starlette.middleware.base import BaseHTTPMiddleware

# Configure logging
log = CustomLogger(use_json=os.getenv("LOG_FORMAT_JSON") == "true")


class LoggingMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        # Generate request_id
        request_id = os.urandom(9).hex()
        request.state.request_id = request_id

        # Log the incoming request
        log.debug(f"{request.method}: {request.url.path} received")

        # Call the actual endpoint (and get the response)
        res = await call_next(request)

        # After endpoint is executed, log the response status
        log.info(
            msg=f"{request.method}: {request.url.path} - {res.status_code}",
            extra={
                "request_id": request_id,
                "method": request.method,
                "url": request.url.path,
                "status_code": res.status_code,
            },
        )

        return res
