# INTEGRATIONS

## AI & LLM Providers
- **Anthropic via Go SDK**: `github.com/anthropics/anthropic-sdk-go`
- **OpenAI via Go SDK**: `github.com/sashabaranov/go-openai`
- **OpenRouter API**: Configured via environment variable `OPENROUTER_API_KEY`
- **Tavily API**: Used for search/research, configured via `TAVILY_API_KEY`

## Cloud block & Services
- **AWS**: AWS SDK for Go (`github.com/aws/aws-sdk-go-v2`), specifically STS for assumed roles/credentials. 
- **Model Context Protocol (MCP)**: Implemented via `github.com/mark3labs/mcp-go` to connect AI execution logic to local/remote tools.

## Internal APIs
- **Goose/PostgreSQL**: Backend database is managed locally via `docker-compose.yml` defining the `stockmind_db` container.
