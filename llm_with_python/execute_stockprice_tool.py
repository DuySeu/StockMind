from vnstock import Vnstock
from datetime import date


def get_stock_price(ticker: str):
    """Use this tool to retrieve the stock price data for a specific company. This tool will get the price as a pandas dataframe, with open, high, low, close, volume and ticker columns.
    The tool has been set to take the price until today and 1 month behind as default. If the user do not specify the date, always get the price for the latest date
    """
    # Initiate the stock object
    stock = Vnstock().stock(symbol=ticker, source="VCI")
    today = date.today().strftime("%Y-%m-%d")

    # Query the latest stock until today
    price_df = stock.quote.history(start=today, end=today)

    return price_df
