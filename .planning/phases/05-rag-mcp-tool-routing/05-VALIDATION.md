# Phase 5 Validation: RAG Tool Verification

## Acceptance Criteria

### Automated Tests
- [ ] `internal/rag` unit test: `TestSearch` verifies gRPC call results are filtered correctly.
- [ ] `internal/mcp` unit test: `TestRetrieveKnowledge` verifies the handler returns correct formatting.
- [ ] Model dimensions check: Query vector length must be 2048.

### Manual Verification
- [ ] Start backend and MCP server. 
- [ ] Use LLM session to ask: "Tỷ lệ P/E là gì?"
    - **Expectation:** LLM calls `retrieve_knowledge`.
- [ ] Ask: "Giá HPG hôm nay bao nhiêu?"
    - **Expectation:** LLM calls `get_stock_price` (or news), NOT `retrieve_knowledge`.
- [ ] Check console logs for "MCP Tool Called: retrieve_knowledge".
- [ ] Check Qdrant query score for threshold adherence. (Log search scores during development).

## Manual Test Cases

| Input | Expected Tool | Reason |
|-------|---------------|--------|
| "Công thức tính Altman Z-Score?" | `piotroski_evaluation` / `altman_z_score` | Specific tools exist. |
| "Phân tích rủi ro của thị trường chứng khoán VN theo tài liệu mới nhất?" | `retrieve_knowledge` | Broad concept covered in docs. |
| "HPG là công ty gì?" | `retrieve_knowledge` | General info from knowledge base. |
| "Mua HPG lúc này được không?" | `retrieve_knowledge` (for context) + `get_stock_price` | Mixed intent handled by the agent orchestrator. |
