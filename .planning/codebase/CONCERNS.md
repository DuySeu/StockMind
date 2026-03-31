# CONCERNS

## Technical Debt & Areas for Improvement
- **Missing Frontend Testing Suite**: No unit tests available for React Hook implementations and API handlers within `frontend/src`.
- **API Key Leakage Warnings**: Multiple environment configurations (`OPENROUTER_API_KEY`, `TAVILY_API_KEY`) rely solely on environment strings. Needs tight `.gitignore` and `.env` isolation checks.
- **LLM Rate Limits / Scaling**: Backend interactions directly depend on the throughput of Anthropic & go-openai limits without robust queuing/backoff architectures explicitly mapped or discussed out-of-the-box besides native SDK retries.
- **Strict Integration Dependencies**: High reliance on external data mapping (financial stats APIs, maybe VN stock libraries as per README notes) which may silently drift formats without robust intermediate schema validation prior to SQLC ingestion.
