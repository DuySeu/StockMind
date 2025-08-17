# Stock market retrieve information application

This project use LLM model to retrieve all releated information about specific stock in VN.

This application is fully using open source framework anh service

## Use case

### 1. Retrieve Financial Statements

- Users can request the financial statement of any listed company (by company name or stock ticker).
- The assistant will provide the financial statement in a previewable format within the application.
- Users can also download the financial statement as a file for offline use.

### 2. Access Fundamental Analysis Indicators

- Users can query key indicators for fundamental analysis of any stock, including but not limited to:
  - P/E, P/S, PEG ratios  
  - ROA, ROE  
  - EPS, Book Value per Share  
  - Dividend Yield, Debt-to-Equity Ratio, and other standard financial metrics.
- The assistant aggregates these indicators and presents them in a structured and easy-to-read format.

### 3. Evaluate Value Stocks (Piotroski Score)

- Users can evaluate whether a stock qualifies as a "value stock" using the **Piotroski F-Score** framework. Ref: [https://www.investopedia.com/terms/p/piotroski-score.asp]
- The assistant provides:
  - A detailed explanation of each of the 9 criteria used in the score.  
  - A breakdown of how the company performs against each criterion.  
  - The final Piotroski Score with an interpretation of its implication for investment decisions.

### 4. Track Financial Events Affecting Stocks

- Users can query whether any financial events (e.g., earnings announcements, dividend declarations, mergers, acquisitions, or regulatory updates) are associated with a specific stock.  
- The assistant retrieves and displays:
  - The **content** or description of the financial event.  
  - The **date** the event occurred or was announced.  
- This feature enables investors to stay informed about significant market-moving events and better understand the context behind stock performance.
