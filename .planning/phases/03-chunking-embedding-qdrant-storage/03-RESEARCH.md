# Phase 3 Research: Chunking, Embedding & Qdrant Storage

## 1. Chunking Strategies & Implementation

### Recursive Character Splitting (Default)
Standard implementation using `github.com/tmc/langchaingo/textsplitter`.
- **Logic**: Attempts to split on a hierarchy of separators (`\n\n`, `\n`, `" "`, `""`) until the chunk size is met.
- **Library**: `langchaingo/textsplitter` provides a robust `RecursiveCharacterTextSplitter`.
- **Config**: 512 tokens with 10% overlap (approx 51 tokens). Note: LangChain splitters typically count characters, so a character count approximation for tokens (~4 chars/token) or a custom token counter integration is needed.

### Paragraph-based Splitting
Simple manual implementation or using `langchaingo/textsplitter` with `\n\n` as the only separator.
- **Logic**: Splits on structural boundaries to preserve semantic units.

### Semantic Chunking
Splits based on meaning shifts rather than character counts.
- **Approach**:
    1. Break text into sentences.
    2. Generate embeddings for each sentence using the same OpenRouter embedder.
    3. Calculate cosine similarity between adjacent sentences.
    4. Split where similarity drops below a certain threshold.
- **Implementation**: Custom logic using sentence boundary detection (regex or basic NLP) and the `internal/rag/embedder.go` component.

---

## 2. Embedding Pipeline with OpenRouter

### API Integration
- **Model**: `nvidia/llama-nemotron-embed-vl-1b-v2:free`
- **Endpoint**: `https://openrouter.ai/api/v1/embeddings`
- **SDK**: `github.com/sashabaranov/go-openai`
- **Configuration**:
    ```go
    config := openai.DefaultConfig(apiKey)
    config.BaseURL = "https://openrouter.ai/api/v1"
    client := openai.NewClientWithConfig(config)
    ```

### Batching & Resilience
- **Batch Size**: 20 chunks per request (as per ROADMAP.md).
- **Concurrency**: The worker pool in Phase 4 will handle document-level concurrency, but Phase 3 should ensure the `Embedder` can process chunks efficiently.
- **Retries**: Use exponential backoff for `429` (Rate Limit) and `5xx` (Server Error) from OpenRouter.

---

## 3. Qdrant Storage & Payload Schema

### Go Client Usage
- **Library**: `github.com/qdrant/go-client`
- **Batching**: Use `client.Upsert` with a slice of `*qdrant.PointStruct`.

### Payload Schema
As per `03-CONTEXT.md`:
- `doc_id`: UUID of the parent document.
- `chunk_index`: Sequence number for sorting.
- `text`: Original plaintext segment.
- `strategy`: The chunking strategy name used.

Example Structure:
```go
point := &qdrant.PointStruct{
    Id:      qdrant.NewIDNum(id), // or NewIDString(uuid)
    Vectors: qdrant.NewVectors(embedding...),
    Payload: qdrant.NewValueMap(map[string]any{
        "doc_id":      docID.String(),
        "chunk_index": i,
        "text":         chunkText,
        "strategy":     strategy,
    }),
}
```

### Model Consistency Check
- **Location**: Store the model name (`nvidia/llama-nemotron-embed-vl-1b-v2:free`) in collection metadata/params during initialization.
- **Check**: Compare the configured model name with the one stored in the collection during startup. Throw error on mismatch.

---

## 4. Engineering Concerns & Pitfalls

1. **Token vs Character mismatch**: `langchaingo` uses characters by default. If exact token limits are required, we must wrap it with a tokenizer (e.g., `tiktoken` although it's for OpenAI, it might not perfectly match the NVIDIA model). For v1, a character-based approximation (e.g., `512 * 4`) is safer unless a specific tokenizer is available.
2. **Embedding Latency**: 2048-dim vectors are large. Batching is essential to avoid excessive network overhead.
3. **Qdrant Health**: Ensure collection creation happens with `on_disk_payload: true` if memory becomes an issue on Alpine containers.

## 5. Validation Architecture

### Structural Validation
- `Parser` interface must produce clean output for `Chunker`.
- `Chunker` output MUST total exactly the input text (no lost characters).

### Integration Validation
- `Embedder.Embed(text)` must return exactly 2048 dimensions.
- `Store.Upsert(chunks)` must reflect correctly in `client.Count()`.

## 6. Verification Plan

1. **Unit Test**: `Chunker` strategies with known text samples.
2. **Integration Test**: Mock OpenRouter embedding call returning dummy 2048-dim vector.
3. **Integration Test**: Real Qdrant container check for point insertion and payload retrieval.
