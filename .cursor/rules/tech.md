# Technology Stack

## Backend
- Go 1.25+ with chi router
- PostgreSQL 17 (pgx/v5 driver, sqlc for query generation)
- Qdrant for vector storage (dense + sparse vectors)
- MinIO for object/file storage
- MCP (Model Context Protocol) via mcp-go for tool serving

## Frontend
- React 19, Vite 7, TypeScript
- Tailwind CSS 4 + shadcn/ui (Radix primitives)
- Axios for REST, native fetch + ReadableStream for SSE
- react-markdown + remark-gfm for chat rendering
- jsPDF for report export

## LLM Providers
- OpenRouter (primary, official go-sdk)
- OpenAI (sashabaranov/go-openai)
- Anthropic (official SDK, supports AWS Bedrock)

## External Services
- OpenRouter — LLM completions + embeddings
- Tavily — web search for market research
- VietCap — Vietnamese stock market data API

## Key Libraries
- urfave/cli/v3 for CLI commands
- goose for SQL migrations
- sqlc for type-safe query generation
- mcp-go for MCP protocol
- minio-go for object storage
- go-client (Qdrant) for vector DB
