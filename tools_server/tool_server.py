from pydantic import BaseModel
from logsim import CustomLogger
from fastapi import APIRouter
from fastapi.responses import JSONResponse

from core.stock_tool import stock_price, stock_event

# Setup logger
logger = CustomLogger()

# Setup router
router = APIRouter()


# Model for Stock Ticker
class StockTicker(BaseModel):
    ticker: str


@router.get("/stock_price/{ticker}")
async def get_stock_price(ticker: str):
    """Get the stock price for a specific ticker"""

    try:
        price = stock_price(ticker)

        return JSONResponse(content=price, status_code=200)
    except Exception as e:
        return JSONResponse(
            content={"message": f"Error in getting all documents: {str(e)}"},
            status_code=500,
        )

@router.get("/stock_event/{ticker}")
async def get_stock_event(ticker: str, limit: int = 5):
    """Get the stock event for a specific ticker"""

    try:
        event = stock_event(ticker, limit)

        return JSONResponse(content=event, status_code=200)
    except Exception as e:
        return JSONResponse(
            content={"message": f"Error in getting all documents: {str(e)}"},
            status_code=500,
        )