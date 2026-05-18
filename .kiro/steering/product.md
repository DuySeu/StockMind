# StockMind Product Overview

AI-powered financial assistant for the Vietnamese stock market.

## Target Users
Vietnamese retail investors who want AI-assisted fundamental analysis and market research.

## Core Features
- Conversational AI chat with agentic tool-calling for real-time financial data
- Piotroski F-Score and Altman Z-Score evaluations via MCP tools
- Automated market research reports (Tavily web search + LLM digest)
- Document knowledge base with hybrid RAG retrieval (dense + sparse + RRF)
- Watchlist with VietCap price data
- PDF export for research reports

## Business Context
- All financial data comes from VietCap Trading API
- LLM access via OpenRouter (supports OpenAI and Anthropic models)
- No authentication currently — single hardcoded user
- Agent flow configs exist in DB but chat uses single provider/model from env vars
