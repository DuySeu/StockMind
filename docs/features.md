# StockMind Features Documentation

This document provides a comprehensive overview of the features available in StockMind. It differentiates between fully implemented features and planned capabilities that are currently under development or slated for future expansion.

## 1. AI Assistant (Chatbot)
The core of StockMind is an intelligent conversational agent powered by Large Language Models (LLMs) and integrated with the Vietnamese stock market data provider (**vnstock3**).

### 🟢 Implemented Capabilities
*   **Conversational Interface**: Users can chat with the assistant via a WebSocket/SSE connection to query financial data in plain text.
*   **Contextual Memory**: The assistant maintains session memory to handle follow-up questions intelligently.
*   **Financial Statement Retrieval**: Users can request the financial statement of any listed company, which the assistant retrieves and previews in the chat.
*   **Fundamental Indicators**: The assistant can calculate and present key financial metrics (P/E, PEG, ROA, ROE, EPS, etc.) directly in the conversation.

### 🟡 Planned Capabilities (Undone)
*   **Piotroski F-Score Evaluation**: 
    *   *Goal*: Automatically calculate the 9-point Piotroski F-Score for a given stock.
    *   *Output*: Provide a breakdown of all 9 criteria (Profitability, Leverage, Operating Efficiency) and the final score, judging whether the stock is a strong "value" stock.
*   **Altman Z-Score Evaluation**:
    *   *Goal*: Automatically calculate the Altman Z-Score to predict the probability of a company going bankrupt within two years.
    *   *Output*: Break down the 5 components of the formula and provide an interpretation of the final score's implications.

---

## 2. Market Dashboard
The dashboard acts as the primary visual hub for investors to track their portfolio and the broader market.

### 🟢 Implemented Capabilities
*   **Price Board (`/stock/price-board`)**: Real-time or delayed stock quotes for various tickers.
*   **Watchlist (`/stock/watchlist`)**: Users can add specific symbols to their personal watchlist to track performance.
*   **Financial News & Events**: 
    *   Retrieves news articles and major announcements regarding the market or specific stocks.
    *   Displays the publication date and event context.

### 🟡 Planned Capabilities (Undone)
*   **Advanced Event Tracking**: Improved tracking of dividends, mergers, acquisitions, and regulatory updates automatically tied to the symbols in the user's watchlist.
*   **Alerting System**: Push notifications or email alerts for significant price changes or breaking news.

---

## 3. Market Researcher System
A specialized flow designed to digest large amounts of unstructured data and output coherent financial research.

### 🟢 Implemented Capabilities
*   **Report Generation (`/stock/research`)**: Users can trigger a deep-dive research task on a specific ticker.
*   **Report Viewing (`/stock/research-reports`)**: Users can view past reports detailing fundamental analyses and recent events.

### 🟡 Planned Capabilities (Undone)
*   **Sentiment Analysis Integration**:
    *   *Goal*: Automatically scan news, analyst reports, and social chatter to gauge the current "mood" around a stock.
    *   *Output*: Append a sentiment score (Bullish, Bearish, Neutral) to the Market Researcher report.
