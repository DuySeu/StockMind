from vnstock import Listing, Quote, Company, Finance
from datetime import date

def stock_price(ticker: str):
    """Use this tool to retrieve the stock price data for a specific company. This tool will get the price as a pandas dataframe, with open, high, low, close, volume and ticker columns.
    The tool has been set to take the price until today and 1 month behind as default. If the user do not specify the date, always get the price for the latest date
    """
    # Initiate the stock object
    quote = Quote(symbol=ticker, source="VCI")
    today = date.today().strftime("%Y-%m-%d")

    # Query the latest stock until today
    price_df = quote.history(start=today, end=today)

    return price_df


def stock_event(ticker: str, limit: int = 5):
    """Use this tool to retrieve the stock info for a specific company. This tool will get the stock info as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    company = Company(symbol=ticker, source="VCI")
    event = company.events().head(limit)
    return event


def stock_news(ticker: str, limit: int = 5):
    """Use this tool to retrieve the stock info for a specific company. This tool will get the stock info as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    company = Company(symbol=ticker, source="VCI")
    news = company.news().head(limit)
    return news

def stock_overview(ticker: str):
    """Use this tool to retrieve the stock overview for a specific company. This tool will get the stock overview as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    company = Company(symbol=ticker, source="VCI")
    overview = company.overview()
    return overview

def stock_ratio(ticker: str):
    """Use this tool to retrieve the stock ratio for a specific company. This tool will get the stock ratio as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    finance = Finance(symbol=ticker, source='VCI')
    ratio = finance.ratio(period='quarter', lang='en', dropna=True).head()
    return ratio

def stock_balance_sheet(ticker: str):
    """Use this tool to retrieve the stock balance sheet for a specific company. This tool will get the stock balance sheet as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    finance = Finance(symbol=ticker, source='VCI')
    balance_sheet = finance.balance_sheet(period='year', lang='vi', dropna=True).head()
    return balance_sheet

def stock_income_statement(ticker: str):
    """Use this tool to retrieve the stock income statement for a specific company. This tool will get the stock income statement as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    finance = Finance(symbol=ticker, source='VCI')
    income_statement = finance.income_statement(period='year', lang='vi', dropna=True).head()
    return income_statement

def stock_cash_flow(ticker: str):
    """Use this tool to retrieve the stock cash flow for a specific company. This tool will get the stock cash flow as a pandas dataframe, with ticker, name, industry, sector, marketcap, price, change, change_percent, volume, open, high, low, close, volume and ticker columns.
    """
    finance = Finance(symbol=ticker, source='VCI')
    cash_flow = finance.cash_flow(period='year', lang='vi', dropna=True).head()
    return cash_flow

def stock_list_by_group(group: str):
    """Use this tool to retrieve the stock list by group.
        Example: HOSE, VN30, VNMidCap, VNSmallCap, VNAllShare, VN100, ETF, HNX, HNX30, HNXCon, HNXFin, HNXLCap, HNXMSCap, HNXMan, UPCOM, FU_INDEX (mã chỉ số hợp đồng tương lai), CW (chứng quyền)
    """
    listing = Listing()
    stock_list = listing.symbols_by_group(group)
    return stock_list

def main():
    # listing = Listing()
    # listing.all_symbols()
    # print(listing.all_symbols())
    # print(stock_ratio("VNM"))
    print(stock_list_by_group("VN30"))

if __name__ == "__main__":
    main()
